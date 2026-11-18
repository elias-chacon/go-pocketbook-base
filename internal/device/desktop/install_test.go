package desktop

import (
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func icon(size int, focused bool) image.Image { return image.NewGray(image.Rect(0, 0, size, size)) }

func setup(t *testing.T, view string) (Config, string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "system", "profiles", "default", "config", "desktop")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, "view.json")
	if err := os.WriteFile(p, []byte(view), 0o644); err != nil {
		t.Fatal(err)
	}
	return Config{
		Root: root, Exe: filepath.Join(root, "applications", "hello.app"),
		Title: "Hello Go", Group: "@General", Icon: icon,
	}, p
}

// testdata/view.json is the unmodified file from a PB741 (Era) running 6.x.
func TestRegisterRealFirmwareFile(t *testing.T) {
	orig, err := os.ReadFile("testdata/view.json")
	if err != nil {
		t.Fatal(err)
	}
	c, p := setup(t, string(orig))

	res, err := Register(c)
	if err != nil || !res.Changed || res.Key != "U_hello" {
		t.Fatalf("Register = %+v, %v", res, err)
	}
	data, _ := os.ReadFile(p)
	var v struct {
		Applications map[string]json.RawMessage
		View         struct {
			Groups []struct {
				Title string
				Apps  []string
			}
		}
	}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	var got struct {
		Path string
		Icon struct{ Path string }
	}
	if err := json.Unmarshal(v.Applications["U_hello"], &got); err != nil {
		t.Fatal(err)
	}
	if got.Path != filepath.ToSlash(c.Exe) || got.Icon.Path != "applications/icons/hello.bmp" {
		t.Errorf("entry = %+v", got)
	}
	if len(v.View.Groups) != 3 || v.View.Groups[0].Apps[0] != "U_hello" || len(v.View.Groups[0].Apps) != 16 {
		t.Errorf("groups = %+v", v.View.Groups)
	}
	if b, _ := os.ReadFile(res.Backup); string(b) != string(orig) {
		t.Error("backup differs from original")
	}
	for _, n := range []string{"hello.bmp", "hello_f.bmp"} {
		if _, err := os.Stat(filepath.Join(c.Root, "applications", "icons", n)); err != nil {
			t.Error(err)
		}
	}

	// Idempotent: second run changes nothing.
	res2, err := Register(c)
	if err != nil || res2.Changed {
		t.Fatalf("second Register = %+v, %v", res2, err)
	}
	if after, _ := os.ReadFile(p); string(after) != string(data) {
		t.Error("second run modified view.json")
	}
}

// An earlier version registered a relative launch path, which the desktop
// cannot start. Running again must repair it (and only it).
func TestRegisterRepairsRelativePath(t *testing.T) {
	orig, _ := os.ReadFile("testdata/view.json")
	c, p := setup(t, string(orig))
	if _, err := Register(c); err != nil {
		t.Fatal(err)
	}
	cur, _ := os.ReadFile(p)
	broken := strings.Replace(string(cur), `"path": "`+filepath.ToSlash(c.Exe)+`"`, `"path": "applications/hello.app"`, 1)
	if broken == string(cur) {
		t.Fatal("test setup: path not found")
	}
	if err := os.WriteFile(p, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Register(c)
	if err != nil || !res.Changed || res.Backup == "" {
		t.Fatalf("repair = %+v, %v", res, err)
	}
	fixed, _ := os.ReadFile(p)
	if string(fixed) != string(cur) {
		t.Errorf("repaired file differs from a fresh registration:\n%s", fixed)
	}
	if res2, err := Register(c); err != nil || res2.Changed {
		t.Fatalf("third run = %+v, %v", res2, err)
	}
}

func TestRegisterEmptyContainers(t *testing.T) {
	c, p := setup(t, `{"applications": {}, "view": {"groups": [{"title": "@General", "apps": []}]}}`)
	if _, err := Register(c); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(p)
	if !json.Valid(data) || !strings.Contains(string(data), `"U_hello"`) {
		t.Errorf("bad result: %s", data)
	}
}

func TestRegisterMissingGroupWritesNothing(t *testing.T) {
	in := `{"applications": {}, "view": {"groups": [{"title": "@Other", "apps": []}]}}`
	c, p := setup(t, in)
	if _, err := Register(c); err == nil {
		t.Fatal("expected error")
	}
	if data, _ := os.ReadFile(p); string(data) != in {
		t.Error("view.json was modified despite the error")
	}
}
