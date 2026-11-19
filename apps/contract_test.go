// Package apps has no code: it holds the test that every folder in ./apps
// follows the app contract (see internal/platform and apps/hello, the blueprint).
package apps

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go-pocketbook-base/internal/localization/i18n"
	"go-pocketbook-base/internal/platform/domain"
)

// appDirs lists the app folders (every sub-folder of ./apps).
func appDirs(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	return dirs
}

func TestHelloIsTheBlueprintAndEveryAppFollowsTheContract(t *testing.T) {
	dirs := appDirs(t)
	hasHello := false
	for _, dir := range dirs {
		hasHello = hasHello || dir == "hello"
		t.Run(dir, func(t *testing.T) { checkApp(t, dir) })
	}
	if !hasHello {
		t.Error("apps/hello is the blueprint and must exist")
	}
}

func checkApp(t *testing.T, dir string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "app.json"))
	if err != nil {
		t.Fatalf("an app needs an app.json: %v", err)
	}
	m, err := domain.ParseManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != dir {
		t.Errorf("the id %q must be the folder name %q (it names %s.app)", m.ID, dir, dir)
	}

	main, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("an app needs a main.go: %v", err)
	}
	for _, want := range []string{"package main", "go:embed app.json", "platform.Run"} {
		if !strings.Contains(string(main), want) {
			t.Errorf("main.go must contain %q (the contract: embed the manifest and run the platform)", want)
		}
	}

	// An app that keeps its own translations must keep both languages complete.
	if _, err := os.Stat(filepath.Join(dir, "locales")); err == nil {
		tr := i18n.New()
		if err := tr.AddFS(os.DirFS(dir), "locales"); err != nil {
			t.Fatal(err)
		}
		for _, lang := range []string{i18n.Primary, i18n.Fallback} {
			if missing := tr.Missing(lang); len(missing) > 0 {
				t.Errorf("locales/%s.json lacks: %v", lang, missing)
			}
		}
	}
}

// An app may ship icon.png: the release publishes it so the Launcher can show the
// app before it is installed, and it is the app's icon on the home screen.
func TestAnIconPNGIsA128x128Picture(t *testing.T) {
	for _, dir := range appDirs(t) {
		f, err := os.Open(filepath.Join(dir, "icon.png"))
		if os.IsNotExist(err) {
			continue // optional: the platform draws a default icon
		}
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := png.DecodeConfig(f)
		f.Close()
		if err != nil {
			t.Errorf("%s/icon.png is not a PNG: %v", dir, err)
		} else if cfg.Width != 128 || cfg.Height != 128 {
			t.Errorf("%s/icon.png is %dx%d, it must be 128x128", dir, cfg.Width, cfg.Height)
		}
	}
}

// Every app is a plugin of the Launcher that also installs without it: it is one
// <id>.app with its own app.json, registers itself, and follows the Hello Go blueprint,
// whose standard top bar has the app name with About and Exit (back to the Launcher or
// the home screen). Hello draws its own menu (it is the model); every other app must
// build the standard bar with platform.Chrome and put it above its screens.
func TestEveryAppFollowsTheHelloBlueprint(t *testing.T) {
	for _, dir := range appDirs(t) {
		if dir == "hello" || dir == "launcher" {
			continue // the model itself, and the host the others plug into
		}
		t.Run(dir, func(t *testing.T) {
			files, err := filepath.Glob(filepath.Join(dir, "*.go"))
			if err != nil {
				t.Fatal(err)
			}
			var source strings.Builder
			for _, f := range files {
				if strings.HasSuffix(f, "_test.go") {
					continue
				}
				data, err := os.ReadFile(f)
				if err != nil {
					t.Fatal(err)
				}
				source.Write(data)
			}
			if !strings.Contains(source.String(), "platform.Chrome(") {
				t.Errorf("apps/%s must show the standard top bar (About and Exit): pass platform.Chrome(env) to its screens, as apps/mail does", dir)
			}
		})
	}
}

// The home screen only accepts an app in a group its view.json already has: an unknown
// group makes the icon registration fail and the Launcher never lists the app (this
// happened with "@Productivity"). @General is the one every firmware has.
func TestEveryAppIsInAGroupTheHomeScreenHas(t *testing.T) {
	for _, dir := range appDirs(t) {
		data, err := os.ReadFile(filepath.Join(dir, "app.json"))
		if err != nil {
			t.Fatal(err)
		}
		m, err := domain.ParseManifest(data)
		if err != nil {
			t.Fatal(err)
		}
		if m.Group != "@General" {
			t.Errorf("apps/%s is in the group %q: use @General, the home screen has no other group we can count on", dir, m.Group)
		}
	}
}
