package application

import (
	"errors"
	"reflect"
	"testing"

	"go-pocketbook-base/internal/platform/domain"
)

type fakeRemover struct {
	unregistered []string
	err          error
}

func (f *fakeRemover) Unregister(id string) (bool, error) {
	f.unregistered = append(f.unregistered, id)
	return true, f.err
}

type fakeFiles struct {
	id    string
	files []string
	err   error
}

func (f *fakeFiles) Remove(id string, files []string) error { f.id, f.files = id, files; return f.err }

type installedApps []domain.InstalledApp

func (a installedApps) Save(domain.InstalledApp) error { return nil }
func (a installedApps) List() []domain.InstalledApp    { return a }

func app(id string, data, shares []string) domain.InstalledApp {
	return domain.InstalledApp{Manifest: domain.Manifest{ID: id, Data: data, Shares: shares}}
}

func TestUninstallRemovesTheIconTheFilesAndTheDataOfTheApp(t *testing.T) {
	d, f := &fakeRemover{}, &fakeFiles{}
	var erased string
	u := Uninstall{
		Apps:    installedApps{app("bible", []string{"bible.db", "bible.db-journal"}, nil), app("hello", nil, nil)},
		Desktop: d, Files: f, EraseData: func(id string) error { erased = id; return nil },
	}
	if err := u.Run("bible"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(d.unregistered, []string{"bible"}) || f.id != "bible" || erased != "bible" ||
		!reflect.DeepEqual(f.files, []string{"bible.db", "bible.db-journal"}) {
		t.Errorf("desktop=%v files=%q %v erased=%q", d.unregistered, f.id, f.files, erased)
	}
}

func TestASharedFileGoesOnlyWithTheLastAppThatSharesIt(t *testing.T) {
	shared := []string{"accounts.json", "secrets.bin"}
	f := &fakeFiles{}
	u := Uninstall{Apps: installedApps{app("mail", nil, shared), app("tasks", nil, shared)}, Desktop: &fakeRemover{}, Files: f}
	if err := u.Run("mail"); err != nil || len(f.files) != 0 {
		t.Fatalf("tasks still uses the accounts: %v %v", f.files, err)
	}
	u.Apps = installedApps{app("tasks", nil, shared)}
	if err := u.Run("tasks"); err != nil || !reflect.DeepEqual(f.files, shared) {
		t.Errorf("the last one takes the accounts: %v %v", f.files, err)
	}
}

func TestUninstallTriesEveryStepAndReportsTheFailuresTogether(t *testing.T) {
	d, f := &fakeRemover{err: errors.New("view.json busy")}, &fakeFiles{err: errors.New("file locked")}
	u := Uninstall{Apps: installedApps{app("x", nil, nil)}, Desktop: d, Files: f, EraseData: func(string) error { return errors.New("db locked") }}
	err := u.Run("x")
	if err == nil {
		t.Fatal("failures must be reported")
	}
	for _, want := range []string{"view.json busy", "db locked", "file locked"} {
		if !contains(err.Error(), want) {
			t.Errorf("%q is missing from %q", want, err)
		}
	}
	if f.id != "x" {
		t.Error("the files are tried even after the home screen failed")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
