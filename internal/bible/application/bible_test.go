package application

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"go-pocketbook-base/internal/bible/domain"
	"go-pocketbook-base/internal/bible/infrastructure/sqlstore"
	"go-pocketbook-base/internal/storage"
)

// fakeCatalog serves a tiny Bible: two books of two chapters, three verses each.
type fakeCatalog struct {
	calls   int
	failAt  int // fail the n-th call to Chapters (0: never)
	missing map[domain.ChapterRef]bool
}

func (f *fakeCatalog) Books() ([]domain.Book, error) {
	return []domain.Book{
		{ID: 1, Slug: "genesis", Names: map[string]string{"en": "Genesis"}, Chapters: 2},
		{ID: 43, Slug: "john", Names: map[string]string{"en": "John", "pt-br": "João"}, Chapters: 2, New: true},
	}, nil
}

func (f *fakeCatalog) Versions() ([]domain.Version, error) {
	return []domain.Version{
		{Slug: "kjv", Name: "King James", ShortName: "KJV", Language: "en", Books: 2, Chapters: 4},
		{Slug: "almeida-livre", Name: "Almeida", Language: "pt-br", Books: 2, Chapters: 4},
	}, nil
}

func (f *fakeCatalog) Chapters(version string, _ []domain.Book, refs []domain.ChapterRef) ([]domain.ChapterText, error) {
	f.calls++
	if f.failAt == f.calls {
		return nil, errors.New("network down")
	}
	var out []domain.ChapterText
	for _, r := range refs {
		if f.missing[r] {
			continue
		}
		c := domain.ChapterText{Version: version, Book: r.Book, Chapter: r.Chapter}
		for v := 1; v <= 3; v++ {
			c.Verses = append(c.Verses, domain.Verse{Number: v, Text: fmt.Sprintf("%s %d:%d:%d", version, r.Book, r.Chapter, v)})
		}
		out = append(out, c)
	}
	return out, nil
}

func setup(t *testing.T) (Bible, *fakeCatalog) {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	lib, err := sqlstore.New(db)
	if err != nil {
		t.Fatal(err)
	}
	cat := &fakeCatalog{}
	return Bible{Catalog: cat, Library: lib, Now: func() time.Time { return time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC) }}, cat
}

func TestBooksStartFromTheBuiltInCanonAndFollowTheCatalogAfterARefresh(t *testing.T) {
	b, _ := setup(t)
	books, err := b.Books()
	if err != nil || len(books) != 66 || books[0].Slug != "genesis" || books[65].Slug != "revelation" || books[42].Name("pt-BR") != "João" {
		t.Fatalf("the canon: %d books, %v", len(books), err)
	}
	if err := b.Refresh(); err != nil {
		t.Fatal(err)
	}
	if books, _ := b.Books(); len(books) != 2 {
		t.Errorf("after a refresh the catalog's books are used: %d", len(books))
	}
	if vs, _ := b.Versions(); len(vs) != 2 {
		t.Errorf("versions: %+v", vs)
	}
	if err := (Bible{Library: b.Library}).Refresh(); !errors.Is(err, ErrNoCatalog) {
		t.Errorf("no catalog: %v", err)
	}
	if err := (Bible{Library: b.Library}).Download("kjv", nil); !errors.Is(err, ErrNoCatalog) {
		t.Errorf("no catalog: %v", err)
	}
}

