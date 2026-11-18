package drives

import (
	"os"
	"path/filepath"
	"testing"
)

func makeReader(t *testing.T, root, rel string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(rel)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, rel), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFinderRecognizesAPocketBookByItsHomeScreenConfiguration(t *testing.T) {
	base := t.TempDir()
	reader1 := filepath.Join(base, "POCKETBOOK")
	reader2 := filepath.Join(base, "OTHER")
	usb := filepath.Join(base, "PENDRIVE")
	makeReader(t, reader1, viewJSON[0])
	makeReader(t, reader2, viewJSON[1])
	if err := os.MkdirAll(filepath.Join(usb, "system"), 0o755); err != nil {
		t.Fatal(err)
	}
	// a folder named like the file is not a reader
	if err := os.MkdirAll(filepath.Join(base, "DIR", viewJSON[0]), 0o755); err != nil {
		t.Fatal(err)
	}

	found := Finder{Roots: func() []string {
		return []string{reader1, usb, reader2, filepath.Join(base, "DIR"), filepath.Join(base, "none")}
	}}.Find()
	if len(found) != 2 || found[0].Root != reader1 || found[1].Root != reader2 {
		t.Fatalf("%+v", found)
	}
	if found[0].Label != "POCKETBOOK" && filepath.Separator != '\\' {
		t.Errorf("the label is the volume name: %q", found[0].Label)
	}
}

func TestRootsPerOperatingSystem(t *testing.T) {
	none := func(string) ([]string, error) { return nil, nil }
	win := rootsFor("windows", "", none)
	if len(win) != 26 || win[0] != `A:\` || win[25] != `Z:\` {
		t.Errorf("every drive letter: %v", win)
	}

	var asked []string
	spy := func(p string) ([]string, error) { asked = append(asked, p); return []string{"/x/b", "/x/a"}, nil }
	mac := rootsFor("darwin", "", spy)
	if len(asked) != 1 || asked[0] != "/Volumes/*" || len(mac) != 2 || mac[0] != "/x/a" {
		t.Errorf("mac: %v %v", asked, mac)
	}
	asked = nil
	lin := rootsFor("linux", "ana", spy)
	if len(asked) != 6 || len(lin) != 2 {
		t.Errorf("linux: %v %v (duplicates are dropped)", asked, lin)
	}
	asked = nil
	if rootsFor("linux", "", spy); len(asked) != 4 {
		t.Errorf("without a user: %v", asked)
	}
}

func TestTheDefaultFinderRunsOnAnyComputer(t *testing.T) {
	if len(DefaultRoots()) == 0 && !filepathExists("/") {
		t.Log("no candidates on this computer")
	}
	for _, r := range (Finder{}).Find() {
		if !IsReader(r.Root) {
			t.Errorf("%v is not a reader", r)
		}
	}
}

func filepathExists(p string) bool { _, err := os.Stat(p); return err == nil }
