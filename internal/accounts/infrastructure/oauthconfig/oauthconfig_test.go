package oauthconfig

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReadsTheClientIds(t *testing.T) {
	old := Default
	Default = Config{} // nothing built in: only the file sets a provider
	t.Cleanup(func() { Default = old })
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
	old := Default
	Default = Config{} // nothing built in: only the file sets a provider
	t.Cleanup(func() { Default = old })
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

func TestTheMicrosoftIdIsBuiltInAndTheFileOnlyOverridesIt(t *testing.T) {
	dir := t.TempDir()
	if Default.Microsoft == "" || Load(dir).Microsoft != Default.Microsoft {
		t.Fatalf("nothing to set up: %+v", Load(dir))
	}
	if ms, g := Sources(dir, http.DefaultClient); ms == nil || g != nil {
		t.Errorf("Microsoft works out of the box, Google needs its own ids: %v %v", ms, g)
	}
	os.WriteFile(filepath.Join(dir, File), []byte(`{"microsoft":"mine","google":"g","google_secret":"s"}`), 0o600)
	if got := Load(dir); got.Microsoft != "mine" || got.Google != "g" || got.GoogleSecret != "s" {
		t.Errorf("the file overrides: %+v", got)
	}
	os.WriteFile(filepath.Join(dir, File), []byte(`{"microsoft":"  "}`), 0o600)
	if Load(dir).Microsoft != Default.Microsoft {
		t.Error("an empty id keeps the built-in one")
	}
}
