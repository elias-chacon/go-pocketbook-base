package main

import (
	"os"
	"testing"

	"go-pocketbook-base/internal/accounts/infrastructure/oauthconfig"
	"go-pocketbook-base/internal/platform"
	"go-pocketbook-base/internal/platform/infrastructure/jsonfiles"
	"go-pocketbook-base/internal/toolkit/widget"
)

func testEnv(t *testing.T, lang string) (platform.Env, string) {
	t.Helper()
	m, err := platform.ParseManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	return platform.Env{
		Width: 1404, Height: 1872, Manifest: m, Lang: lang, Version: "1.2.3",
		Settings: jsonfiles.NewSettings(root),
		Keyboard: &widget.FakeKeyboard{Reply: "x"}, Exit: func() {},
	}, root
}

func TestTheManifestFollowsTheContract(t *testing.T) {
	m, err := platform.ParseManifest(manifest)
	if err != nil || m.ID != "hello" {
		t.Fatalf("%+v %v", m, err)
	}
	if m.TitleIn("pt-BR") != "Hello Go" || m.DescriptionIn("en") == "" {
		t.Errorf("title=%q description=%q", m.TitleIn("pt-BR"), m.DescriptionIn("en"))
	}
}

func TestHelloIsAnAppWithKeysAndTheLanguageIsSharedThroughTheSettings(t *testing.T) {
	env, root := testEnv(t, "en")
	app := newHello(env)
	if app.Document() == nil {
		t.Fatal("the app must give the platform a document")
	}
	h := app.(*hello)

	var _ platform.KeyHandler = h
	h.Key(platform.KeyNext)
	h.Key(platform.KeyNext)
	h.Key(platform.KeyPrevious)
	if got := h.demo.Pages(); got[0] != "Inputs" {
		t.Errorf("the demo starts in the language from the settings: %v", got)
	}
	if h.Key(platform.KeyOther) {
		t.Error("a key the app does not use is not handled")
	}

	m := widget.NewModal("Hi", widget.NewLabel("x"))
	m.Open(h.demo.Doc)
	if h.Key(platform.KeyNext) {
		t.Error("with a modal open the right key does nothing")
	}
	m.Close()

	// switching the language in the app saves it for every app
	h.demo.SetLang("pt-BR")
	if got := env.Settings.Load().Lang; got != "pt-BR" {
		t.Errorf("saved language = %q", got)
	}
	h.demo.SetLang("en")
	if got := env.Settings.Load().Lang; got != "en" {
		t.Errorf("saved language = %q", got)
	}
	if _, err := os.Stat(jsonfiles.SettingsPath(root)); err != nil {
		t.Error(err)
	}
}

func TestTheAccountPageUsesTheRealSignInsWhenTheClientIDsExist(t *testing.T) {
	old := oauthconfig.Default
	oauthconfig.Default = oauthconfig.Config{} // nothing built in: only the file sets a provider
	t.Cleanup(func() { oauthconfig.Default = old })
	env, _ := testEnv(t, "en")
	dir := t.TempDir()
	for _, p := range oauthProviders(env, dir, func() string { return "" }) {
		if p.Start != nil {
			t.Errorf("%s must be disabled without client ids", p.ID)
		}
	}
	if err := os.WriteFile(dir+"/oauth.json", []byte(`{"microsoft":"m","google":"g"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got := oauthProviders(env, dir, func() string { return "" })
	if len(got) != 1 || got[0].Label != "Microsoft" || got[0].Start == nil {
		t.Errorf("only Microsoft is offered, enabled by its client id (Google is not offered in the apps): %+v", got)
	}
}
