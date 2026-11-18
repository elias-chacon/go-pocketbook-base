package readerfs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"go-pocketbook-base/internal/installer/domain"
)

func TestDeployWritesTheAppThenRegistersIt(t *testing.T) {
	root := t.TempDir()
	var registered []string
	d := Deployer{Register: func(root string, p domain.Package) (bool, error) {
		registered = append(registered, p.ID)
		return true, nil
	}}
	pkg := domain.Package{ID: "hello", Binary: []byte("v1")}

	changed, err := d.Deploy(domain.Reader{Root: root}, pkg)
	if err != nil || !changed || len(registered) != 1 {
		t.Fatalf("%v %v %v", changed, err, registered)
	}
	got, _ := os.ReadFile(filepath.Join(root, "applications", "hello.app"))
	if string(got) != "v1" {
		t.Errorf("content: %q", got)
	}

	pkg.Binary = []byte("v2-longer")
	if _, err := d.Deploy(domain.Reader{Root: root}, pkg); err != nil {
		t.Fatal(err)
	}
	got, _ = os.ReadFile(filepath.Join(root, "applications", "hello.app"))
	if string(got) != "v2-longer" {
		t.Errorf("an update replaces the file: %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, "applications", "hello.app.tmp")); err == nil {
		t.Error("no temporary file is left")
	}
}

func TestDeployReportsWhatWentWrong(t *testing.T) {
	boom := errors.New("no icon")
	d := Deployer{Register: func(string, domain.Package) (bool, error) { return false, boom }}
	if _, err := d.Deploy(domain.Reader{Root: t.TempDir()}, domain.Package{ID: "x"}); err != boom {
		t.Errorf("a registration error is returned: %v", err)
	}

	// the root is a file: the applications folder cannot be created
	file := filepath.Join(t.TempDir(), "file")
	os.WriteFile(file, nil, 0o644)
	if _, err := d.Deploy(domain.Reader{Root: file}, domain.Package{ID: "x"}); err == nil {
		t.Error("a reader that cannot be written to is an error")
	}

	// the target is a folder: it cannot be replaced
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "applications", "x.app", "sub"), 0o755)
	if _, err := d.Deploy(domain.Reader{Root: root}, domain.Package{ID: "x"}); err == nil {
		t.Error("an app that cannot be replaced is an error")
	}
	if _, err := os.Stat(filepath.Join(root, "applications", "x.app.tmp")); err == nil {
		t.Error("the temporary file is removed on failure")
	}
}
