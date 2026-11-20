package desktop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type viewFile struct {
	Applications map[string]json.RawMessage
	View         struct {
		Groups []struct {
			Title string
			Apps  []string
		}
	}
}

func readView(t *testing.T, p string) viewFile {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	var v viewFile
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("view.json is not valid: %v", err)
	}
	return v
}

func TestUnregisterTakesTheAppOutOfTheRealFirmwareFileAndKeepsTheOthers(t *testing.T) {
	orig, err := os.ReadFile("testdata/view.json")
	if err != nil {
		t.Fatal(err)
	}
	c, p := setup(t, string(orig))
	before := readView(t, p)
	for _, name := range []string{"hello", "tasks", "mail"} {
		c.Exe = filepath.Join(c.Root, "applications", name+".app")
		if _, err := Register(c); err != nil {
			t.Fatal(err)
		}
	}

	changed, err := Unregister(c.Root, "tasks")
	if err != nil || !changed {
		t.Fatalf("Unregister = %v, %v", changed, err)
	}
	checkOnlyTasksIsGone(t, readView(t, p))
	if changed, err := Unregister(c.Root, "tasks"); err != nil || changed {
		t.Errorf("again: %v %v", changed, err)
	}

	for _, name := range []string{"hello", "mail"} {
		if _, err := Unregister(c.Root, name); err != nil {
			t.Fatal(err)
		}
	}
	after := readView(t, p)
	if !reflect.DeepEqual(before.Applications, after.Applications) || !reflect.DeepEqual(before.View, after.View) {
		t.Error("registering and unregistering everything gives back the original content")
	}
}

func TestUnregisterDeletesTheIconsAndBacksUpTheFile(t *testing.T) {
	c, p := setup(t, `{"applications": {}, "view": {"groups": [{"title": "@General", "apps": []}]}}`)
	if _, err := Register(c); err != nil {
		t.Fatal(err)
	}
	icons := filepath.Join(c.Root, "applications", "icons")
	if _, err := os.Stat(filepath.Join(icons, "hello.bmp")); err != nil {
		t.Fatal("Register writes the icon")
	}
	if _, err := Unregister(c.Root, "hello"); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"hello.bmp", "hello_f.bmp"} {
		if _, err := os.Stat(filepath.Join(icons, f)); err == nil {
			t.Errorf("%s must be deleted", f)
		}
	}
	backups, _ := filepath.Glob(p + ".bak-*")
	if len(backups) < 1 {
		t.Errorf("the change is backed up: %v", backups)
	}
	if text, _ := os.ReadFile(p); strings.Contains(string(text), "U_hello") {
		t.Error("nothing of the app remains in view.json")
	}
}

// checkOnlyTasksIsGone: the entry and its place in the groups are gone, the other apps stay.
func checkOnlyTasksIsGone(t *testing.T, v viewFile) {
	t.Helper()
	if _, ok := v.Applications["U_tasks"]; ok {
		t.Error("the entry must be gone")
	}
	for _, key := range []string{"U_hello", "U_mail"} {
		if _, ok := v.Applications[key]; !ok {
			t.Errorf("%s must stay", key)
		}
	}
	for _, g := range v.View.Groups {
		for _, a := range g.Apps {
			if a == "U_tasks" {
				t.Error("it must leave the group's apps")
			}
		}
	}
}
