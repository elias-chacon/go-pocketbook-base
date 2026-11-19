package prefsfile

import (
	"os"
	"path/filepath"
	"testing"

	"go-pocketbook-base/internal/launcher/domain"
)

func TestPreferencesRoundTripAndAreNormalizedOnTheWay(t *testing.T) {
	s := New(t.TempDir())
	if got := s.Load(); got != domain.DefaultPreferences() {
		t.Errorf("nothing saved: %+v", got)
	}
	want := domain.Preferences{Grid: domain.Grid{Columns: 3, Rows: 4}, IconSize: 180, Image: "sea.png"}
	if err := s.Save(want); err != nil {
		t.Fatal(err)
	}
	if got := s.Load(); got != want {
		t.Errorf("%+v, want %+v", got, want)
	}
	if err := s.Save(domain.Preferences{IconSize: 5000}); err != nil {
		t.Fatal(err)
	}
	if got := s.Load(); got.IconSize != domain.MaxIconSize {
		t.Errorf("saved within bounds: %+v", got)
	}
	if _, err := os.Stat(s.Path + ".tmp"); err == nil {
		t.Error("no temporary file is left")
	}
}

func TestABrokenFileGivesTheDefaults(t *testing.T) {
	s := New(t.TempDir())
	os.MkdirAll(filepath.Dir(s.Path), 0o755)
	os.WriteFile(s.Path, []byte("{not json"), 0o644)
	if got := s.Load(); got != domain.DefaultPreferences() {
		t.Errorf("%+v", got)
	}
}

func TestSaveReportsWhatCannotBeWritten(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	os.WriteFile(file, nil, 0o644)
	if err := New(file).Save(domain.DefaultPreferences()); err == nil {
		t.Error("a storage that cannot be written")
	}
	s := New(t.TempDir())
	os.MkdirAll(s.Path, 0o755) // the target is a folder: it cannot be replaced
	if err := s.Save(domain.DefaultPreferences()); err == nil {
		t.Error("a target that cannot be replaced")
	}
	if _, err := os.Stat(s.Path + ".tmp"); err == nil {
		t.Error("the temporary file is removed on failure")
	}
}
