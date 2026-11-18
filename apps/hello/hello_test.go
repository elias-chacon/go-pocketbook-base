package main

import (
	"os"
	"testing"

	"go-pocketbook-base/internal/appkit"
	"go-pocketbook-base/internal/inkview"
	"go-pocketbook-base/internal/widget"
)

func testEnv(t *testing.T, lang string) appkit.Env {
	t.Helper()
	m, err := appkit.ParseManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return appkit.Env{
		Width: 1404, Height: 1872, Manifest: m, Lang: lang, Version: "1.2.3", Root: t.TempDir(),
		Keyboard: &widget.FakeKeyboard{Reply: "x"}, Exit: func() {},
	}
}

func TestTheManifestFollowsTheContract(t *testing.T) {
	m, err := appkit.ParseManifest(manifest)
	if err != nil || m.ID != "hello" {
		t.Fatalf("%+v %v", m, err)
	}
	if m.TitleIn("pt-BR") != "Hello Go" || m.DescriptionIn("en") == "" {
		t.Errorf("title=%q description=%q", m.TitleIn("pt-BR"), m.DescriptionIn("en"))
	}
}

func TestHelloIsAnAppWithKeysAndTheLanguageIsSharedThroughTheSettings(t *testing.T) {
	env := testEnv(t, "en")
	app := newHello(env)
	if app.Document() == nil {
		t.Fatal("the app must give the kit a document")
	}
	h := app.(*hello)

	var _ appkit.KeyHandler = h
	h.Key(inkview.KeyNext)
	h.Key(inkview.KeyNext)
	h.Key(inkview.KeyPrev)
	if got := h.demo.Pages(); got[0] != "Inputs" {
		t.Errorf("the demo starts in the language from the settings: %v", got)
	}
	if h.Key(0x41) {
		t.Error("a key the app does not use is not handled")
	}

	m := widget.NewModal("Hi", widget.NewLabel("x"))
	m.Open(h.demo.Doc)
	if h.Key(inkview.KeyNext) {
		t.Error("with a modal open the right key does nothing")
	}
	m.Close()

	// switching the language in the app saves it for every app
	h.demo.SetLang("pt-BR")
	if got := appkit.LoadSettings(env.Root).Lang; got != "pt-BR" {
		t.Errorf("saved language = %q", got)
	}
	h.demo.SetLang("en")
	if got := appkit.LoadSettings(env.Root).Lang; got != "en" {
		t.Errorf("saved language = %q", got)
	}
	if _, err := os.Stat(appkit.SettingsPath(env.Root)); err != nil {
		t.Error(err)
	}
}
