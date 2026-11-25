package jsonfiles

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"go-pocketbook-base/internal/platform/domain"
)

func manifest(id string) domain.Manifest {
	return domain.Manifest{
		ID:          id,
		Title:       map[string]string{"pt-BR": "Título", "en": "Title"},
		Description: map[string]string{"pt-BR": "d", "en": "d"},
	}
}

// Both stores satisfy the domain's ports.
var (
	_ domain.AppRegistry   = (*Apps)(nil)
	_ domain.SettingsStore = (*Settings)(nil)
)

func TestAppRecordsRoundTripAndSkipWhatIsBroken(t *testing.T) {
	root := t.TempDir()
	apps := filepath.Join(root, "applications")
	if err := os.MkdirAll(apps, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := func(name string) string {
		p := filepath.Join(apps, name+".app")
		if err := os.WriteFile(p, []byte("elf"), 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	reg := NewApps(root)
	for _, in := range []domain.InstalledApp{
		{Manifest: manifest("hello"), Version: "1.2.3", Exe: exe("hello"), Icon: "/x/hello.bmp"},
		{Manifest: manifest("alpha"), Version: "0.1.0", Exe: exe("alpha")},
		{Manifest: manifest("gone"), Exe: filepath.Join(apps, "gone.app")}, // .app deleted
	} {
		if err := reg.Save(in); err != nil {
			t.Fatal(err)
		}
	}
	// junk that must not hide the good ones
	meta := metaDir(root)
	os.WriteFile(filepath.Join(meta, "broken.json"), []byte("{nope"), 0o644)
	os.WriteFile(filepath.Join(meta, "invalid.json"), []byte(`{"id":"UPPER"}`), 0o644)
	os.WriteFile(filepath.Join(meta, "notes.txt"), []byte("x"), 0o644)
	if err := NewSettings(root).Save(domain.Settings{Lang: "en"}); err != nil { // not an app record
		t.Fatal(err)
	}

	got := reg.List()
	var ids []string
	for _, in := range got {
		ids = append(ids, in.ID)
	}
	if !reflect.DeepEqual(ids, []string{"alpha", "hello"}) {
		t.Fatalf("installed = %v (sorted, only the valid ones with an .app)", ids)
	}
	if got[1].Version != "1.2.3" || got[1].Icon != "/x/hello.bmp" || got[1].TitleIn("en") != "Title" {
		t.Errorf("record = %+v", got[1])
	}
	if err := reg.Save(domain.InstalledApp{Manifest: domain.Manifest{ID: "BAD"}}); err == nil {
		t.Error("an invalid manifest must not be saved")
	}
	if len(NewApps(filepath.Join(root, "nowhere")).List()) != 0 {
		t.Error("no folder, no apps")
	}
	if _, err := os.Stat(RecordPath(root, "hello")); err != nil {
		t.Errorf("the record is where RecordPath says: %v", err)
	}
}

func TestSettingsDefaultsAndPersistence(t *testing.T) {
	root := t.TempDir()
	store := NewSettings(root)
	if s := store.Load(); s.Lang != "pt-BR" {
		t.Errorf("default language = %q", s.Lang)
	}
	if err := store.Save(domain.Settings{Lang: "en_US"}); err != nil {
		t.Fatal(err)
	}
	if s := store.Load(); s.Lang != "en-US" {
		t.Errorf("a saved tag is normalized: %q", s.Lang)
	}
	os.WriteFile(SettingsPath(root), []byte("{broken"), 0o644)
	if s := store.Load(); s.Lang != "pt-BR" {
		t.Errorf("a broken file gives the defaults: %q", s.Lang)
	}
	os.WriteFile(SettingsPath(root), []byte(`{"lang": ""}`), 0o644)
	if s := store.Load(); s.Lang != "pt-BR" {
		t.Errorf("an empty language gives the default: %q", s.Lang)
	}
}

func TestUpdateStateIsRememberedAndABrokenFileIsForgotten(t *testing.T) {
	root := t.TempDir()
	s := NewUpdateState(root, "hello")
	if got := s.Load(); got.Dismissed != "" || !got.CheckedAt.IsZero() {
		t.Fatalf("nothing remembered yet: %+v", got)
	}
	if err := s.Save(domain.UpdateState{Dismissed: "1.2.3", Available: domain.Update{Version: "1.2.3", URL: "u"}}); err != nil {
		t.Fatal(err)
	}
	if got := s.Load(); got.Dismissed != "1.2.3" || got.Available.URL != "u" {
		t.Errorf("state = %+v", got)
	}
	os.WriteFile(UpdatePath(root, "hello"), []byte("{broken"), 0o644)
	if got := s.Load(); got.Dismissed != "" {
		t.Errorf("a broken file must give an empty state: %+v", got)
	}
	if apps := NewApps(root).List(); len(apps) != 0 {
		t.Errorf("the update file is not an installed app: %+v", apps)
	}
}

func TestEachAppRemembersItsOwnUpdateChecks(t *testing.T) {
	root := t.TempDir()
	hello, tasks := NewUpdateState(root, "hello"), NewUpdateState(root, "tasks")
	if err := hello.Save(domain.UpdateState{Dismissed: "aaa", CheckedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if got := tasks.Load(); got.Dismissed != "" || !got.CheckedAt.IsZero() {
		t.Errorf("another app's check must not count for tasks: %+v", got)
	}
	if err := tasks.Save(domain.UpdateState{Dismissed: "bbb"}); err != nil {
		t.Fatal(err)
	}
	if got := hello.Load(); got.Dismissed != "aaa" {
		t.Errorf("hello lost its own state: %+v", got)
	}
	if err := (Files{root: root}).Remove("hello", nil); err != nil {
		t.Fatal(err)
	}
	if got := hello.Load(); got.Dismissed != "" {
		t.Errorf("removing an app removes its update file: %+v", got)
	}
	if got := tasks.Load(); got.Dismissed != "bbb" {
		t.Errorf("removing hello must keep tasks: %+v", got)
	}
}

func TestFilesRemoveTheAppItsRecordItsLogAndItsDataButNeverOutsideTheFolder(t *testing.T) {
	root := t.TempDir()
	write := func(rel string) string {
		p := filepath.Join(root, filepath.FromSlash(rel))
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte("x"), 0o644)
		return p
	}
	app, log, rec := write("applications/bible.app"), write("applications/bible.log"), write("applications/.pbgo/bible.json")
	db, other := write("applications/.pbgo/bible.db"), write("applications/.pbgo/tasks.json")
	outside := write("applications/keep.txt")
	if err := NewFiles(root).Remove("bible", []string{"bible.db", "missing.db", "../keep.txt"}); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{app, log, rec, db} {
		if _, err := os.Stat(p); err == nil {
			t.Errorf("%s must be gone", p)
		}
	}
	for _, p := range []string{other, outside} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%s must stay", p)
		}
	}
}

func TestFilesRemoveWhatIsNamedAfterTheAppEvenWhenTheRecordListsNothing(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "applications", ".pbgo")
	os.MkdirAll(dir, 0o755)
	for _, n := range []string{"bible.db", "bible.db-journal", "bible-x_y.json", "bibleX.json", "accounts.json", "biblical.db"} {
		os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644)
	}
	if err := NewFiles(root).Remove("bible", nil); err != nil {
		t.Fatal(err)
	}
	for n, gone := range map[string]bool{"bible.db": true, "bible.db-journal": true, "bible-x_y.json": true, "bibleX.json": false, "accounts.json": false, "biblical.db": false} {
		_, err := os.Stat(filepath.Join(dir, n))
		if gone != (err != nil) {
			t.Errorf("%s: gone=%v, want %v", n, err != nil, gone)
		}
	}
}
