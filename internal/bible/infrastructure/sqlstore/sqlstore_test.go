package sqlstore

import (
	"path/filepath"
	"testing"
	"time"

	"go-pocketbook-base/internal/bible/domain"
	"go-pocketbook-base/internal/storage"
)

func open(t *testing.T) *Store {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	s, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func chapter(version string, book, ch int, texts ...string) domain.ChapterText {
	c := domain.ChapterText{Version: version, Book: book, Chapter: ch}
	for i, t := range texts {
		c.Verses = append(c.Verses, domain.Verse{Number: i + 1, Text: t})
	}
	return c
}

func seed(t *testing.T, s *Store) {
	t.Helper()
	if err := s.SaveVersions([]domain.Version{
		{Slug: "kjv", Name: "King James", ShortName: "KJV", Language: "en", Books: 66, Chapters: 1189},
		{Slug: "short", Name: "Short", Language: "en", Books: 1, Chapters: 1},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveChapters([]domain.ChapterText{
		chapter("kjv", 1, 1, "In the beginning God created", "And the earth was without form"),
		chapter("kjv", 1, 2, "Thus the heavens", "And on the seventh day"),
		chapter("kjv", 43, 3, "There was a man", "The same came by night", "Jesus answered"),
		chapter("short", 1, 1, "Only the first verse"),
	}); err != nil {
		t.Fatal(err)
	}
}

func TestBooksAndVersionsAreKeptAndUpdatedInPlace(t *testing.T) {
	s := open(t)
	books := []domain.Book{
		{ID: 43, Slug: "john", Names: map[string]string{"en": "John", "pt-br": "João"}, Chapters: 21, New: true},
		{ID: 1, Slug: "genesis", Names: map[string]string{"en": "Genesis"}, Chapters: 50},
	}
	if err := s.SaveBooks(books); err != nil {
		t.Fatal(err)
	}
	got, err := s.Books()
	if err != nil || len(got) != 2 || got[0].ID != 1 || got[1].Name("pt-br") != "João" || !got[1].New || got[0].New {
		t.Fatalf("books in canonical order: %+v %v", got, err)
	}
	books[0].Chapters = 22
	if err := s.SaveBooks(books); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Books(); len(got) != 2 || got[1].Chapters != 22 {
		t.Errorf("saved again it is updated, not duplicated: %+v", got)
	}

	seed(t, s)
	if err := s.MarkDownloaded("kjv"); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveVersions([]domain.Version{{Slug: "kjv", Name: "King James Version", ShortName: "KJV", Language: "en", Books: 66, Chapters: 1189}}); err != nil {
		t.Fatal(err)
	}
	v, ok, err := s.Version("kjv")
	if err != nil || !ok || v.Name != "King James Version" || !v.Downloaded || v.Verses != 7 {
		t.Errorf("a refreshed catalog keeps what was downloaded: %+v %v %v", v, ok, err)
	}
	if _, ok, _ := s.Version("nope"); ok {
		t.Error("an unknown version is not found")
	}
	all, _ := s.Versions()
	if len(all) != 2 {
		t.Errorf("versions: %+v", all)
	}
}

func TestAChapterIsReadByRangeAndStoredAgainWithoutLeftovers(t *testing.T) {
	s := open(t)
	seed(t, s)
	c, err := s.Chapter("kjv", 43, 3)
	if err != nil || len(c.Verses) != 3 || c.Verses[2] != (domain.Verse{Number: 3, Text: "Jesus answered"}) {
		t.Fatalf("chapter: %+v %v", c, err)
	}
	if empty, _ := s.Chapter("kjv", 43, 4); len(empty.Verses) != 0 {
		t.Error("a chapter that is not there is empty")
	}
	if err := s.SaveChapters([]domain.ChapterText{chapter("kjv", 43, 3, "Only one now")}); err != nil {
		t.Fatal(err)
	}
	if c, _ := s.Chapter("kjv", 43, 3); len(c.Verses) != 1 {
		t.Errorf("storing a chapter again replaces it: %+v", c)
	}
	if err := s.SaveChapters(nil); err != nil {
		t.Error(err)
	}

	stored, err := s.StoredChapters("kjv")
	if err != nil || len(stored) != 3 || !stored[domain.ChapterRef{Book: 1, Chapter: 2}] {
		t.Errorf("stored chapters: %v %v", stored, err)
	}
	text, ok, err := s.VerseText("kjv", domain.Ref{Book: 1, Chapter: 1, Verse: 2})
	if err != nil || !ok || text != "And the earth was without form" {
		t.Errorf("verse: %q %v %v", text, ok, err)
	}
	if _, ok, _ := s.VerseText("kjv", domain.Ref{Book: 9, Chapter: 9, Verse: 9}); ok {
		t.Error("a verse that is not there")
	}
	pre, err := s.Previews("kjv", 1, 10)
	if err != nil || pre[1] != "In the begi" && pre[1] != "In the beg" || len(pre) != 2 {
		t.Errorf("previews are the start of verse 1: %v %v", pre, err)
	}
}

func TestSearchFindsWordsWithoutAccentsAndStopsAtTheLimit(t *testing.T) {
	s := open(t)
	if err := s.SaveChapters([]domain.ChapterText{chapter("pt", 1, 1, "No princípio Deus criou", "A terra era sem forma", "Deus disse: haja luz")}); err != nil {
		t.Fatal(err)
	}
	hits, err := s.Search("pt", domain.Words("PRINCIPIO deus"), 10)
	if err != nil || len(hits) != 1 || hits[0].Ref != (domain.Ref{Book: 1, Chapter: 1, Verse: 1}) {
		t.Fatalf("hits: %+v %v", hits, err)
	}
	if hits, _ := s.Search("pt", []string{"deus"}, 1); len(hits) != 1 {
		t.Errorf("the limit stops the search: %+v", hits)
	}
	if hits, _ := s.Search("pt", nil, 5); hits != nil {
		t.Error("no words, no results")
	}
	if hits, _ := s.Search("other", []string{"deus"}, 5); len(hits) != 0 {
		t.Error("another version has nothing")
	}
}

func TestProgressIsPerPlaceNotPerVersionAndAdaptsToTheVersion(t *testing.T) {
	s := open(t)
	seed(t, s)
	at := time.Unix(1_700_000_000, 0)
	if err := s.MarkRead([]domain.Ref{domain.At(1, 1, 1), domain.At(1, 1, 2), domain.At(1, 1, 2), domain.At(43, 3, 1)}, at); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkRead(nil, at); err != nil {
		t.Fatal(err)
	}
	read, total, err := s.ReadCount("kjv")
	if err != nil || read != 3 || total != 7 {
		t.Fatalf("kjv: read=%d total=%d %v", read, total, err)
	}
	// the other version has one verse only: the place Gn 1:1 is the same, the rest does not count
	read, total, _ = s.ReadCount("short")
	if read != 1 || total != 1 {
		t.Errorf("short: read=%d total=%d (the denominator follows the version)", read, total)
	}
	if ok, _ := s.IsRead(domain.Ref{Book: 43, Chapter: 3, Verse: 1}); !ok {
		t.Error("read")
	}
	if ok, _ := s.IsRead(domain.Ref{Book: 43, Chapter: 3, Verse: 2}); ok {
		t.Error("not read")
	}
	if r, tt, _ := s.ReadCount("none"); r != 0 || tt != 0 {
		t.Errorf("no version, no progress: %d/%d", r, tt)
	}
}

func TestPickVerseIsAnyVerseOfTheVersion(t *testing.T) {
	s := open(t)
	seed(t, s)
	first, _ := s.PickVerse("kjv", 0)
	if first != (domain.Ref{Book: 1, Chapter: 1, Verse: 1}) {
		t.Errorf("first = %+v", first)
	}
	last, _ := s.PickVerse("kjv", 6)
	wrapped, _ := s.PickVerse("kjv", 7+6)
	neg, _ := s.PickVerse("kjv", -6)
	if last != (domain.Ref{Book: 43, Chapter: 3, Verse: 3}) || wrapped != last || neg != last {
		t.Errorf("last=%+v wrapped=%+v negative=%+v", last, wrapped, neg)
	}
	if r, err := s.PickVerse("empty", 3); err != nil || r.Valid() {
		t.Errorf("a version with nothing gives no verse: %+v %v", r, err)
	}
}

func TestDeletingAVersionFreesTheTextAndKeepsTheProgress(t *testing.T) {
	s := open(t)
	seed(t, s)
	_ = s.MarkDownloaded("kjv")
	_ = s.MarkRead([]domain.Ref{domain.At(1, 1, 1)}, time.Now())
	if err := s.DeleteVersion("kjv"); err != nil {
		t.Fatal(err)
	}
	v, _, _ := s.Version("kjv")
	if v.Downloaded || v.Verses != 0 {
		t.Errorf("after deleting: %+v", v)
	}
	if c, _ := s.Chapter("kjv", 1, 1); len(c.Verses) != 0 {
		t.Error("the text is gone")
	}
	if ok, _ := s.IsRead(domain.Ref{Book: 1, Chapter: 1, Verse: 1}); !ok {
		t.Error("what was read stays read")
	}
}

func TestPositionPreferencesAndTheDailyVerseSurviveReopening(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bible.db")
	db, err := storage.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	s, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	if p, _ := s.Position(); p.Valid() {
		t.Errorf("nothing opened yet: %+v", p)
	}
	if _, ok, _ := s.LoadPreferences(); ok {
		t.Error("no preferences saved yet")
	}
	prefs := domain.Preferences{Lang: "en", VersionLang: "pt-br", Version: "almeida", FontSize: 44, Font: "mono", Invert: true, Speech: true, ChapterTitles: false, Alphabetical: true}
	if err := s.SavePosition(domain.Position{Book: 43, Chapter: 3, Verse: 16}); err != nil {
		t.Fatal(err)
	}
	if err := s.SavePosition(domain.Position{Book: 43, Chapter: 4, Verse: 2}); err != nil {
		t.Fatal(err)
	}
	if err := s.SavePreferences(prefs); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveDaily(domain.DailyVerse{Day: "2026-11-19", Ref: domain.Ref{Book: 19, Chapter: 23, Verse: 1}}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveDaily(domain.DailyVerse{Day: "2026-11-19", Ref: domain.Ref{Book: 19, Chapter: 23, Verse: 1}, Read: true}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveDaily(domain.DailyVerse{Day: "2026-09-01", Ref: domain.Ref{Book: 1, Chapter: 1, Verse: 1}}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveDaily(domain.DailyVerse{Day: "2026-11-20", Ref: domain.Ref{Book: 1, Chapter: 1, Verse: 1}}); err != nil {
		t.Fatal(err)
	}
	db.Close()

	db, err = storage.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s, err = New(db)
	if err != nil {
		t.Fatal(err)
	}
	if p, _ := s.Position(); p != (domain.Position{Book: 43, Chapter: 4, Verse: 2}) {
		t.Errorf("the last position wins: %+v", p)
	}
	got, ok, err := s.LoadPreferences()
	if err != nil || !ok || got != prefs {
		t.Errorf("preferences: %+v %v %v", got, ok, err)
	}
	d, ok, _ := s.Daily("2026-11-19")
	if !ok || !d.Read || d.Ref.Book != 19 {
		t.Errorf("daily: %+v %v", d, ok)
	}
	if _, ok, _ := s.Daily("2026-09-01"); ok {
		t.Error("days older than a month are forgotten")
	}
	if _, ok, _ := s.Daily("2027-01-01"); ok {
		t.Error("no verse chosen for that day")
	}
}
