package desktop

import (
	"encoding/json"
	"image"
	"os"
	"path/filepath"
	"regexp"
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

// An install replaces what an older version registered: title, icons and path come from the new
// registration, the position in the file and every other app stay, and doing it again changes nothing.
func TestInstallReplacesAnOlderEntryAndItsIcons(t *testing.T) {
	orig, _ := os.ReadFile("testdata/view.json")
	c, p := setup(t, string(orig))
	if _, err := Register(c); err != nil {
		t.Fatal(err)
	}
	cur, _ := os.ReadFile(p)
	old := strings.Replace(string(cur), `"title": "Hello Go"`, `"title": "Old name"`, 1)
	old = strings.Replace(old, `"path": "applications/icons/hello.bmp"`, `"path": "applications/icons/gone.bmp"`, 1)
	if old == string(cur) {
		t.Fatal("test setup: nothing to age")
	}
	os.WriteFile(p, []byte(old), 0o644)
	iconFile := filepath.Join(c.Root, "applications", "icons", "hello.bmp")
	os.WriteFile(iconFile, []byte("an old icon"), 0o644)

	if res, err := Register(c); err != nil || res.Changed {
		t.Fatalf("a plain registration keeps what is listed: %+v %v", res, err)
	}
	c.Replace = true
	res, err := Register(c)
	if err != nil || !res.Changed {
		t.Fatalf("an install must replace the entry: %+v %v", res, err)
	}
	now, _ := os.ReadFile(p)
	if string(now) != string(cur) {
		t.Errorf("the entry must be what a fresh registration writes, in the same place:\n%s", now)
	}
	if data, _ := os.ReadFile(iconFile); string(data) == "an old icon" {
		t.Error("the icon file must be written again")
	}
	if res, err := Register(c); err != nil || res.Changed {
		t.Errorf("installing twice changes nothing more: %+v %v", res, err)
	}
}

func TestInstallPutsBackAnAppThatLostItsPlaceInTheGroup(t *testing.T) {
	orig, _ := os.ReadFile("testdata/view.json")
	c, p := setup(t, string(orig))
	if _, err := Register(c); err != nil {
		t.Fatal(err)
	}
	cur, _ := os.ReadFile(p)
	lost := regexp.MustCompile(`\s*"U_hello"\s*,`).ReplaceAllString(string(cur), "")
	if strings.Count(lost, `"U_hello"`) != 1 { // only the entry key is left
		t.Fatalf("test setup: %d keys", strings.Count(lost, `"U_hello"`))
	}
	os.WriteFile(p, []byte(lost), 0o644)
	c.Replace = true
	if _, err := Register(c); err != nil {
		t.Fatal(err)
	}
	now, _ := os.ReadFile(p)
	if strings.Count(string(now), `"U_hello"`) != 2 {
		t.Errorf("the app must be in the group again:\n%s", now)
	}
	var v interface{}
	if err := json.Unmarshal(now, &v); err != nil {
		t.Errorf("invalid json: %v", err)
	}
}
