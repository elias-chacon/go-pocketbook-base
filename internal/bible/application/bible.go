// Package application holds the use cases of the Bible app: keeping the catalog, downloading
// a version for offline reading, reading a chapter, counting what was read, resuming where
// the person stopped and choosing the verse of the day. It talks to the Bible only through
// the ports of the domain.
package application

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go-pocketbook-base/internal/bible/domain"
)

// ErrNoCatalog is returned when something needs the network and there is no catalog (no HTTP client).
var ErrNoCatalog = errors.New("bible: there is no connection to the catalog")

// ErrNotDownloaded is returned when a version has no text on the reader.
var ErrNotDownloaded = errors.New("bible: the version is not downloaded")

// SearchLimit is the most results a search brings (a page of them is shown at a time).
const SearchLimit = 60

// defaults are the versions a language starts with (any other starts with its first one).
var defaults = map[string]string{"pt-br": "almeida-livre", "en": "kjv"}

// Bible is the set of use cases.
type Bible struct {
	// Catalog is the API; nil when the app has no way to the network (it still reads what is stored).
	Catalog domain.Catalog
	Library domain.Library
	// Now is the clock (time.Now when nil).
	Now func() time.Time
}

func (b Bible) now() time.Time {
	if b.Now != nil {
		return b.Now()
	}
	return time.Now()
}

// Books are the books of the Bible in the canonical order: the ones the catalog gave, or the
// built-in canon until it is fetched.
func (b Bible) Books() ([]domain.Book, error) {
	books, err := b.Library.Books()
	if err != nil {
		return nil, err
	}
	if len(books) == 0 {
		return domain.Canon, nil
	}
	return books, nil
}

// Versions are the translations known to the reader, with whether each is downloaded.
func (b Bible) Versions() ([]domain.Version, error) { return b.Library.Versions() }

// Refresh fetches the books and the versions from the catalog and keeps them.
func (b Bible) Refresh() error {
	if b.Catalog == nil {
		return ErrNoCatalog
	}
	books, err := b.Catalog.Books()
	if err != nil {
		return err
	}
	versions, err := b.Catalog.Versions()
	if err != nil {
		return err
	}
	if err := b.Library.SaveBooks(books); err != nil {
		return err
	}
	return b.Library.SaveVersions(versions)
}

// Progress reports a download: chapters stored out of the chapters to fetch.
type Progress func(done, total int)

// downloadBatch is how many chapters one request brings.
const downloadBatch = 20

// Download is a download of a version in progress, one batch of chapters at a time: the screen
// calls Step between two touches, so the reader never stays blocked for more than one request.
type Download struct {
	b       Bible
	slug    string
	books   []domain.Book
	missing []domain.ChapterRef
	total   int
	next    int // the first chapter of missing not fetched yet
	done    bool
}

// StartDownload plans the download of a version: the chapters it has not got yet.
func (b Bible) StartDownload(slug string) (*Download, error) {
	if b.Catalog == nil {
		return nil, ErrNoCatalog
	}
	books, err := b.Books()
	if err != nil {
		return nil, err
	}
	have, err := b.Library.StoredChapters(slug)
	if err != nil {
		return nil, err
	}
	d := &Download{b: b, slug: slug, books: books}
	for _, bk := range books {
		for ch := 1; ch <= bk.Chapters; ch++ {
			d.total++
			if ref := (domain.ChapterRef{Book: bk.ID, Chapter: ch}); !have[ref] {
				d.missing = append(d.missing, ref)
			}
		}
	}
	return d, nil
}

// Slug is the version being downloaded.
func (d *Download) Slug() string { return d.slug }

// Progress is the chapters stored out of the chapters of the version.
func (d *Download) Progress() (done, total int) { return d.total - len(d.missing) + d.next, d.total }

// Finished reports whether every chapter is stored and the version is marked as downloaded.
func (d *Download) Finished() bool { return d.done }

// Step brings the next batch of chapters and keeps it (the last step also marks the version as
// downloaded). A failed download goes on from where it stopped when started again.
func (d *Download) Step() error {
	if d.done {
		return nil
	}
	if d.next < len(d.missing) {
		end := d.next + downloadBatch
		if end > len(d.missing) {
			end = len(d.missing)
		}
		chapters, err := d.b.Catalog.Chapters(d.slug, d.books, d.missing[d.next:end])
		if err != nil {
			return err
		}
		if err := d.b.Library.SaveChapters(chapters); err != nil {
			return err
		}
		d.next = end
		if d.next < len(d.missing) {
			return nil
		}
	}
	stored, err := d.b.Library.StoredChapters(d.slug)
	if err != nil {
		return err
	}
	if len(stored) == 0 {
		return fmt.Errorf("bible: the catalog has no text for %s", d.slug)
	}
	if err := d.b.Library.MarkDownloaded(d.slug); err != nil {
		return err
	}
	d.done = true
	return nil
}

// Download brings every chapter of a version that is not on the reader yet and keeps it,
// a few at a time (so a failed or interrupted download goes on from where it stopped).
func (b Bible) Download(slug string, progress Progress) error {
	d, err := b.StartDownload(slug)
	if err != nil {
		return err
	}
	for !d.Finished() {
		if err := d.Step(); err != nil {
			return err
		}
		if progress != nil {
			progress(d.Progress())
		}
	}
	return nil
}

