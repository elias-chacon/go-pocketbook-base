package presentation

import (
	"errors"
	"image"
	"reflect"
	"strings"
	"testing"

	"go-pocketbook-base/internal/launcher/application"
	"go-pocketbook-base/internal/launcher/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

type settingsEnv struct {
	savedPrefs []domain.Preferences
	savedLang  []string
	prefsErr   error
	langErr    error
}

func settingsHome(env *settingsEnv, backgrounds []string, load func(string) (image.Image, error)) *Home {
	return NewHome(Config{
		Width: 1404, Height: 1872, Tr: NewTranslator("pt-BR"),
		Home:    application.Home{Apps: registry{apps: apps(30)}, Launcher: &launcher{}, SelfID: "launcher"},
		IconFor: func(domain.Tile) image.Image { return image.NewGray(image.Rect(0, 0, 128, 128)) },
		SavePrefs: func(p domain.Preferences) error {
			env.savedPrefs = append(env.savedPrefs, p)
			return env.prefsErr
		},
		SaveLang: func(l string) error {
			env.savedLang = append(env.savedLang, l)
			return env.langErr
		},
		Backgrounds:    func() []string { return backgrounds },
		LoadBackground: load,
	})
}

func open(h *Home) {
	h.OpenSettings()
	h.Doc.Layout()
}

func TestTheSettingsScreenOpensFromTheFooterAndFitsTheScreen(t *testing.T) {
	h := settingsHome(&settingsEnv{}, []string{"sea.png"}, nil)
	h.Doc.Layout()
	if h.InSettings() {
		t.Fatal("the home shows first")
	}
	tap(h.Doc, find(h, "home-settings"))
	h.Doc.Layout()
	if !h.InSettings() {
		t.Fatal("the footer button opens the settings")
	}

	for _, id := range []string{"settings-columns", "settings-rows", "settings-size", "settings-language", "settings-background", "settings-save", "settings-cancel", "settings-reset"} {
		w := find(h, id)
		if w == nil {
			t.Fatalf("%s is missing", id)
		}
		if b := w.Elem().Bounds; b.Bottom() > h.Doc.H || b.Right() > h.Doc.W+0.01 || b.H < 56 && id != "settings-size" {
			t.Errorf("%s does not fit or is too small to touch: %+v", id, b)
		}
	}
	if !strings.Contains(texts(h), "Configurações") {
		t.Errorf("the title: %s", texts(h))
	}
	h.OpenSettings() // opening twice changes nothing
	if !h.InSettings() {
		t.Error("still in the settings")
	}
}

func TestSavingAppliesTheGridTheIconSizeAndKeepsThePreferences(t *testing.T) {
	env := &settingsEnv{}
	h := settingsHome(env, nil, nil)
	h.Doc.Layout()
	if len(tiles(h)) != 20 {
		t.Fatalf("the default page holds 20: %d", len(tiles(h)))
	}

	open(h)
	find(h, "settings-columns").(*widget.Input).Value = "3"
	find(h, "settings-rows").(*widget.Input).Value = "2"
	find(h, "settings-size").(*widget.Slider).SetValue(180)
	tap(h.Doc, find(h, "settings-save"))
	h.Doc.Layout()

	if h.InSettings() || len(env.savedPrefs) != 1 {
		t.Fatalf("saved and back home: in=%v saved=%v", h.InSettings(), env.savedPrefs)
	}
	want := domain.Preferences{Grid: domain.Grid{Columns: 3, Rows: 2}, IconSize: 180}
	if !reflect.DeepEqual(env.savedPrefs[0], want) {
		t.Errorf("saved %+v, want %+v", env.savedPrefs[0], want)
	}
	if len(tiles(h)) != 6 || h.Pages() != 5 {
		t.Errorf("a 3x2 grid holds 6 tiles per page: %d tiles, %d pages", len(tiles(h)), h.Pages())
	}
	if img := tiles(h)[0].Elem().Children()[0].Elem(); img.Bounds.H < 179 || img.Bounds.H > 181 {
		t.Errorf("the icons are 180 tall: %+v", img.Bounds)
	}
	if len(env.savedLang) != 0 {
		t.Error("the language did not change: it is not saved")
	}
}

func TestChangingTheLanguageIsSavedAndTranslatesTheScreen(t *testing.T) {
	env := &settingsEnv{}
	h := settingsHome(env, nil, nil)
	open(h)
	find(h, "settings-language").(*widget.ComboBox).SetValue("en")
	tap(h.Doc, find(h, "settings-save"))
	h.Doc.Layout()

	if len(env.savedLang) != 1 || env.savedLang[0] != "en" {
		t.Errorf("saved language: %v", env.savedLang)
	}
	if !strings.Contains(texts(h), "My apps") {
		t.Errorf("the screen speaks English now: %s", texts(h))
	}
}

func TestCancelAndTheBackKeyDoNotSaveAnything(t *testing.T) {
	env := &settingsEnv{}
	h := settingsHome(env, nil, nil)
	open(h)
	find(h, "settings-columns").(*widget.Input).Value = "2"
	tap(h.Doc, find(h, "settings-cancel"))
	h.Doc.Layout()
	if h.InSettings() || len(env.savedPrefs) != 0 || len(tiles(h)) != 20 {
		t.Errorf("cancel: in=%v saved=%v tiles=%d", h.InSettings(), env.savedPrefs, len(tiles(h)))
	}

	open(h)
	h.CloseSettings()
	h.CloseSettings() // closing when closed changes nothing
	if h.InSettings() {
		t.Error("CloseSettings goes back")
	}
}

func TestDefaultPutsTheDefaultsInTheFormButSavesNothingYet(t *testing.T) {
	env := &settingsEnv{}
	h := settingsHome(env, []string{"sea.png"}, nil)
	h.cfg.Prefs = domain.Preferences{Grid: domain.Grid{Columns: 2, Rows: 2}, IconSize: 200, Image: "sea.png"}
	open(h)
	tap(h.Doc, find(h, "settings-reset"))

	if find(h, "settings-columns").(*widget.Input).Value != "4" || find(h, "settings-rows").(*widget.Input).Value != "5" ||
		find(h, "settings-size").(*widget.Slider).Value != float64(domain.DefaultIconSize) ||
		find(h, "settings-background").(*widget.ComboBox).Value() != "" {
		t.Error("the form shows the defaults")
	}
	if len(env.savedPrefs) != 0 {
		t.Error("nothing is saved until Save")
	}
}

func TestAnInvalidFormOrAFailingSaveStaysOnTheFormAndSaysWhy(t *testing.T) {
	env := &settingsEnv{}
	h := settingsHome(env, nil, nil)
	open(h)
	find(h, "settings-columns").(*widget.Input).Value = ""
	tap(h.Doc, find(h, "settings-save"))
	if !h.InSettings() || !strings.Contains(find(h, "settings-problem").(*widget.Label).Text, "Digite números") {
		t.Error("an empty field is refused")
	}

	find(h, "settings-columns").(*widget.Input).Value = "4"
	env.prefsErr = errors.New("disco cheio")
	tap(h.Doc, find(h, "settings-save"))
	if !h.InSettings() || !strings.Contains(find(h, "settings-problem").(*widget.Label).Text, "disco cheio") {
		t.Error("a failing store is reported")
	}

	env.prefsErr = nil
	env.langErr = errors.New("sem permissão")
	find(h, "settings-language").(*widget.ComboBox).SetValue("en")
	tap(h.Doc, find(h, "settings-save"))
	if !h.InSettings() || !strings.Contains(find(h, "settings-problem").(*widget.Label).Text, "sem permissão") {
		t.Error("a failing language store is reported")
	}
}

func TestABackgroundPictureIsChosenFromTheFolderAndLoaded(t *testing.T) {
	env := &settingsEnv{}
	loaded := []string{}
	picture := image.NewGray(image.Rect(0, 0, 64, 64))
	h := settingsHome(env, []string{"sea.png", "broken.png"}, func(name string) (image.Image, error) {
		loaded = append(loaded, name)
		if name == "broken.png" {
			return nil, errors.New("not a picture")
		}
		return picture, nil
	})
	if h.root.Style.BackgroundImage != nil {
		t.Error("no background by default")
	}

	open(h)
	find(h, "settings-background").(*widget.ComboBox).SetValue("sea.png")
	tap(h.Doc, find(h, "settings-save"))
	if h.root.Style.BackgroundImage != picture || env.savedPrefs[0].Image != "sea.png" {
		t.Errorf("the picture is the background and the choice is kept: %+v", env.savedPrefs)
	}

	open(h)
	find(h, "settings-background").(*widget.ComboBox).SetValue("broken.png")
	tap(h.Doc, find(h, "settings-save"))
	if h.root.Style.BackgroundImage != nil {
		t.Error("a picture that cannot be read leaves no background")
	}
}

func TestWithoutPicturesTheScreenExplainsWhereToPutThem(t *testing.T) {
	h := settingsHome(&settingsEnv{}, nil, nil)
	open(h)
	if !strings.Contains(texts(h), "applications/backgrounds") {
		t.Errorf("a hint when the folder is empty: %s", texts(h))
	}
	h2 := settingsHome(&settingsEnv{}, []string{"a.png"}, nil)
	open(h2)
	if strings.Contains(texts(h2), "applications/backgrounds") {
		t.Error("no hint when there are pictures")
	}
}

func TestWithoutStoresTheSettingsStillApply(t *testing.T) {
	h := NewHome(Config{
		Width: 1404, Height: 1872, Tr: NewTranslator("pt-BR"),
		Home: application.Home{Apps: registry{apps: apps(5)}, Launcher: &launcher{}, SelfID: "launcher"},
	})
	open(h)
	find(h, "settings-columns").(*widget.Input).Value = "2"
	tap(h.Doc, find(h, "settings-save"))
	h.Doc.Layout()
	if h.InSettings() || h.cfg.Prefs.Grid.Columns != 2 {
		t.Error("applied without being kept")
	}
}
