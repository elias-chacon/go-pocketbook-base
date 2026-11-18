// Package apps has no code: it holds the test that every folder in ./apps
// follows the app contract (see internal/platform and apps/hello, the blueprint).
package apps

import (
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
