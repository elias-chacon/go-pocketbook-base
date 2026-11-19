package main

import (
	"bytes"
	"errors"
	"go-pocketbook-base/internal/launcher/infrastructure/iconcache"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/bmp"

	"go-pocketbook-base/internal/launcher/domain"
	"go-pocketbook-base/internal/platform"
	platformdomain "go-pocketbook-base/internal/platform/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

type registry struct{ apps []platform.InstalledApp }

func (r registry) Save(platform.InstalledApp) error { return nil }
func (r registry) List() []platform.InstalledApp    { return r.apps }

func installed(id, title string) platform.InstalledApp {
	return platform.InstalledApp{
		Manifest: platform.Manifest{ID: id, Title: map[string]string{"pt-BR": title}, Description: map[string]string{"pt-BR": "d"}},
		Exe:      "/apps/" + id + ".app",
	}
}

func testEnv(t *testing.T, apps ...platform.InstalledApp) platform.Env {
	t.Helper()
	m, err := platform.ParseManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return platform.Env{
		Width: 1404, Height: 1872, Manifest: m, Lang: "pt-BR", Version: "1.2.3",
		Apps: registry{apps: apps}, Keyboard: &widget.FakeKeyboard{},
	}
}

func TestTheManifestFollowsTheContract(t *testing.T) {
	m, err := platform.ParseManifest(manifest)
	if err != nil || m.ID != "launcher" {
		t.Fatalf("%+v %v", m, err)
	}
}

func TestTheLauncherListsTheOtherAppsAndPagesWithTheKeys(t *testing.T) {
	apps := []platform.InstalledApp{installed("launcher", "Launcher")}
	for i := 0; i < 25; i++ { // more than one default page (20)
		apps = append(apps, installed(string(rune('a'+i)), "App "+string(rune('A'+i))))
	}
	app := newLauncher(testEnv(t, apps...))
	doc := app.Document()
	doc.Layout()

	count := func() int {
		n := 0
		doc.Walk(func(w widget.Widget) bool {
			if id := w.Elem().ID; len(id) > 4 && id[:4] == "app-" {
				n++
			}
			return true
		})
		return n
	}
	if count() != 20 {
		t.Fatalf("the launcher does not list itself, and a page holds 20: %d", count())
	}

	var _ platform.KeyHandler = app.(*launcher)
	if !app.(*launcher).Key(platform.KeyNext) {
		t.Error("the right key turns the page")
	}
	doc.Layout()
	if count() != 5 {
		t.Errorf("the second page holds the other 5: %d", count())
	}
	if !app.(*launcher).Key(platform.KeyPrevious) || app.(*launcher).Key(platform.KeyOther) {
		t.Error("left turns back; any other key is not ours")
	}
}

func TestIconsComeFromTheBMPTheAppWroteOrFallBackToTheDefaultOne(t *testing.T) {
	// a real BMP, as desktop.Register writes them
	src := image.NewGray(image.Rect(0, 0, 128, 128))
	for i := range src.Pix {
		src.Pix[i] = 40
	}
	var buf bytes.Buffer
	if err := bmp.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "hello.bmp")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	got := iconFor(domain.Tile{Title: "Hello", IconPath: path})
	if got.Bounds().Dx() != 128 {
		t.Errorf("the icon on disk is used: %v", got.Bounds())
	}
	if g, ok := got.(*image.Gray); ok && g.GrayAt(5, 5).Y != 40 {
		t.Errorf("pixel = %d", g.GrayAt(5, 5).Y)
	}
	for _, missing := range []string{"", filepath.Join(t.TempDir(), "none.bmp")} {
		fallback := iconFor(domain.Tile{Title: "Livros", IconPath: missing})
		if fallback == nil || fallback.Bounds().Dx() != 128 {
			t.Errorf("a missing icon (%q) falls back to the default one", missing)
		}
	}
}

// The launcher registers itself like any app and knows its way back for the apps it starts.
func TestNewLauncherUsesItsOwnIdSoItDoesNotListItself(t *testing.T) {
	env := testEnv(t, installed("launcher", "Launcher"))
	app := newLauncher(env).(*launcher)
	var _ platformdomain.AppRegistry = env.Apps
	app.home.Doc.Layout()
	found := false
	app.home.Doc.Walk(func(w widget.Widget) bool {
		found = found || w.Elem().ID == "home-empty"
		return true
	})
	if !found {
		t.Error("with only itself installed the home is empty")
	}
}

