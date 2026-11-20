package presentation

import (
	"errors"
	"fmt"
	"image"
	"strings"
	"testing"
	"time"

	"go-pocketbook-base/internal/launcher/application"
	"go-pocketbook-base/internal/launcher/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

type catalog struct {
	list []domain.Listing
	err  error
}

func (c catalog) List(string) ([]domain.Listing, error) { return c.list, c.err }

func searchHome(n int, cat domain.Catalog) *Home {
	return NewHome(Config{
		Width: 1404, Height: 1872, Tr: NewTranslator("pt-BR"), Catalog: cat,
		Home:    application.Home{Apps: registry{apps: apps(n)}, Launcher: &launcher{}, SelfID: "launcher"},
		IconFor: func(domain.Tile) image.Image { return image.NewGray(image.Rect(0, 0, 128, 128)) },
	})
}

func find(h *Home, id string) widget.Widget {
	var found widget.Widget
	h.Doc.Walk(func(w widget.Widget) bool {
		if w.Elem().ID == id {
			found = w
		}
		return true
	})
	return found
}

func texts(h *Home) string {
	var sb strings.Builder
	collect := func(w widget.Widget) bool {
		if l, ok := w.(*widget.Label); ok {
			sb.WriteString(l.Text + "|")
		}
		return true
	}
	h.Doc.Walk(collect)
	return sb.String()
}

func listings(n int) []domain.Listing {
	out := make([]domain.Listing, n)
	for i := range out {
		out[i] = domain.Listing{ID: fmt.Sprintf("web%02d", i), Title: fmt.Sprintf("Web %02d", i), Version: "1.0.0", AppSHA256: "new"}
	}
	return out
}

func TestTheHomeShowsTheAppsOfTheCatalogThatAreNotInstalledToo(t *testing.T) {
	h := searchHome(3, catalog{list: listings(2)})
	h.Doc.Layout()
	if len(tiles(h)) != 5 || find(h, "app-web00") == nil || find(h, "app-app01") == nil {
		t.Fatalf("installed and missing apps share the grid: %d tiles", len(tiles(h)))
	}
	if !strings.Contains(texts(h), "Toque para instalar") {
		t.Error("a missing app says that a tap installs it")
	}
}

func TestTheCatalogFailingOrMissingLeavesTheInstalledApps(t *testing.T) {
	for name, cat := range map[string]domain.Catalog{"offline": catalog{err: errors.New("sem rede")}, "none": nil} {
		h := searchHome(2, cat)
		h.Doc.Layout()
		if len(tiles(h)) != 2 {
			t.Errorf("%s: %d tiles", name, len(tiles(h)))
		}
	}
}

func TestSearchFiltersTheWholeGridInstalledAndMissing(t *testing.T) {
	h := searchHome(5, catalog{list: listings(2)})
	h.Search("  web  ")
	h.Doc.Layout()
	if h.Query() != "web" || len(tiles(h)) != 2 {
		t.Fatalf("query %q, %d tiles", h.Query(), len(tiles(h)))
	}
	h.Search("")
	h.Doc.Layout()
	if len(tiles(h)) != 7 {
		t.Errorf("clearing the query brings every app back: %d", len(tiles(h)))
	}
	in := find(h, "home-search").(*widget.Input)
	in.Value = "app03"
	widget.Fire(in.OnChange, in)
	h.Doc.Layout()
	if len(tiles(h)) != 1 || find(h, "app-app03") == nil {
		t.Errorf("the field filters when its text changes: %d", len(tiles(h)))
	}
}

func TestTheCatalogIsFetchedInTheBackgroundAndShownWhenItArrives(t *testing.T) {
	var later []func()
	h := NewHome(Config{
		Width: 1404, Height: 1872, Tr: NewTranslator("pt-BR"), Catalog: catalog{list: listings(2)},
		CachedCatalog: func(string) []domain.Listing { return listings(1) },
		After:         func(_ time.Duration, f func()) { later = append(later, f) },
		Home:          application.Home{Apps: registry{apps: apps(1)}, Launcher: &launcher{}, SelfID: "launcher"},
		IconFor:       func(domain.Tile) image.Image { return image.NewGray(image.Rect(0, 0, 128, 128)) },
	})
	h.Doc.Layout()
	if len(tiles(h)) != 2 {
		t.Fatalf("the kept catalog shows at once: %d tiles", len(tiles(h)))
	}
	for i := 0; i < 200 && len(tiles(h)) != 3; i++ {
		fs := later
		later = nil
		for _, f := range fs {
			f()
		}
		time.Sleep(5 * time.Millisecond)
	}
	h.Doc.Layout()
	if len(tiles(h)) != 3 {
		t.Errorf("the answer of the network is shown: %d tiles", len(tiles(h)))
	}
}

type installer struct {
	err       error
	installed []string
	onInstall func()
}

func (i *installer) Install(l domain.Listing) error {
	i.installed = append(i.installed, l.ID)
	if i.onInstall != nil {
		i.onInstall()
	}
	return i.err
}

func installHome(inst *installer, flushed *int) (*Home, *registry) {
	reg := &registry{apps: apps(1)}
	h := NewHome(Config{
		Width: 1404, Height: 1872, Tr: NewTranslator("pt-BR"),
		Catalog:   catalog{list: listings(2)},
		Installer: inst,
		Flush:     func() { *flushed++ },
		Home:      application.Home{Apps: reg, Launcher: &launcher{}, SelfID: "launcher"},
		IconFor:   func(domain.Tile) image.Image { return image.NewGray(image.Rect(0, 0, 128, 128)) },
	})
	return h, reg
}

func TestTappingAMissingAppDownloadsAndInstallsIt(t *testing.T) {
	flushed := 0
	inst := &installer{}
	h, reg := installHome(inst, &flushed)
	inst.onInstall = func() {
		if !strings.Contains(texts(h), "Baixando “Web 00”…") {
			t.Error("the screen says it is downloading before the download blocks")
		}
		reg.apps = append(reg.apps, apps(1)[0]) // what registering does: the app appears in the registry
		reg.apps[1].ID = "web00"
		reg.apps[1].Title = map[string]string{"pt-BR": "Web 00"}
	}
	h.Doc.Layout()
	tap(h.Doc, find(h, "app-web00"))
	if len(inst.installed) != 1 || inst.installed[0] != "web00" || flushed != 1 {
		t.Fatalf("installed %v, flushed %d", inst.installed, flushed)
	}
	h.Doc.Layout()
	if !strings.Contains(texts(h), "“Web 00” instalado") {
		t.Errorf("the result is told: %s", texts(h))
	}
	if h.all[1].ID != "web00" && h.all[2].ID != "web00" {
		t.Error("web00 is in the list")
	}
	for _, tl := range h.all {
		if tl.ID == "web00" && !tl.Installed {
			t.Error("web00 is installed now")
		}
	}
}

func TestAFailedInstallIsTold(t *testing.T) {
	flushed := 0
	h, _ := installHome(&installer{err: errors.New("sem rede")}, &flushed)
	h.Doc.Layout()
	tap(h.Doc, find(h, "app-web01"))
	h.Doc.Layout()
	if !strings.Contains(texts(h), "Não foi possível instalar “Web 01”: sem rede") {
		t.Errorf("%s", texts(h))
	}
}

func TestAMissingAppShowsTheIconOfTheCatalogWhenThereIsOne(t *testing.T) {
	asked := map[string]bool{}
	h := NewHome(Config{
		Width: 1404, Height: 1872, Tr: NewTranslator("pt-BR"), Catalog: catalog{list: listings(2)},
		ListingIcon: func(l domain.Listing) image.Image {
			asked[l.ID] = true
			return image.NewGray(image.Rect(0, 0, 128, 128))
		},
		Home: application.Home{Apps: registry{}, Launcher: &launcher{}, SelfID: "launcher"},
	})
	h.Doc.Layout()
	if !asked["web00"] || !asked["web01"] {
		t.Errorf("the icons come from the catalog cache: %v", asked)
	}
}