func TestDownloadingAVersionStoresItAndReportsProgress(t *testing.T) {
	b, cat := setup(t)
	if err := b.Refresh(); err != nil {
		t.Fatal(err)
	}
	var steps [][2]int
	if err := b.Download("kjv", func(done, total int) { steps = append(steps, [2]int{done, total}) }); err != nil {
		t.Fatal(err)
	}
	if len(steps) != 1 || steps[0] != [2]int{4, 4} {
		t.Errorf("progress: %v", steps)
	}
	v, _, _ := b.Library.Version("kjv")
	if !v.Downloaded || v.Verses != 12 {
		t.Errorf("version: %+v", v)
	}
	c, err := b.Chapter("kjv", 43, 2)
	if err != nil || len(c.Verses) != 3 || c.Verses[0].Text != "kjv 43:2:1" {
		t.Errorf("chapter: %+v %v", c, err)
	}
	calls := cat.calls
	if err := b.Download("kjv", nil); err != nil || cat.calls != calls {
		t.Errorf("a version already stored asks for nothing: %v (calls %d -> %d)", err, calls, cat.calls)
	}
	if err := b.Remove("kjv"); err != nil {
		t.Fatal(err)
	}
	if v, _, _ := b.Library.Version("kjv"); v.Downloaded || v.Verses != 0 {
		t.Errorf("removed: %+v", v)
	}
}

func TestAnInterruptedDownloadGoesOnFromWhereItStopped(t *testing.T) {
	b, cat := setup(t)
	_ = b.Refresh()
	// 66 books of the canon would be many batches; use the catalog's 4 chapters in batches of 20: one call.
	// Fail the first call, then succeed.
	cat.failAt = 1
	if err := b.Download("kjv", nil); err == nil {
		t.Fatal("the first attempt fails")
	}
	if v, _, _ := b.Library.Version("kjv"); v.Downloaded {
		t.Error("a failed download is not marked as downloaded")
	}
	cat.failAt = 0
	cat.missing = map[domain.ChapterRef]bool{{Book: 1, Chapter: 2}: true}
	if err := b.Download("kjv", nil); err != nil {
		t.Fatal(err)
	}
	stored, _ := b.Library.StoredChapters("kjv")
	if len(stored) != 3 {
		t.Errorf("a chapter the version lacks is left out: %v", stored)
	}

	cat.missing = map[domain.ChapterRef]bool{{Book: 1, Chapter: 1}: true, {Book: 1, Chapter: 2}: true, {Book: 43, Chapter: 1}: true, {Book: 43, Chapter: 2}: true}
	if err := b.Download("web", nil); err == nil {
		t.Error("a version with no text at all is an error")
	}
}

