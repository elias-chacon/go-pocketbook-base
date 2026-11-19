package embedded

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestPackagesPairEachAppWithItsManifestAndSkipTheRest(t *testing.T) {
	fsys := fstest.MapFS{
		"zeta.app":     {Data: []byte("Z")},
		"zeta.json":    {Data: []byte(`{"id":"zeta"}`)},
		"alpha.app":    {Data: []byte("A")},
		"alpha.json":   {Data: []byte(`{"id":"alpha"}`)},
		"VERSION":      {Data: []byte("1.2.3\n")},
		"README.txt":   {Data: []byte("not an app")},
		"sub/deep.app": {Data: []byte("ignored: in a folder")},
	}
	pkgs, err := Source{FS: fsys}.Packages()
	if err != nil || len(pkgs) != 2 {
		t.Fatalf("%v %v", pkgs, err)
	}
	if pkgs[0].ID != "alpha" || pkgs[1].ID != "zeta" || pkgs[0].Version != "1.2.3" {
		t.Errorf("sorted by id, with the version: %+v", pkgs)
	}
	if string(pkgs[0].Binary) != "A" || string(pkgs[0].Manifest) != `{"id":"alpha"}` {
		t.Errorf("content: %+v", pkgs[0])
	}
}

func TestAnAppWithoutAManifestIsAnError(t *testing.T) {
	if _, err := (Source{FS: fstest.MapFS{"x.app": {Data: []byte("x")}}}).Packages(); err == nil {
		t.Error("the manifest is required")
	}
}

func TestAnEmptyOrUnreadableFolder(t *testing.T) {
	if pkgs, err := (Source{FS: fstest.MapFS{}}).Packages(); err != nil || len(pkgs) != 0 {
		t.Errorf("an empty folder is just no packages: %v %v", pkgs, err)
	}
	if _, err := (Source{FS: failing{}}).Packages(); err == nil {
		t.Error("an unreadable folder is an error")
	}
}

type failing struct{}

func (failing) Open(string) (fs.File, error) { return nil, errors.New("boom") }

func TestAnIconNextToTheAppIsCarriedAndItIsOptional(t *testing.T) {
	fsys := fstest.MapFS{
		"a.app": {Data: []byte("A")}, "a.json": {Data: []byte("{}")}, "a.png": {Data: []byte("PNG")},
		"b.app": {Data: []byte("B")}, "b.json": {Data: []byte("{}")},
	}
	pkgs, err := Source{FS: fsys}.Packages()
	if err != nil || len(pkgs) != 2 {
		t.Fatalf("%v %v", pkgs, err)
	}
	if string(pkgs[0].Icon) != "PNG" || pkgs[1].Icon != nil {
		t.Errorf("icons: %q %q", pkgs[0].Icon, pkgs[1].Icon)
	}
}
