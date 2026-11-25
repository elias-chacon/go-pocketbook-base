package desktop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOrphanEntriesAreTakenOutAndTheOnesWithAProgramStay(t *testing.T) {
	orig, _ := os.ReadFile("testdata/view.json")
	c, p := setup(t, string(orig))
	if _, err := Register(c); err != nil { // hello: its program is not there
		t.Fatal(err)
	}
	kept := c
	kept.Exe, kept.Title = filepath.Join(c.Root, "applications", "kept.app"), "Kept"
	if _, err := Register(kept); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(kept.Exe, []byte("program"), 0o755); err != nil {
		t.Fatal(err)
	}

	var told []string
	gone, err := RemoveOrphans(c.Root, func(id string) { told = append(told, id) })
	if err != nil || len(gone) != 1 || gone[0] != "hello" || len(told) != 1 || told[0] != "hello" {
		t.Fatalf("removed %v, %v", gone, err)
	}
	data, _ := os.ReadFile(p)
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if strings.Contains(string(data), `"U_hello"`) || strings.Count(string(data), `"U_kept"`) != 2 {
		t.Errorf("hello goes (entry and group), kept stays:\n%s", data)
	}
	if _, err := os.Stat(filepath.Join(c.Root, "applications", "icons", "hello.bmp")); err == nil {
		t.Error("its icons go too")
	}
	if _, err := os.Stat(filepath.Join(c.Root, "applications", "icons", "kept.bmp")); err != nil {
		t.Error("the icons of an app that is there stay")
	}
	backups, _ := filepath.Glob(p + ".bak-*")
	if len(backups) == 0 {
		t.Error("view.json is backed up before it changes")
	}
}

func TestNothingIsWrittenWhenThereAreNoOrphans(t *testing.T) {
	orig, _ := os.ReadFile("testdata/view.json")
	c, p := setup(t, string(orig))
	before, _ := os.ReadFile(p)
	gone, err := RemoveOrphans(c.Root, nil)
	if err != nil || len(gone) != 0 {
		t.Fatalf("%v %v", gone, err)
	}
	after, _ := os.ReadFile(p)
	if string(before) != string(after) {
		t.Error("view.json must be left as it was")
	}
	if backups, _ := filepath.Glob(p + ".bak-*"); len(backups) != 0 {
		t.Errorf("no backup for no change: %v", backups)
	}
}

func TestThePathsOfTheReaderAreFoundUnderTheDrive(t *testing.T) {
	if got := onDisk("/drive", "/mnt/ext1/applications/x.app"); filepath.ToSlash(got) != "/drive/applications/x.app" {
		t.Errorf("onDisk = %q", got)
	}
}
