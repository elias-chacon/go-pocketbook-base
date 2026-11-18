package presentation

import (
	"errors"
	"fmt"
	"image"
	"testing"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/launcher/application"
	"go-pocketbook-base/internal/launcher/domain"
	platform "go-pocketbook-base/internal/platform/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

type registry struct{ apps []platform.InstalledApp }

func (r registry) Save(platform.InstalledApp) error { return nil }
func (r registry) List() []platform.InstalledApp    { return r.apps }

type launcher struct {
	opened []string
	err    error
}

func (l *launcher) Launch(t domain.Tile) error { l.opened = append(l.opened, t.ID); return l.err }

func apps(n int) []platform.InstalledApp {
	out := make([]platform.InstalledApp, n)
	for i := range out {
		id := fmt.Sprintf("app%02d", i)
		out[i] = platform.InstalledApp{
			Manifest: platform.Manifest{ID: id, Title: map[string]string{"pt-BR": "App " + id}, Description: map[string]string{"pt-BR": "d"}},
			Exe:      "/apps/" + id + ".app",
		}
	}
	return out
}

func newHome(n int, grid domain.Grid, l *launcher) *Home {
	return NewHome(Config{
		Width: 1404, Height: 1872, Tr: NewTranslator("pt-BR"), Grid: grid,
		Home: application.Home{Apps: registry{apps: apps(n)}, Launcher: l, SelfID: "launcher"},
		IconFor: func(domain.Tile) image.Image {
			return image.NewGray(image.Rect(0, 0, 128, 128))
		},
	})
}

func tapAt(d *widget.Document, p geom.Point) {
	d.HandlePointer(widget.PointerDown, p.X*d.Scale, p.Y*d.Scale)
	d.HandlePointer(widget.PointerUp, p.X*d.Scale, p.Y*d.Scale)
}

func tap(d *widget.Document, w widget.Widget) { tapAt(d, w.Elem().Bounds.Center()) }

func tiles(h *Home) []widget.Widget {
	var out []widget.Widget
	h.Doc.Walk(func(w widget.Widget) bool {
		if id := w.Elem().ID; len(id) > 4 && id[:4] == "app-" {
			out = append(out, w)
		}
		return true
	})
	return out
}

func TestEveryTextIsTranslatedInBothLanguages(t *testing.T) {
	tr := NewTranslator("pt-BR")
	for _, lang := range []string{"pt-BR", "en"} {
		if missing := tr.Missing(lang); len(missing) != 0 {
			t.Errorf("%s lacks: %v", lang, missing)
		}
	}
	if NewTranslator("en").T("home.title") != "My apps" || tr.T("home.title") != "Meus aplicativos" {
		t.Error("the title follows the language")
	}
}

func TestTheHomeShowsTheInstalledAppsAndOpensOneOnTap(t *testing.T) {
	l := &launcher{}
	h := newHome(3, domain.Grid{Columns: 3, Rows: 2}, l)
	h.Doc.Layout()

	got := tiles(h)
	if len(got) != 3 {
		t.Fatalf("tiles = %d", len(got))
	}
	for _, w := range got {
		if target := h.Doc.TargetAt(w.Elem().Bounds.Center()); target != w {
			t.Errorf("a touch at the center of %s reaches %T", w.Elem().ID, target)
		}
		if b := w.Elem().Bounds; b.W < 56 || b.H < 56 {
			t.Errorf("tile %s is too small to tap: %vx%v", w.Elem().ID, b.W, b.H)
		}
	}
	// tapping the title text inside the tile (not its border) opens it too: the click bubbles up
	tap(h.Doc, got[1])
	if len(l.opened) != 1 || l.opened[0] != "app01" {
		t.Errorf("opened = %v", l.opened)
	}
	if h.Doc.Render().Bounds().Dx() != 1404 {
		t.Error("renders")
	}
}

func TestAnAppThatCannotBeStartedExplainsWhy(t *testing.T) {
	l := &launcher{err: errors.New("exec format error")}
	h := newHome(1, domain.DefaultGrid, l)
	h.Doc.Layout()
	tap(h.Doc, tiles(h)[0])
	if h.note.Text != "Não foi possível abrir “App app00”: exec format error" {
		t.Errorf("message = %q", h.note.Text)
	}
}

func TestPagingWithButtonsAndMethods(t *testing.T) {
	h := newHome(7, domain.Grid{Columns: 2, Rows: 2}, &launcher{}) // 4 per page -> 2 pages
	h.Doc.Layout()
	if h.Pages() != 2 || h.Page() != 0 || h.indicator.Text != "1 / 2" || len(tiles(h)) != 4 {
		t.Fatalf("first page: pages=%d page=%d %q tiles=%d", h.Pages(), h.Page(), h.indicator.Text, len(tiles(h)))
	}
	if !h.prev.Disabled || h.next.Disabled {
		t.Error("on the first page only Next is enabled")
	}

	h.Doc.Layout()
	tap(h.Doc, h.next)
	if h.Page() != 1 || h.indicator.Text != "2 / 2" || len(tiles(h)) != 3 {
		t.Fatalf("second page: page=%d %q tiles=%d", h.Page(), h.indicator.Text, len(tiles(h)))
	}
	if h.prev.Disabled || !h.next.Disabled {
		t.Error("on the last page only Previous is enabled")
	}
	h.NextPage() // stops at the end
	if h.Page() != 1 {
		t.Error("Next stops at the last page")
	}
	h.Doc.Layout()
	tap(h.Doc, h.prev)
	h.PrevPage() // stops at the start
	if h.Page() != 0 {
		t.Errorf("Previous stops at the first page: %d", h.Page())
	}
}

func TestAnEmptyHomeSaysHowToGetApps(t *testing.T) {
	h := newHome(0, domain.DefaultGrid, &launcher{})
	h.Doc.Layout()
	if len(tiles(h)) != 0 {
		t.Error("no apps, no tiles")
	}
	var empty widget.Widget
	h.Doc.Walk(func(w widget.Widget) bool {
		if w.Elem().ID == "home-empty" {
			empty = w
		}
		return empty == nil
	})
	if empty == nil {
		t.Fatal("the empty state explains what to do")
	}
	if h.Pages() != 1 || !h.prev.Disabled || !h.next.Disabled {
		t.Error("an empty home has one page and nowhere to go")
	}
}

func TestRefreshPicksUpAppsInstalledMeanwhile(t *testing.T) {
	reg := &registry{apps: apps(1)}
	h := NewHome(Config{
		Width: 1404, Height: 1872, Tr: NewTranslator("en"),
		Home: application.Home{Apps: reg, Launcher: &launcher{}},
	})
	h.Doc.Layout()
	if len(tiles(h)) != 1 {
		t.Fatal("one app")
	}
	reg.apps = apps(3)
	h.Refresh()
	h.Doc.Layout()
	if len(tiles(h)) != 3 {
		t.Errorf("after Refresh: %d tiles", len(tiles(h)))
	}
}

func TestAFullDefaultPageFitsTheScreen(t *testing.T) {
	h := newHome(40, domain.DefaultGrid, &launcher{}) // 2 pages of 20
	h.Doc.Layout()
	if len(tiles(h)) != 20 {
		t.Fatalf("a default page holds 20 tiles, got %d", len(tiles(h)))
	}
	root := h.Doc.Root.Elem()
	footer := root.Children()[3].Elem().Bounds
	if footer.Bottom() > h.Doc.H+0.01 {
		t.Errorf("the footer is pushed off the screen: %.0f > %.0f", footer.Bottom(), h.Doc.H)
	}
	body := root.Children()[2].Elem().Bounds
	for _, w := range tiles(h) {
		if b := w.Elem().Bounds; b.Bottom() > body.Bottom()+0.01 || b.Right() > h.Doc.W+0.01 {
			t.Errorf("tile %s overflows: %+v (body bottom %.0f)", w.Elem().ID, b, body.Bottom())
		}
	}
}

func TestAMissingIconDrawsNothingInsteadOfFailing(t *testing.T) {
	h := NewHome(Config{
		Width: 1404, Height: 1872, Tr: NewTranslator("pt-BR"),
		Home: application.Home{Apps: registry{apps: apps(2)}, Launcher: &launcher{}},
		// IconFor is nil: no icons at all
	})
	h.Doc.Layout()
	if len(tiles(h)) != 2 || h.Doc.Render() == nil {
		t.Error("the tiles are still there, with their titles")
	}
}
