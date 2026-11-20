// Package domain is the vocabulary of the Bible app: books, versions (translations),
// the place of a verse, what the person has read and how they like to read.
// It depends on nothing but the kernel.
package domain

import (
	"sort"
	"strings"
)

// Book is one of the books of the Bible. IDs follow the canonical order (1 Genesis
// ... 66 Revelation), the same in every version and language, which is what lets
// the reading progress be shared by every version.
type Book struct {
	ID       int
	Slug     string            // the English slug the API takes in its addresses ("john")
	Names    map[string]string // language tag in lower case ("pt-br", "en") -> name
	Chapters int
	New      bool // New Testament
}

// Name is the book's name in a language, English when it has no name in it and the slug as the last resort.
func (b Book) Name(lang string) string {
	if n := b.Names[strings.ToLower(lang)]; n != "" {
		return n
	}
	if n := b.Names["en"]; n != "" {
		return n
	}
	return b.Slug
}

// SortAlphabetically orders books by their name in a language (accents and case ignored).
func SortAlphabetically(books []Book, lang string) []Book {
	out := append([]Book(nil), books...)
	sort.SliceStable(out, func(i, j int) bool {
		return Fold(out[i].Name(lang)) < Fold(out[j].Name(lang))
	})
	return out
}

// InCanonicalOrder orders books as they appear in the Bible.
func InCanonicalOrder(books []Book) []Book {
	out := append([]Book(nil), books...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// Version is a translation of the Bible.
type Version struct {
	Slug      string
	Name      string
	ShortName string
	Language  string // language tag in lower case ("pt-br", "en", "la")
	Books     int
	Chapters  int
	// Downloaded is true once every chapter is stored on the reader.
	Downloaded bool
	// Verses is how many verses are stored (0 until something is downloaded).
	Verses int
}

// Label is the short text a list shows for a version.
func (v Version) Label() string {
	if v.Name == "" {
		return v.ShortName
	}
	if v.ShortName == "" || strings.EqualFold(v.ShortName, v.Name) {
		return v.Name
	}
	return v.Name + " (" + v.ShortName + ")"
}

// Ref is the place of a verse. It does not name a version: the same place is the
// same verse in any of them.
type Ref struct{ Book, Chapter, Verse int }

// Valid reports whether the reference points somewhere.
func (r Ref) Valid() bool { return r.Book > 0 && r.Chapter > 0 && r.Verse > 0 }

// Verse is a verse with its text.
type Verse struct {
	Number int
	Text   string
}

// ChapterText is the verses of one chapter of one version.
type ChapterText struct {
	Version string
	Book    int
	Chapter int
	Verses  []Verse
}

// Hit is one result of a search.
type Hit struct {
	Ref  Ref
	Text string
}

// Language is a language the versions come in.
type Language struct {
	Tag      string // "pt-br"
	Versions int
}

// LanguagesOf lists the languages of a set of versions, with how many each has.
func LanguagesOf(versions []Version) []Language {
	count := map[string]int{}
	for _, v := range versions {
		count[v.Language]++
	}
	out := make([]Language, 0, len(count))
	for tag, n := range count {
		out = append(out, Language{Tag: tag, Versions: n})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Tag < out[j].Tag })
	return out
}

// VersionsIn are the versions in a language, by name.
func VersionsIn(versions []Version, lang string) []Version {
	var out []Version
	for _, v := range versions {
		if strings.EqualFold(v.Language, lang) {
			out = append(out, v)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return Fold(out[i].Name) < Fold(out[j].Name) })
	return out
}

// Position is where the person stopped reading: the book, chapter and verse last opened.
type Position struct {
	Book, Chapter, Verse int
}

// Valid reports whether there is a place to resume from.
func (p Position) Valid() bool { return p.Book > 0 && p.Chapter > 0 }

// Percent is read out of total as a number from 0 to 100 (0 when there is nothing to read).
func Percent(read, total int) float64 {
	if total <= 0 || read <= 0 {
		return 0
	}
	if read >= total {
		return 100
	}
	return float64(read) * 100 / float64(total)
}

// At is the place of a verse (Ref is a plain struct; this reads better outside the package,
// where `go vet` rejects unkeyed fields).
func At(book, chapter, verse int) Ref { return Ref{Book: book, Chapter: chapter, Verse: verse} }

// Chapter is the place of a chapter.
func Chapter(book, chapter int) ChapterRef { return ChapterRef{Book: book, Chapter: chapter} }
