// Package midvash is the domain.Catalog over the Midvash API (https://api.midvash.com),
// a free Bible API with no key: the books, the versions and whole chapters, many of
// them in one request (/v1/passages), so a version comes down in a few dozen calls.
package midvash

import (
	"fmt"
	"net/url"
	"strings"

	"go-pocketbook-base/internal/bible/domain"
	"go-pocketbook-base/internal/kernel/webjson"
)

// DefaultBase is the address of the API.
const DefaultBase = "https://api.midvash.com/v1"

// BatchSize is how many chapters one request asks for (the API takes up to 50; a smaller
// batch keeps each answer, held in memory while it is stored, small).
const BatchSize = 20

// Client implements domain.Catalog.
type Client struct {
	HTTP webjson.Doer
	Base string // DefaultBase when empty
}

var _ domain.Catalog = Client{}

func (c Client) web() webjson.Client {
	base := c.Base
	if base == "" {
		base = DefaultBase
	}
	return webjson.Client{HTTP: c.HTTP, Base: base}
}

type booksAnswer struct {
	Data []struct {
		ID        int               `json:"id"`
		Name      map[string]string `json:"name"`
		Slug      map[string]string `json:"slug"`
		Chapters  int               `json:"chapters"`
		Testament string            `json:"testament"`
	} `json:"data"`
}

// Books implements domain.Catalog.
func (c Client) Books() ([]domain.Book, error) {
	var a booksAnswer
	if err := c.web().Do("GET", "/books", nil, &a); err != nil {
		return nil, err
	}
	out := make([]domain.Book, 0, len(a.Data))
	for _, b := range a.Data {
		names := make(map[string]string, len(b.Name))
		for lang, n := range b.Name {
			names[strings.ToLower(lang)] = n
		}
		out = append(out, domain.Book{ID: b.ID, Slug: b.Slug["en"], Names: names, Chapters: b.Chapters, New: b.Testament == "new"})
	}
	return out, nil
}

type versionsAnswer struct {
	Data []struct {
		Slug          string `json:"slug"`
		Name          string `json:"name"`
		ShortName     string `json:"shortName"`
		Language      string `json:"language"`
		TotalBooks    int    `json:"totalBooks"`
		TotalChapters int    `json:"totalChapters"`
	} `json:"data"`
}

// Versions implements domain.Catalog.
func (c Client) Versions() ([]domain.Version, error) {
	var a versionsAnswer
	if err := c.web().Do("GET", "/versions", nil, &a); err != nil {
		return nil, err
	}
	out := make([]domain.Version, 0, len(a.Data))
	for _, v := range a.Data {
		out = append(out, domain.Version{
			Slug: v.Slug, Name: v.Name, ShortName: v.ShortName, Language: strings.ToLower(v.Language),
			Books: v.TotalBooks, Chapters: v.TotalChapters,
		})
	}
	return out, nil
}

type passagesAnswer struct {
	Data []struct {
		Book    string   `json:"book"`
		Chapter int      `json:"chapter"`
		Verse   int      `json:"verse"`
		Verses  []string `json:"verses"`
	} `json:"data"`
}

// Chapters implements domain.Catalog: at most BatchSize chapters per request.
func (c Client) Chapters(version string, books []domain.Book, refs []domain.ChapterRef) ([]domain.ChapterText, error) {
	slugs := make(map[int]string, len(books))
	ids := make(map[string]int, len(books))
	for _, b := range books {
		slugs[b.ID], ids[b.Slug] = b.Slug, b.ID
	}
	var out []domain.ChapterText
	for start := 0; start < len(refs); start += BatchSize {
		end := start + BatchSize
		if end > len(refs) {
			end = len(refs)
		}
		names, err := passageNames(slugs, refs[start:end])
		if err != nil {
			return nil, err
		}
		query := url.Values{"refs": {strings.Join(names, ",")}, "version": {version}}
		var a passagesAnswer
		if err := c.web().Do("GET", "/passages?"+query.Encode(), nil, &a); err != nil {
			return nil, err
		}
		for _, p := range a.Data {
			if book, ok := ids[p.Book]; ok && len(p.Verses) > 0 { // else: a chapter this version does not have
				out = append(out, chapterOf(version, book, p.Chapter, p.Verse, p.Verses))
			}
		}
	}
	return out, nil
}

// passageNames writes the references of a batch the way the API asks for them ("john 3").
func passageNames(slugs map[int]string, refs []domain.ChapterRef) ([]string, error) {
	names := make([]string, 0, len(refs))
	for _, r := range refs {
		slug, ok := slugs[r.Book]
		if !ok {
			return nil, fmt.Errorf("midvash: unknown book %d", r.Book)
		}
		names = append(names, fmt.Sprintf("%s %d", slug, r.Chapter))
	}
	return names, nil
}

// chapterOf builds a chapter from the verses of the API: they start at verse first (1 when the
// answer does not say) and a blank one is left out.
func chapterOf(version string, book, chapter, first int, texts []string) domain.ChapterText {
	if first < 1 {
		first = 1
	}
	ch := domain.ChapterText{Version: version, Book: book, Chapter: chapter, Verses: make([]domain.Verse, 0, len(texts))}
	for i, text := range texts {
		if text = strings.TrimSpace(text); text != "" {
			ch.Verses = append(ch.Verses, domain.Verse{Number: first + i, Text: text})
		}
	}
	return ch
}
