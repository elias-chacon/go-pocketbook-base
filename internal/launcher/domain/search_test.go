package domain

import "testing"

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

func TestFilterTilesAndListings(t *testing.T) {
	tiles := []Tile{{ID: "hello", Title: "Hello Go"}, {ID: "notes", Title: "Notas", Description: "Anotações"}}
	if got := FilterTiles(tiles, "anotacoes"); len(got) != 1 || got[0].ID != "notes" {
		t.Errorf("description matches: %v", got)
	}
	if got := FilterTiles(tiles, ""); len(got) != 2 {
		t.Errorf("empty query keeps all: %v", got)
	}
	list := []Listing{{ID: "hello", Title: "Hello Go"}, {ID: "calc", Title: "Calculadora"}, {ID: "cal", Title: "Calendário"}}
	got := FilterListings(list, "cal", []string{"hello"})
	if len(got) != 2 || got[0].ID != "calc" || got[1].ID != "cal" {
		t.Errorf("installed ones are left out, order kept: %v", got)
	}
	if got := FilterListings(list, "hello", []string{"hello"}); len(got) != 0 {
		t.Errorf("an installed app is not available: %v", got)
	}
}