type stubCatalog struct {
	list []domain.Listing
	err  error
}

func (c stubCatalog) List(string) ([]domain.Listing, error) { return c.list, c.err }

type stubInstaller struct{ err error }

func (i stubInstaller) Install(domain.Listing) error { return i.err }

func TestTheNetworkWrappersPassTheAnswersThrough(t *testing.T) {
	boom := errors.New("offline")
	list := []domain.Listing{{ID: "x"}}
	if got, err := (&loggedCatalog{inner: stubCatalog{list: list}}).List("en"); err != nil || len(got) != 1 {
		t.Errorf("%v %v", got, err)
	}
	if _, err := (&loggedCatalog{inner: stubCatalog{err: boom}}).List("en"); err != boom {
		t.Errorf("the error is returned (and logged): %v", err)
	}
	if err := (loggedInstaller{inner: stubInstaller{}}).Install(list[0]); err != nil {
		t.Error(err)
	}
	if err := (loggedInstaller{inner: stubInstaller{err: boom}}).Install(list[0]); err != boom {
		t.Errorf("%v", err)
	}
}

func TestTheRegistrarUsesThePublishedIconOrTheDefaultOne(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewGray(image.Rect(0, 0, 32, 32))); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/missing.png" {
			http.NotFound(w, r)
			return
		}
		w.Write(buf.Bytes())
	}))
	defer srv.Close()
	reg := iconRegistrar{icons: iconcache.Cache{Client: srv.Client(), Dir: t.TempDir()}}

	icon := reg.iconOf(domain.Listing{ID: "a", Version: "1", IconURL: srv.URL + "/a.png"})
	if icon == nil || icon(64, false).Bounds().Dx() != 64 {
		t.Error("the published icon becomes the home screen icon")
	}
	for _, l := range []domain.Listing{{ID: "b"}, {ID: "c", Version: "1", IconURL: srv.URL + "/missing.png"}} {
		if reg.iconOf(l) != nil {
			t.Errorf("%s: without a usable icon the default one is used", l.ID)
		}
	}
}

func TestTheMenuKeyOpensAndClosesTheSettingsAndLeftLeavesThem(t *testing.T) {
	app := newLauncher(testEnv(t, installed("hello", "Hello"))).(*launcher)
	if !app.Key(platform.KeyMenu) || !app.home.InSettings() {
		t.Fatal("Menu opens the settings")
	}
	if !app.Key(platform.KeyNext) || !app.home.InSettings() {
		t.Error("right does nothing in the settings, but the key is ours")
	}
	if app.Key(platform.KeyOther) {
		t.Error("another key is not ours")
	}
	if !app.Key(platform.KeyPrevious) || app.home.InSettings() {
		t.Error("left leaves the settings")
	}
	app.Key(platform.KeyMenu)
	if !app.Key(platform.KeyMenu) || app.home.InSettings() {
		t.Error("Menu closes them too")
	}
}

type memorySettings struct{ s platformdomain.Settings }

func (m *memorySettings) Load() platformdomain.Settings        { return m.s }
func (m *memorySettings) Save(s platformdomain.Settings) error { m.s = s; return nil }

func TestTheLanguageIsKeptInTheSettingsEveryAppShares(t *testing.T) {
	store := &memorySettings{s: platformdomain.DefaultSettings()}
	env := testEnv(t)
	env.Settings = store
	if err := languageSaver(env)("en"); err != nil || store.s.Lang != "en" {
		t.Errorf("%+v %v", store.s, err)
	}
	env.Settings = nil
	if err := languageSaver(env)("en"); err != nil {
		t.Errorf("without a store there is nothing to keep: %v", err)
	}
}

func TestABackgroundIsShrunkToTheScreenAndAMissingOneIsAnError(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	png.Encode(&buf, image.NewGray(image.Rect(0, 0, 2000, 1000)))
	os.WriteFile(filepath.Join(dir, "sea.png"), buf.Bytes(), 0o644)

	load := backgroundLoader(dir, 700, 900)
	img, err := load("sea.png")
	if err != nil || img.Bounds().Dx() > 700 || img.Bounds().Dy() > 900 {
		t.Errorf("shrunk to the screen: %v %v", img, err)
	}
	if _, err := load("missing.png"); err == nil {
		t.Error("a missing picture")
	}
}
