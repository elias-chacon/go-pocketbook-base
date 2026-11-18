package platform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go-pocketbook-base/internal/device/inkview"
)

const manifest = `{
  "id": "hello",
  "title": {"pt-BR": "Olá Go", "en": "Hello Go"},
  "description": {"pt-BR": "Vitrine", "en": "Showcase"}
}`

// A minimal desktop config in the shape the desktop package expects.
const viewJSON = `{
    "applications": {
    },
    "groups": [
        { "title": "@General", "apps": [
        ] }
    ]
}`

func readerRoot(t *testing.T) (root, view string) {
	t.Helper()
	root = t.TempDir()
	dir := filepath.Join(root, "system", "profiles", "default", "config", "desktop")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	view = filepath.Join(dir, "view.json")
	if err := os.WriteFile(view, []byte(viewJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	return root, view
}

func TestRunRejectsABrokenManifestBeforeTouchingTheDevice(t *testing.T) {
	if err := Run(Config{Manifest: []byte(`{"id": "BAD"}`)}, nil); err == nil {
		t.Error("an invalid manifest must stop the app at start")
	}
}

// Run wires every layer: on the desktop the reader is a stub, so it registers the
// app (domain + application + infrastructure) and then reports it is not on a device.
func TestRunRegistersTheAppRecordsItAndStartsTheReader(t *testing.T) {
	root, view := readerRoot(t)
	err := Run(Config{Manifest: []byte(manifest), Root: root}, func(Env) App { return nil })
	if err != inkview.ErrNotOnDevice {
		t.Fatalf("off-device Run = %v", err)
	}

	data, _ := os.ReadFile(view)
	if !strings.Contains(string(data), "\"U_") || !strings.Contains(string(data), "Olá Go") {
		t.Errorf("the home screen entry uses the translated title: %s", data)
	}
	// Icons are named after the executable (<id>.app in the real world; here, the test binary).
	icons, _ := filepath.Glob(filepath.Join(root, "applications", "icons", "*.bmp"))
	if len(icons) != 2 {
		t.Errorf("the normal and the focused icon are written: %v", icons)
	}

	// The record points at this test binary (os.Executable), so it counts as installed.
	rec, err := os.ReadFile(filepath.Join(root, "applications", ".pbgo", "hello.json"))
	if err != nil || !strings.Contains(string(rec), `"id": "hello"`) || !strings.Contains(string(rec), `"version"`) {
		t.Errorf("record = %s (%v)", rec, err)
	}

	// The language comes from the shared settings.
	os.WriteFile(filepath.Join(root, "applications", ".pbgo", "settings.json"), []byte(`{"lang":"en"}`), 0o644)
	Run(Config{Manifest: []byte(manifest), Root: root}, func(Env) App { return nil })
	if m, _ := ParseManifest([]byte(manifest)); m.TitleIn("en") != "Hello Go" || DefaultIcon("x") == nil {
		t.Error("the facade re-exports the domain")
	}
}
