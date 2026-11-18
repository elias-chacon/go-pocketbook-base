package presentation

import (
	"errors"
	"fmt"
	"image"
	"strings"
	"testing"

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
	h.Doc.Walk(func(w widget.Widget) bool {
		if l, ok := w.(*widget.Label); ok {
			sb.WriteString(l.Text + "|")
		}
		return true
	})
	return sb.String()
}

func listings(n int) []domain.Listing {
	out := make([]domain.Listing, n)
	for i := range out {
		out[i] = domain.Listing{ID: fmt.Sprintf("web%02d", i), Title: fmt.Sprintf("Web %02d", i), Version: "1.0.0"}
	}
	return out
}

func TestWithoutAQueryThereAreNoSections(t *testing.T) {
	h := searchHome(3, catalog{list: listings(2)})
	h.Doc.Layout()
	if find(h, "search.installed") != nil || find(h, "search.available") != nil || len(tiles(h)) != 3 {
		t.Error("the plain home is the grid of installed apps only")
	}
	if h.Query() != "" {
		t.Error("no query")
	}
}

func TestSearchingShowsInstalledAndAvailableSections(t *testing.T) {
	h := searchHome(5, catalog{list: append(listings(2), domain.Listing{ID: "app01", Title: "App app01"})})
	h.Search("  app0  ")
	h.Doc.Layout()
	if h.Query() != "app0" || len(tiles(h)) != 5 {
		t.Fatalf("query %q, %d tiles", h.Query(), len(tiles(h)))
	}
	if find(h, "search.installed") == nil || find(h, "search.available") == nil {
		t.Fatal("both sections are shown")
	}
	if find(h, "avail-web00") != nil {
		t.Error("web00 does not match the query")
	}

	h.Search("web")
	h.Doc.Layout()
	if len(tiles(h)) != 0 || find(h, "avail-web00") == nil || find(h, "avail-web01") == nil {
		t.Error("only the catalog matches: no installed tiles, two rows")
	}
	if !strings.Contains(texts(h), "Nenhum resultado para “web”") {
		t.Errorf("an empty installed section says so: %s", texts(h))
	}
	if find(h, "avail-app01") != nil {
		t.Error("an installed app is not listed as available")
	}

	h.Search("")
	h.Doc.Layout()
	if find(h, "search.available") != nil || len(tiles(h)) != 5 {
		t.Error("clearing the query brings the plain home back")
	}
}

func TestTheSearchFieldFiltersWhenItsTextChanges(t *testing.T) {
	h := searchHome(5, nil)
	h.Doc.Layout()
	in := find(h, "home-search").(*widget.Input)
	in.Value = "app03"
	widget.Fire(in.OnChange, in)
	h.Doc.Layout()
	if len(tiles(h)) != 1 || find(h, "app-app03") == nil {
		t.Errorf("one tile matches: %d", len(tiles(h)))
	}
	if find(h, "search-nocatalog") == nil {
		t.Error("without a catalog the available section explains it")
	}
}

func TestTheCatalogFailingIsShownAndTheInstalledStillAnswer(t *testing.T) {
	h := searchHome(2, catalog{err: errors.New("sem rede")})
	h.Search("app")
	h.Doc.Layout()
	if len(tiles(h)) != 2 || !strings.Contains(texts(h), "Catálogo indisponível: sem rede") {
		t.Errorf("%d tiles; %s", len(tiles(h)), texts(h))
	}
}

func TestOnlyAFewAvailableAppsAreListedAndTheRestIsCounted(t *testing.T) {
	h := searchHome(1, catalog{list: listings(7)})
	h.Search("web")
	h.Doc.Layout()
	rows := 0
	h.Doc.Walk(func(w widget.Widget) bool {
		if strings.HasPrefix(w.Elem().ID, "avail-") {
			rows++
		}
		return true
	})
	if rows != availableMax || !strings.Contains(texts(h), "… e mais 3") {
		t.Errorf("%d rows; %s", rows, texts(h))
	}
}

func TestASearchPageFitsTheScreenAndKeepsPaging(t *testing.T) {
	h := searchHome(40, catalog{list: listings(9)})
	h.Search("app")
	h.Doc.Layout()
	if len(tiles(h)) != 4*searchRows || h.Pages() != 4 {
		t.Fatalf("a search page holds %d tiles (3 rows), %d pages", len(tiles(h)), h.Pages())
	}
	h.Search("")
	h.Search("app")
	if h.Page() != 0 {
		t.Error("a new search starts on the first page")
	}
	h.Doc.Layout()
	root := h.Doc.Root.Elem()
	footer := root.Children()[3].Elem().Bounds
	if footer.Bottom() > h.Doc.H+0.01 {
		t.Errorf("the footer is pushed off the screen: %.0f > %.0f", footer.Bottom(), h.Doc.H)
	}
	for _, c := range root.Children()[2].Elem().Children() {
		if b := c.Elem().Bounds; b.Bottom() > footer.Y+0.01 {
			t.Errorf("%s overflows into the footer: %+v", c.Elem().ID, b)
		}
	}
	// every tile can be touched at its center
	tap(h.Doc, tiles(h)[0])
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

func TestTappingAnAvailableAppDownloadsItAndMovesItToInstalled(t *testing.T) {
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
	h.Search("web")
	h.Doc.Layout()
	tap(h.Doc, find(h, "avail-web00"))

	if len(inst.installed) != 1 || inst.installed[0] != "web00" || flushed != 1 {
		t.Fatalf("installed %v, flushed %d", inst.installed, flushed)
	}
	h.Doc.Layout()
	if find(h, "app-web00") == nil || find(h, "avail-web00") != nil || find(h, "avail-web01") == nil {
		t.Error("web00 is installed now; web01 is still available")
	}
	if !strings.Contains(texts(h), "“Web 00” instalado") {
		t.Errorf("the result is told: %s", texts(h))
	}
}

func TestAFailedInstallIsToldAndTheListsStayAsTheyWere(t *testing.T) {
	flushed := 0
	h, _ := installHome(&installer{err: errors.New("sem rede")}, &flushed)
	h.Search("web")
	h.Doc.Layout()
	tap(h.Doc, find(h, "avail-web01"))
	h.Doc.Layout()
	if !strings.Contains(texts(h), "Não foi possível instalar “Web 01”: sem rede") || find(h, "avail-web01") == nil {
		t.Errorf("%s", texts(h))
	}
}

func TestWithoutAnInstallerTheRowsOnlyShow(t *testing.T) {
	h := searchHome(1, catalog{list: listings(1)})
	h.Search("web")
	h.Doc.Layout()
	if row := find(h, "avail-web00"); row == nil || row.Elem().OnClick != nil {
		t.Error("a row without an installer is not tappable")
	}
}
