package jsonaccounts

import (
	"os"
	"path/filepath"
	"testing"

	"go-pocketbook-base/internal/accounts/domain"
)

func account(name, email string) domain.Account {
	p, _ := domain.PresetFor(email)
	return p.Account(name, email)
}

func TestAccountsAreKeptListedByNameAndReplacedById(t *testing.T) {
	s := New(t.TempDir())
	if list, err := s.List(); err != nil || len(list) != 0 {
		t.Fatalf("nothing yet: %v %v", list, err)
	}
	if err := s.Save(account("Zeta", "z@icloud.com")); err != nil {
		t.Fatal(err)
	}
	s.Save(account("Alfa", "a@outlook.com"))

	list, _ := s.List()
	if len(list) != 2 || list[0].Name != "Alfa" || list[1].Name != "Zeta" {
		t.Errorf("by name: %v", list)
	}
	got, err := s.Get("alfa")
	if err != nil || got.Email != "a@outlook.com" || !got.Has(domain.Mail) {
		t.Errorf("%+v %v", got, err)
	}

	changed := got
	changed.Name = "Alfa 2"
	s.Save(changed)
	if list, _ := s.List(); len(list) != 2 || list[0].Name != "Alfa 2" {
		t.Errorf("the same id is replaced, not duplicated: %v", list)
	}
	if _, err := New(filepath.Dir(s.Path)).Get("zeta"); err != nil {
		t.Errorf("a new run reads them back: %v", err)
	}
}

func TestDeleteAndNotFound(t *testing.T) {
	s := New(t.TempDir())
	s.Save(account("A", "a@icloud.com"))
	s.Save(account("B", "b@icloud.com"))
	if err := s.Delete("a"); err != nil {
		t.Fatal(err)
	}
	if list, _ := s.List(); len(list) != 1 || list[0].ID != "b" {
		t.Errorf("%v", list)
	}
	if err := s.Delete("a"); err != domain.ErrNotFound {
		t.Errorf("%v", err)
	}
	if _, err := s.Get("a"); err != domain.ErrNotFound {
		t.Errorf("%v", err)
	}
}

func TestABrokenFileAndAnUnwritableFolderAreErrors(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	os.WriteFile(s.Path, []byte("{broken"), 0o600)
	if _, err := s.List(); err == nil {
		t.Error("a broken file")
	}
	if _, err := s.Get("a"); err == nil {
		t.Error("a broken file on Get")
	}
	if err := s.Save(account("A", "a@icloud.com")); err == nil {
		t.Error("a broken file on Save")
	}
	if err := s.Delete("a"); err == nil {
		t.Error("a broken file on Delete")
	}

	file := filepath.Join(t.TempDir(), "file")
	os.WriteFile(file, nil, 0o644)
	if err := New(filepath.Join(file, "sub")).Save(account("A", "a@icloud.com")); err == nil {
		t.Error("a folder under a file cannot be made")
	}
	bad := New(t.TempDir())
	os.MkdirAll(bad.Path, 0o755) // the target is a folder
	if _, err := bad.List(); err == nil {
		t.Error("a folder where the file should be")
	}
	if err := bad.Save(account("A", "a@icloud.com")); err == nil {
		t.Error("cannot read the target")
	}
	ok := New(t.TempDir())
	ok.Save(account("A", "a@icloud.com"))
	os.Remove(ok.Path)
	os.MkdirAll(filepath.Join(ok.Path, "x"), 0o755)
	if err := ok.Save(account("B", "b@icloud.com")); err == nil {
		t.Error("a target that cannot be replaced")
	}
}