// Remove deletes the text of a version from the reader (what was read stays counted).
func (b Bible) Remove(slug string) error { return b.Library.DeleteVersion(slug) }

// Preferences are the person's choices; the first time they are made from the language of the reader.
func (b Bible) Preferences(deviceLang string) (domain.Preferences, error) {
	p, ok, err := b.Library.LoadPreferences()
	if err != nil {
		return p, err
	}
	if !ok {
		p = domain.DefaultPreferences()
		p.Lang = deviceLang
		p.VersionLang = strings.ToLower(deviceLang)
		p.Version = defaults[p.VersionLang]
		if p.Version == "" {
			p.Version, p.VersionLang = defaults["en"], "en"
		}
	}
	return p, nil
}

// SavePreferences keeps the person's choices.
func (b Bible) SavePreferences(p domain.Preferences) error { return b.Library.SavePreferences(p) }

// Chapter is the text of a chapter of a version: only that chapter is read from the database.
func (b Bible) Chapter(version string, book, chapter int) (domain.ChapterText, error) {
	return b.Library.Chapter(version, book, chapter)
}

// Seen records that the verses of a chapter were shown on the screen (first to last, by number
// of position in the chapter) and remembers the place as the one to resume from.
func (b Bible) Seen(book, chapter int, verses []int) error {
	if len(verses) == 0 {
		return nil
	}
	refs := make([]domain.Ref, len(verses))
	for i, n := range verses {
		refs[i] = domain.Ref{Book: book, Chapter: chapter, Verse: n}
	}
	if err := b.Library.MarkRead(refs, b.now()); err != nil {
		return err
	}
	return b.Library.SavePosition(domain.Position{Book: book, Chapter: chapter, Verse: verses[0]})
}

// Remember keeps the place the person opened without counting anything as read.
func (b Bible) Remember(p domain.Position) error { return b.Library.SavePosition(p) }

// Resume is the place to continue from (the zero Position when nothing was opened yet).
func (b Bible) Resume() (domain.Position, error) { return b.Library.Position() }

// Status is the reading progress of a version.
type Status struct {
	Read, Total int
	Percent     float64 // 0 to 100
}

// Progress is how much of the version has been read: the verses it has are the denominator,
// and a verse read in any version counts when this one has it too.
func (b Bible) Progress(version string) (Status, error) {
	read, total, err := b.Library.ReadCount(version)
	if err != nil {
		return Status{}, err
	}
	return Status{Read: read, Total: total, Percent: domain.Percent(read, total)}, nil
}

// Daily is the verse of the day: its place, its text in the version, and whether it was read.
type Daily struct {
	Ref  domain.Ref
	Text string
	Read bool
	// Festive is true when the verse is tied to the day (Christmas, Easter...).
	Festive bool
}

// VerseOfTheDay is the verse of a day. A holy day has its own verse; any other day has one
// drawn from the version, the same all day. The choice is kept, so it is the same after
// the version is changed or the app is reopened. ok is false while no version is downloaded.
func (b Bible) VerseOfTheDay(version string, day time.Time) (Daily, bool, error) {
	key := day.Format("2006-01-02")
	saved, found, err := b.Library.Daily(key)
	if err != nil {
		return Daily{}, false, err
	}
	festive, isFestive := domain.FestiveVerse(day)
	d := Daily{Ref: saved.Ref, Read: saved.Read, Festive: isFestive}
	if !found {
		d.Ref = festive
		if _, has, err := b.Library.VerseText(version, d.Ref); err != nil || !isFestive || !has {
			d.Festive = false
			if d.Ref, err = b.Library.PickVerse(version, domain.DailySeed(day)); err != nil {
				return Daily{}, false, err
			}
		}
		if !d.Ref.Valid() {
			return Daily{}, false, nil
		}
		if err := b.Library.SaveDaily(domain.DailyVerse{Day: key, Ref: d.Ref}); err != nil {
			return Daily{}, false, err
		}
	}
	text, has, err := b.Library.VerseText(version, d.Ref)
	if err != nil {
		return Daily{}, false, err
	}
	d.Text = text
	return d, has, nil
}

// DailyRead marks the verse of the day as read: it counts in the progress like any other.
func (b Bible) DailyRead(day time.Time) error {
	key := day.Format("2006-01-02")
	d, found, err := b.Library.Daily(key)
	if err != nil || !found {
		return err
	}
	if err := b.Library.MarkRead([]domain.Ref{d.Ref}, b.now()); err != nil {
		return err
	}
	d.Read = true
	return b.Library.SaveDaily(d)
}

// Search finds the verses of a version that have every word of the query.
func (b Bible) Search(version, query string) ([]domain.Hit, error) {
	return b.Library.Search(version, domain.Words(query), SearchLimit)
}

// ChapterPreviews are the openings of the chapters of a book, to tell them apart in a list.
func (b Bible) ChapterPreviews(version string, book int) (map[int]string, error) {
	return b.Library.Previews(version, book, 70)
}
