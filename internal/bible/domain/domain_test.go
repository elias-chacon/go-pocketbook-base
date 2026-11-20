package domain

import (
	"testing"
	"time"
)

func day(y int, m time.Month, d int) time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }

func TestEasterFollowsTheGregorianCalendar(t *testing.T) {
	for year, want := range map[int]time.Time{
		2024: day(2024, 3, 31), 2025: day(2025, 4, 20), 2026: day(2026, 4, 5), 2027: day(2027, 3, 28), 2038: day(2038, 4, 25),
	} {
		if got := Easter(year); !got.Equal(want) {
			t.Errorf("Easter(%d) = %v, want %v", year, got.Format("2006-01-02"), want.Format("2006-01-02"))
		}
	}
}

func TestFestiveVersesAreTiedToTheirDays(t *testing.T) {
	cases := map[time.Time]Ref{
		day(2026, 12, 25): {42, 2, 11},
		day(2026, 12, 24): {23, 9, 6},
		day(2027, 1, 1):   {25, 3, 23},
		day(2027, 1, 6):   {40, 2, 2},
		day(2026, 3, 29):  {40, 21, 9},  // Palm Sunday: a week before Easter (April 5)
		day(2026, 4, 3):   {43, 19, 30}, // Good Friday
		day(2026, 4, 5):   {40, 28, 6},  // Easter
		day(2026, 5, 14):  {44, 1, 9},   // Ascension: 39 days after
		day(2026, 5, 24):  {44, 2, 4},   // Pentecost: 49 days after
	}
	for d, want := range cases {
		got, ok := FestiveVerse(d)
		if !ok || got != want {
			t.Errorf("%s: %+v %v, want %+v", d.Format("2006-01-02"), got, ok, want)
		}
	}
	if _, ok := FestiveVerse(day(2026, 8, 12)); ok {
		t.Error("an ordinary day has no festive verse")
	}
}

func TestTheDailySeedIsStableForADayAndChangesWithIt(t *testing.T) {
	a, b := DailySeed(day(2026, 11, 19)), DailySeed(day(2026, 11, 19).Add(5*time.Hour))
	if a != b || a < 0 {
		t.Errorf("the same day: %d %d", a, b)
	}
	seen := map[int]bool{}
	for i := 0; i < 30; i++ {
		seen[DailySeed(day(2026, 11, 1).AddDate(0, 0, i))%1000] = true
	}
	if len(seen) < 25 {
		t.Errorf("30 days gave only %d different picks", len(seen))
	}
}

func TestBooksAreNamedAndOrdered(t *testing.T) {
	books := []Book{
		{ID: 2, Slug: "exodus", Names: map[string]string{"en": "Exodus", "pt-br": "Êxodo"}},
		{ID: 1, Slug: "genesis", Names: map[string]string{"en": "Genesis", "pt-br": "Gênesis"}},
		{ID: 3, Slug: "x"},
	}
	if books[0].Name("pt-BR") != "Êxodo" || books[0].Name("fr") != "Exodus" || books[2].Name("en") != "x" {
		t.Errorf("names: %q %q %q", books[0].Name("pt-BR"), books[0].Name("fr"), books[2].Name("en"))
	}
	if got := InCanonicalOrder(books); got[0].ID != 1 || got[2].ID != 3 {
		t.Errorf("canonical: %+v", got)
	}
	if got := SortAlphabetically(books, "pt-br"); got[0].Slug != "exodus" || got[1].Slug != "genesis" {
		t.Errorf("alphabetical (Êxodo before Gênesis): %+v", got)
	}
	if books[0].ID != 2 {
		t.Error("sorting must not change the original")
	}
}

func TestVersionsAreGroupedByLanguage(t *testing.T) {
	vs := []Version{
		{Slug: "kjv", Name: "King James", ShortName: "KJV", Language: "en"},
		{Slug: "web", Name: "World English", ShortName: "WEB", Language: "en"},
		{Slug: "al", Name: "Almeida", ShortName: "Almeida", Language: "pt-br"},
	}
	langs := LanguagesOf(vs)
	if len(langs) != 2 || langs[0].Tag != "en" || langs[0].Versions != 2 || langs[1].Tag != "pt-br" {
		t.Errorf("languages: %+v", langs)
	}
	if got := VersionsIn(vs, "EN"); len(got) != 2 || got[0].Slug != "kjv" {
		t.Errorf("english: %+v", got)
	}
	if vs[0].Label() != "King James (KJV)" || vs[2].Label() != "Almeida" || (Version{ShortName: "X"}).Label() != "X" {
		t.Errorf("labels: %q %q", vs[0].Label(), vs[2].Label())
	}
}

func TestSearchFoldsAccentsAndRequiresEveryWord(t *testing.T) {
	words := Words("  AMOR, Esperança! ")
	if len(words) != 2 || words[0] != "amor" || words[1] != "esperanca" {
		t.Fatalf("words = %v", words)
	}
	if !MatchesAll("A fé, a esperança e o amor", words) || MatchesAll("só o amor", words) || MatchesAll("x", nil) {
		t.Error("every word must be in the text, and no words match nothing")
	}
	if Fold("ÊXODO Ça") != "exodo ca" {
		t.Errorf("fold = %q", Fold("ÊXODO Ça"))
	}
}

func TestPercentStaysBetweenZeroAndAHundred(t *testing.T) {
	for _, c := range []struct {
		read, total int
		want        float64
	}{{0, 100, 0}, {50, 100, 50}, {100, 100, 100}, {150, 100, 100}, {-1, 100, 0}, {5, 0, 0}} {
		if got := Percent(c.read, c.total); got != c.want {
			t.Errorf("Percent(%d, %d) = %v, want %v", c.read, c.total, got, c.want)
		}
	}
}

func TestPreferencesAreNormalized(t *testing.T) {
	if p := DefaultPreferences(); p.FontSize != DefaultFontSize || !p.ChapterTitles || p.Font != "go" {
		t.Errorf("defaults: %+v", p)
	}
	p := Preferences{FontSize: 999, Font: "comic"}.Normalize()
	if p.FontSize != DefaultFontSize || p.Font != "go" {
		t.Errorf("out of range: %+v", p)
	}
	if p := (Preferences{FontSize: 41, Font: "mono"}).Normalize(); p.FontSize != 40 || p.Font != "mono" {
		t.Errorf("a size is a multiple of the step: %+v", p)
	}
	if (Position{}).Valid() || !(Position{Book: 1, Chapter: 1}).Valid() || (Ref{1, 1, 0}).Valid() || !(Ref{1, 1, 1}).Valid() {
		t.Error("validity")
	}
}