func TestReadingCountsVerseByPlaceAndThePercentFollowsTheVersion(t *testing.T) {
	b, _ := setup(t)
	_ = b.Refresh()
	_ = b.Download("kjv", nil)
	_ = b.Download("almeida-livre", nil)
	if st, _ := b.Progress("kjv"); st.Percent != 0 || st.Total != 12 {
		t.Fatalf("nothing read: %+v", st)
	}
	if err := b.Seen(1, 1, []int{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	if err := b.Seen(1, 1, nil); err != nil {
		t.Fatal(err)
	}
	st, _ := b.Progress("kjv")
	other, _ := b.Progress("almeida-livre")
	if st.Read != 3 || other.Read != 3 || st.Percent != 25 || other.Percent != 25 {
		t.Errorf("reading a verse counts in every version that has it: %+v %+v", st, other)
	}
	if pos, _ := b.Resume(); pos != (domain.Position{Book: 1, Chapter: 1, Verse: 1}) {
		t.Errorf("resume: %+v", pos)
	}
	if err := b.Remember(domain.Position{Book: 43, Chapter: 2, Verse: 3}); err != nil {
		t.Fatal(err)
	}
	if pos, _ := b.Resume(); pos.Book != 43 || pos.Verse != 3 {
		t.Errorf("remember: %+v", pos)
	}
	if st, _ := b.Progress("kjv"); st.Read != 3 {
		t.Error("remembering a place does not count as reading")
	}
}

func TestPreferencesStartFromTheLanguageOfTheReader(t *testing.T) {
	b, _ := setup(t)
	p, err := b.Preferences("pt-BR")
	if err != nil || p.Version != "almeida-livre" || p.VersionLang != "pt-br" || p.Lang != "pt-BR" || p.FontSize != domain.DefaultFontSize {
		t.Fatalf("portuguese reader: %+v %v", p, err)
	}
	if p, _ := b.Preferences("fr"); p.Version != "kjv" || p.VersionLang != "en" {
		t.Errorf("a language with no default starts in English: %+v", p)
	}
	p.Invert, p.Font, p.FontSize = true, "mono", 48
	if err := b.SavePreferences(p); err != nil {
		t.Fatal(err)
	}
	if got, _ := b.Preferences("pt-BR"); !got.Invert || got.Font != "mono" || got.FontSize != 48 {
		t.Errorf("saved preferences win over the language of the reader: %+v", got)
	}
}

func TestTheVerseOfTheDayIsStableFestiveWhenItCanBeAndCountsWhenRead(t *testing.T) {
	b, _ := setup(t)
	_ = b.Refresh()
	day := time.Date(2026, 11, 19, 9, 0, 0, 0, time.UTC)
	if _, ok, err := b.VerseOfTheDay("kjv", day); ok || err != nil {
		t.Fatalf("with nothing downloaded there is no verse: %v %v", ok, err)
	}
	_ = b.Download("kjv", nil)

	d, ok, err := b.VerseOfTheDay("kjv", day)
	if err != nil || !ok || !d.Ref.Valid() || d.Text == "" || d.Festive || d.Read {
		t.Fatalf("a day: %+v %v %v", d, ok, err)
	}
	again, _, _ := b.VerseOfTheDay("kjv", day.Add(3*time.Hour))
	if again.Ref != d.Ref {
		t.Errorf("the same all day: %+v %+v", d.Ref, again.Ref)
	}

	before, _ := b.Progress("kjv")
	if err := b.DailyRead(day); err != nil {
		t.Fatal(err)
	}
	after, _ := b.Progress("kjv")
	if after.Read != before.Read+1 {
		t.Errorf("reading the verse of the day counts: %d -> %d", before.Read, after.Read)
	}
	if d, _, _ := b.VerseOfTheDay("kjv", day); !d.Read {
		t.Error("it stays read")
	}
	if err := b.DailyRead(day.AddDate(0, 0, 1)); err != nil {
		t.Errorf("a day with no verse chosen is ignored: %v", err)
	}

	// Palm Sunday 2026 is Matthew 21:9; this tiny Bible has no Matthew, so it falls back to a drawn verse
	palm := time.Date(2026, 3, 29, 0, 0, 0, 0, time.UTC)
	if d, ok, _ := b.VerseOfTheDay("kjv", palm); !ok || d.Festive {
		t.Errorf("a festive verse the version lacks falls back: %+v %v", d, ok)
	}
	// John 19:30 does not exist here either; but a version that has the verse uses it
	lib := b.Library
	_ = lib.SaveChapters([]domain.ChapterText{{Version: "kjv", Book: 42, Chapter: 2, Verses: []domain.Verse{{Number: 11, Text: "For unto you is born this day"}}}})
	xmas := time.Date(2026, 12, 25, 0, 0, 0, 0, time.UTC)
	if d, ok, _ := b.VerseOfTheDay("kjv", xmas); !ok || !d.Festive || d.Ref != (domain.Ref{Book: 42, Chapter: 2, Verse: 11}) {
		t.Errorf("christmas: %+v %v", d, ok)
	}
}

func TestSearchAndPreviewsReadFromTheStoredText(t *testing.T) {
	b, _ := setup(t)
	_ = b.Refresh()
	_ = b.Download("kjv", nil)
	hits, err := b.Search("kjv", "KJV 1:2:3")
	if err != nil || len(hits) == 0 || hits[0].Ref != domain.At(1, 2, 3) {
		t.Errorf("search: %+v %v", hits, err)
	}
	if hits, _ := b.Search("kjv", "  "); len(hits) != 0 {
		t.Error("an empty search finds nothing")
	}
	pre, err := b.ChapterPreviews("kjv", 1)
	if err != nil || len(pre) != 2 || pre[1] == "" {
		t.Errorf("previews: %v %v", pre, err)
	}
}
