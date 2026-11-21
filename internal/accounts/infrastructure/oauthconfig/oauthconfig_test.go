package oauthconfig

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsTheClientIds(t *testing.T) {
	dir := t.TempDir()
	if got := Load(dir); got != (Config{}) {
		t.Fatalf("no file: %+v", got)
	}
	write := func(s string) {
		if err := os.WriteFile(filepath.Join(dir, File), []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(`{"microsoft":" m ","google":"g","google_secret":"s"}`)
	if got := Load(dir); got.Microsoft != "m" || got.Google != "g" || got.GoogleSecret != "s" {
		t.Fatalf("got %+v", got)
	}
	write(`not json`)
	if got := Load(dir); got != (Config{}) {
		t.Fatalf("broken file: %+v", got)
	}
}

func TestSourcesBuildsOnlyTheProvidersThatHaveAClientID(t *testing.T) {
	dir := t.TempDir()
	if ms, g := Sources(dir, http.DefaultClient); ms != nil || g != nil {
		t.Error("no file, no providers")
	}
	if err := os.WriteFile(filepath.Join(dir, File), []byte(`{"microsoft":"abc"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if ms, g := Sources(dir, http.DefaultClient); ms == nil || g != nil {
		t.Errorf("microsoft only: %v %v", ms, g)
	}
	if err := os.WriteFile(filepath.Join(dir, File), []byte(`{"microsoft":"abc","google":"g","google_secret":"s"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if ms, g := Sources(dir, http.DefaultClient); ms == nil || g == nil {
		t.Errorf("both: %v %v", ms, g)
	}
}
