package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

const sampleICS = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//t//EN\r\nBEGIN:VEVENT\r\nUID:a1@t\r\nDTSTAMP:20260501T000000Z\r\nDTSTART:20260520T120000Z\r\nDTEND:20260520T130000Z\r\nSUMMARY:Lunch\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"

func TestTheImporterReadsAFileAndAnAddressWithoutDuplicating(t *testing.T) {
	dir := t.TempDir()
	a := newCalendarAt(testEnv(t), dir, "")
	t.Cleanup(a.close)
	imp, ok := a.screen.Importer().(icsImporter)
	if !ok {
		t.Fatal("the app has no importer")
	}
	file := filepath.Join(dir, "sub", "plan.ics")
	if err := os.MkdirAll(dir+"/sub", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(sampleICS), 0o644); err != nil {
		t.Fatal(err)
	}
	imp.roots = []string{dir}
	if got := imp.Files(); len(got) != 1 || got[0] != file {
		t.Fatalf("files = %v", got)
	}
	for i := 0; i < 2; i++ { // twice: no duplicate
		if n, _, err := imp.Import(file); err != nil || n != 1 {
			t.Fatalf("import = %d, %v", n, err)
		}
	}
	from, to := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	if got, _ := imp.agenda.Store.Between(from, to); len(got) != 1 {
		t.Fatalf("%d events after importing the same file twice", len(got))
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(sampleICS))
	}))
	defer srv.Close()
	imp.web = srv.Client()
	if n, _, err := imp.Import(srv.URL + "/cal.ics"); err != nil || n != 1 {
		t.Fatalf("address import = %d, %v", n, err)
	}
	if _, _, err := imp.Import(srv.URL + "/missing"); err == nil {
		t.Fatal("a 404 must be an error")
	}
	if _, _, err := imp.Import(dir + "/nope.ics"); err == nil {
		t.Fatal("a missing file must be an error")
	}
	if _, _, err := imp.Import("webcal://127.0.0.1:1/x"); err == nil {
		t.Fatal("an unreachable address must be an error")
	}
}
