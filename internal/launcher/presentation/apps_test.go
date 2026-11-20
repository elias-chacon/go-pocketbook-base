package presentation

import (
	"errors"
	"image"
	"strings"
	"testing"
	"time"

	"go-pocketbook-base/internal/launcher/application"
	"go-pocketbook-base/internal/launcher/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

type remover struct {
	removed []string
	err     error
}

func (r *remover) Uninstall(id string) error { r.removed = append(r.removed, id); return r.err }

// manageEnv is a home with 2 installed apps (app00, app01) and the catalog knowing app01 and web00.
type manageEnv struct {
	h     *Home
	inst  *installer
	rm    *remover
	saved []domain.Preferences
	sums  map[string]string
	now   time.Time
}

func newManageEnv(t *testing.T) *manageEnv {
	t.Helper()
	e := &manageEnv{inst: &installer{}, rm: &remover{}, sums: map[string]string{"app00": "old", "app01": "same"}, now: time.Unix(1000, 0)}
	cat := catalog{list: []domain.Listing{
		{ID: "app01", Title: "App app01", Version: "2.0.0", AppSHA256: "SAME"},
		{ID: "web00", Title: "Web 00", Version: "1.0.0", AppSHA256: "x"},
		{ID: "app00", Title: "App app00", Version: "2.0.0", AppSHA256: "new"},
	}}
	e.h = NewHome(Config{
		Width: 1404, Height: 1872, Tr: NewTranslator("pt-BR"), Catalog: cat, Installer: e.inst, Uninstaller: e.rm,
		InstalledSum: func(id string) string { return e.sums[id] },
		SavePrefs:    func(p domain.Preferences) error { e.saved = append(e.saved, p); return nil },
		Home:         application.Home{Apps: registry{apps: apps(2)}, Launcher: &launcher{}, SelfID: "launcher"},
		IconFor:      func(domain.Tile) image.Image { return image.NewGray(image.Rect(0, 0, 128, 128)) },
	})
	e.h.Doc.Now = func() time.Time { return e.now }
	e.h.Doc.Layout()
	return e
}

// longPress holds a touch on the tile for two seconds.
func (e *manageEnv) longPress(id string) {
	e.h.Doc.Layout()
	p := find(e.h, id).Elem().Bounds.Center()
	e.h.Doc.HandlePointer(widget.PointerDown, p.X*e.h.Doc.Scale, p.Y*e.h.Doc.Scale)
	e.now = e.now.Add(2 * time.Second)
	e.h.Doc.HandlePointer(widget.PointerUp, p.X*e.h.Doc.Scale, p.Y*e.h.Doc.Scale)
	e.h.Doc.Layout()
}

func TestALongPressOpensTheWindowOfTheAppInsteadOfStartingIt(t *testing.T) {
	e := newManageEnv(t)
	e.longPress("app-app01")
	if find(e.h, "modal-app") == nil || len(e.h.cfg.Home.Launcher.(*launcher).opened) != 0 {
		t.Fatal("a long press manages the app, it does not open it")
	}
	if find(e.h, "manage-hide") == nil || find(e.h, "manage-remove") == nil {
		t.Error("an installed app can be hidden and removed")
	}
}

func TestUpdateIsOfferedOnlyWhenTheChecksumOfTheInstalledFileDiffers(t *testing.T) {
	e := newManageEnv(t)
	e.longPress("app-app01")
	if find(e.h, "manage-update") != nil || !strings.Contains(texts(e.h), "É a build publicada") {
		t.Errorf("the same checksum (any case) is up to date: %s", texts(e.h))
	}
	e.h.Doc.Modal().Close()

	e.longPress("app-app00")
	if find(e.h, "manage-update") == nil {
		t.Fatalf("another checksum can be updated: %s", texts(e.h))
	}
	tap(e.h.Doc, find(e.h, "manage-update"))
	if len(e.inst.installed) != 1 || e.inst.installed[0] != "app00" || e.h.Doc.HasModal() {
		t.Errorf("the update installs the listing: %v", e.inst.installed)
	}
	if !strings.Contains(texts(e.h), "“App app00” atualizado") {
		t.Errorf("%s", texts(e.h))
	}

	e.sums["app00"] = ""
	e.longPress("app-app00")
	if find(e.h, "manage-update") != nil {
		t.Error("an unknown checksum offers nothing")
	}
}

func TestAMissingAppIsInstalledFromItsWindowAndCanBeHiddenToo(t *testing.T) {
	e := newManageEnv(t)
	e.longPress("app-web00")
	if find(e.h, "manage-install") == nil || find(e.h, "manage-remove") != nil || find(e.h, "manage-update") != nil {
		t.Fatal("a missing app: install and hide only")
	}
	tap(e.h.Doc, find(e.h, "manage-hide"))
	if find(e.h, "app-web00") != nil || len(e.saved) != 1 || len(e.saved[0].HiddenApps) != 1 || e.saved[0].HiddenApps[0] != "web00" {
		t.Errorf("hidden and kept: %+v", e.saved)
	}
}

func TestHidingAnInstalledAppTakesItOffTheGridAndTheSettingsShowItAgain(t *testing.T) {
	e := newManageEnv(t)
	e.longPress("app-app01")
	tap(e.h.Doc, find(e.h, "manage-hide"))
	e.h.Doc.Layout()
	if find(e.h, "app-app01") != nil || !strings.Contains(texts(e.h), "está escondido") {
		t.Fatalf("hidden: %s", texts(e.h))
	}
	e.h.OpenHidden()
	e.h.Doc.Layout()
	table := find(e.h, "hidden-table").(*widget.Table)
	if len(table.Rows) != 1 || table.Rows[0][0] != "App app01" {
		t.Fatalf("the hidden list: %v", table.Rows)
	}
	table.Selected = 0
	table.OnSelect(&widget.Event{})
	tap(e.h.Doc, find(e.h, "hidden-show"))
	e.h.Doc.Layout()
	if find(e.h, "app-app01") == nil || len(e.h.cfg.Prefs.HiddenApps) != 0 {
		t.Error("shown again")
	}
	if find(e.h, "modal-hidden-none") == nil {
		t.Error("the hidden list comes back, now empty")
	}
}

func TestRemovingAnAppAsksFirstThenUninstallsIt(t *testing.T) {
	e := newManageEnv(t)
	e.longPress("app-app01")
	tap(e.h.Doc, find(e.h, "manage-remove"))
	if find(e.h, "modal-remove") == nil || len(e.rm.removed) != 0 {
		t.Fatal("it must ask before removing")
	}
	e.h.Doc.Modal().Cancel()
	if len(e.rm.removed) != 0 {
		t.Fatal("cancel removes nothing")
	}
	e.longPress("app-app01")
	tap(e.h.Doc, find(e.h, "manage-remove"))
	e.h.Doc.Modal().Confirm()
	if len(e.rm.removed) != 1 || e.rm.removed[0] != "app01" || !strings.Contains(texts(e.h), "“App app01” removido") {
		t.Errorf("removed %v: %s", e.rm.removed, texts(e.h))
	}
	e.rm.err = errors.New("arquivo em uso")
	e.longPress("app-app00")
	tap(e.h.Doc, find(e.h, "manage-remove"))
	e.h.Doc.Modal().Confirm()
	if !strings.Contains(texts(e.h), "Não foi possível remover “App app00” por completo: arquivo em uso") {
		t.Errorf("a failure is told: %s", texts(e.h))
	}
}

func TestTheSettingsListTheAppsToManageAndKeepTheHiddenOnesWhenSaving(t *testing.T) {
	e := newManageEnv(t)
	e.h.cfg.Prefs = e.h.cfg.Prefs.WithHidden("web00", true)
	e.h.Refresh()
	e.h.OpenSettings()
	e.h.Doc.Layout()
	tap(e.h.Doc, find(e.h, "settings-apps"))
	e.h.Doc.Layout()
	table := find(e.h, "manager-table").(*widget.Table)
	if len(table.Rows) != 2 {
		t.Fatalf("the apps to manage, without the hidden: %v", table.Rows)
	}
	table.Selected = 1
	table.OnSelect(&widget.Event{})
	e.h.Doc.Layout()
	tap(e.h.Doc, find(e.h, "manager-open"))
	if find(e.h, "modal-app") == nil {
		t.Fatal("the button opens the window of the selected app")
	}
	e.h.Doc.Modal().Cancel() // closing it brings the list back
	if find(e.h, "manager-table") == nil {
		t.Error("the list comes back after the window closes")
	}
	e.h.Doc.Modal().Close()
	e.h.Doc.Layout()
	tap(e.h.Doc, find(e.h, "settings-save"))
	last := e.saved[len(e.saved)-1]
	if len(last.HiddenApps) != 1 || last.HiddenApps[0] != "web00" {
		t.Errorf("saving the settings keeps the hidden apps: %+v", last)
	}
}
