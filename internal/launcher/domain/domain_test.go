package domain

import (
	"reflect"
	"testing"

	platform "go-pocketbook-base/internal/platform/domain"
)

func app(id, pt, en string) platform.InstalledApp {
	return platform.InstalledApp{
		Manifest: platform.Manifest{
			ID:          id,
			Title:       map[string]string{"pt-BR": pt, "en": en},
			Description: map[string]string{"pt-BR": "desc " + pt, "en": "desc " + en},
		},
		Version: "1.0.0", Exe: "/apps/" + id + ".app", Icon: "/icons/" + id + ".bmp",
	}
}

func ids(tiles []Tile) []string {
	out := make([]string, len(tiles))
	for i, t := range tiles {
		out[i] = t.ID
	}
	return out
}

func TestTilesAreTranslatedSortedAndDoNotListTheLauncher(t *testing.T) {
	apps := []platform.InstalledApp{
		app("zeta", "Zebra", "Zebra"), app("launcher", "Launcher", "Launcher"),
		app("books", "livros", "Books"), app("alpha", "Álbum", "Album"),
	}
	pt := TilesFrom(apps, "pt-BR", "launcher")
	if got := ids(pt); !reflect.DeepEqual(got, []string{"alpha", "books", "zeta"}) {
		// "Álbum" < "livros" < "Zebra": case and accents do not count
		t.Fatalf("pt order = %v", got)
	}
	en := TilesFrom(apps, "en", "launcher")
	if got := ids(en); !reflect.DeepEqual(got, []string{"alpha", "books", "zeta"}) {
		t.Fatalf("en order = %v (sorted by the title in the language shown)", got)
	}
	if en[1].Title != "Books" || en[1].Description != "desc Books" || en[1].Exe != "/apps/books.app" ||
		en[1].IconPath != "/icons/books.bmp" || en[1].Version != "1.0.0" {
		t.Errorf("tile = %+v", en[1])
	}
	if len(TilesFrom(nil, "en", "x")) != 0 {
		t.Error("no apps, no tiles")
	}
}

func TestGridLimitsAndPaging(t *testing.T) {
	if g := (Grid{}).Normalized(); g != DefaultGrid {
		t.Errorf("a zero grid is the default: %+v", g)
	}
	if g := (Grid{Columns: -3, Rows: 99}).Normalized(); g.Columns != MinSide || g.Rows != MaxSide {
		t.Errorf("limits: %+v", g)
	}
	g := Grid{Columns: 2, Rows: 2} // 4 per page
	if g.PerPage() != 4 || g.Pages(0) != 1 || g.Pages(4) != 1 || g.Pages(5) != 2 || g.Pages(9) != 3 {
		t.Errorf("paging: per=%d pages(0,4,5,9)=%d %d %d %d", g.PerPage(), g.Pages(0), g.Pages(4), g.Pages(5), g.Pages(9))
	}

	tiles := make([]Tile, 9)
	for i := range tiles {
		tiles[i] = Tile{ID: string(rune('a' + i))}
	}
	if got := ids(g.Page(tiles, 0)); !reflect.DeepEqual(got, []string{"a", "b", "c", "d"}) {
		t.Errorf("page 0 = %v", got)
	}
	if got := ids(g.Page(tiles, 2)); !reflect.DeepEqual(got, []string{"i"}) {
		t.Errorf("the last page holds what is left: %v", got)
	}
	if got := ids(g.Page(tiles, 99)); !reflect.DeepEqual(got, []string{"i"}) {
		t.Errorf("a page past the end is the last one: %v", got)
	}
	if got := g.Page(tiles, -5); ids(got)[0] != "a" {
		t.Errorf("a negative page is the first one: %v", ids(got))
	}
	if g.Page(nil, 0) != nil {
		t.Error("no tiles, an empty page")
	}
	if ClampPage(3, 3) != 2 || ClampPage(-1, 3) != 0 || ClampPage(1, 3) != 1 {
		t.Error("ClampPage")
	}
}
