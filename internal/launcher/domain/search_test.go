package domain

import (
	"testing"

	platform "go-pocketbook-base/internal/platform/domain"
)

func TestMatchesIgnoresCaseAccentsAndWordOrder(t *testing.T) {
	cases := []struct {
		query  string
		fields []string
		want   bool
	}{
		{"", []string{"qualquer"}, true},
		{"  ", []string{"x"}, true},
		{"ola", []string{"Olá Mundo"}, true},
		{"MUNDO olá", []string{"Olá Mundo"}, true},
		{"calc", []string{"Agenda", "Uma calculadora"}, true},
		{"xyz", []string{"Agenda"}, false},
		{"agenda xyz", []string{"Agenda"}, false},
	}
	for _, c := range cases {
		if got := Matches(c.query, c.fields...); got != c.want {
			t.Errorf("Matches(%q, %v) = %v", c.query, c.fields, got)
		}
	}
}

func TestFilterTiles(t *testing.T) {
	tiles := []Tile{{ID: "hello", Title: "Hello Go"}, {ID: "notes", Title: "Notas", Description: "Anotações"}}
	if got := FilterTiles(tiles, "anotacoes"); len(got) != 1 || got[0].ID != "notes" {
		t.Errorf("description matches: %v", got)
	}
	if got := FilterTiles(tiles, ""); len(got) != 2 {
		t.Errorf("empty query keeps all: %v", got)
	}
}

func inst(id, title string) platform.InstalledApp {
	return platform.InstalledApp{Manifest: platform.Manifest{ID: id, Title: map[string]string{"pt-BR": title}, Description: map[string]string{"pt-BR": "d"}}}
}

func TestAppsOfIsTheWholeEcosystemInstalledOnesAndTheOnesOnlyTheCatalogHas(t *testing.T) {
	listings := []Listing{{ID: "hello", Title: "Hello Go", Version: "2.0.0"}, {ID: "bible", Title: "Bíblia", Version: "2.0.0"}, {ID: "launcher", Title: "Launcher"}}
	tiles := AppsOf([]platform.InstalledApp{inst("hello", "Hello Go"), inst("launcher", "Launcher")}, listings, "pt-BR", "launcher")
	if len(tiles) != 2 || tiles[0].ID != "bible" || tiles[1].ID != "hello" {
		t.Fatalf("sorted, without the Launcher: %+v", tiles)
	}
	if tiles[0].Installed || tiles[0].Listing == nil {
		t.Errorf("a missing app is not installed and knows its listing: %+v", tiles[0])
	}
	if !tiles[1].Installed || tiles[1].Listing == nil || tiles[1].Listing.Version != "2.0.0" {
		t.Errorf("an installed app keeps the listing the update is made from: %+v", tiles[1])
	}
	local := AppsOf([]platform.InstalledApp{inst("mine", "Mine")}, nil, "pt-BR", "launcher")
	if len(local) != 1 || local[0].Listing != nil || !local[0].Installed {
		t.Errorf("an app the catalog does not know: %+v", local)
	}
}

func TestVisibleAndHiddenSplitTheTilesByTheHiddenIds(t *testing.T) {
	tiles := []Tile{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	if v, h := Visible(tiles, []string{"b"}), Hidden(tiles, []string{"b", "zzz"}); len(v) != 2 || v[1].ID != "c" || len(h) != 1 || h[0].ID != "b" {
		t.Errorf("visible %v hidden %v", v, h)
	}
}

func TestPreferencesHideAndShowAnAppWithoutRepeats(t *testing.T) {
	p := DefaultPreferences().WithHidden("bible", true).WithHidden("bible", true).WithHidden("mail", true)
	if len(p.HiddenApps) != 2 || p.HiddenApps[0] != "bible" {
		t.Fatalf("hidden: %v", p.HiddenApps)
	}
	if p = p.WithHidden("bible", false); len(p.HiddenApps) != 1 || p.HiddenApps[0] != "mail" {
		t.Errorf("shown again: %v", p.HiddenApps)
	}
	if n := (Preferences{HiddenApps: []string{" a ", "a", "", "b"}}).Normalized(); len(n.HiddenApps) != 2 {
		t.Errorf("normalized: %v", n.HiddenApps)
	}
}
