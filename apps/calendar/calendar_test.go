package main

import (
	"os"
	"strings"
	"testing"

	"go-pocketbook-base/internal/platform"
	"go-pocketbook-base/internal/platform/infrastructure/jsonfiles"
	"go-pocketbook-base/internal/toolkit/widget"
)

func testEnv(t *testing.T) platform.Env {
	t.Helper()
	m, err := platform.ParseManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return platform.Env{
		Width: 1404, Height: 1872, Manifest: m, Lang: "pt-BR", Version: "1.0.0",
		Settings: jsonfiles.NewSettings(t.TempDir()), Keyboard: &widget.FakeKeyboard{Reply: "x"}, Exit: func() {},
	}
}

func TestTheManifestFollowsTheContract(t *testing.T) {
	m, err := platform.ParseManifest(manifest)
	if err != nil || m.ID != "calendar" {
		t.Fatalf("%+v %v", m, err)
	}
	if m.TitleIn("pt-BR") != "Calendário" || m.TitleIn("en") != "Calendar" || m.DescriptionIn("en") == "" {
		t.Errorf("title = %q / %q", m.TitleIn("pt-BR"), m.TitleIn("en"))
	}
}

func labels(d *widget.Document) string {
	var b strings.Builder
	d.Walk(func(w widget.Widget) bool {
		if l, ok := w.(*widget.Label); ok {
			b.WriteString(l.Text + "\n")
		}
		return true
	})
	return b.String()
}

func TestTheAppReopensOverTheSameDatabase(t *testing.T) {
	dir := t.TempDir()
	a := newCalendarAt(testEnv(t), dir, "America/Sao_Paulo")
	if a.screen == nil {
		t.Fatalf("no screen:\n%s", labels(a.Document()))
	}
	a.close()

	b := newCalendarAt(testEnv(t), dir, "America/Sao_Paulo")
	t.Cleanup(b.close)
	b.Document().Layout()
	if b.screen == nil || !strings.Contains(labels(b.Document()), "2026") {
		t.Errorf("did not reopen:\n%s", labels(b.Document()))
	}
	if !b.Key(platform.KeyNext) || !b.Key(platform.KeyPrevious) || b.Key(platform.KeyMenu) {
		t.Error("the page keys must be handled and the menu key must not")
	}
}

func TestAnUnusableDataFolderShowsTheError(t *testing.T) {
	file := t.TempDir() + "/not-a-folder"
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := newCalendarAt(testEnv(t), file+"/sub", "")
	a.Document().Layout()
	if a.screen != nil || labels(a.Document()) == "" || a.Key(platform.KeyNext) {
		t.Errorf("expected an error text, got %q", labels(a.Document()))
	}
}

func TestZoneNameReadsTheEnvironment(t *testing.T) {
	t.Setenv("TZ", ":America/Sao_Paulo")
	if got := zoneName(); got != "America/Sao_Paulo" {
		t.Errorf("zone = %q", got)
	}
}
