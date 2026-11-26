// Package presentation draws the Bible app: a Home page (the reading progress and the
// verse of the day), the Bible page (versions, search, books, chapters, verses and the
// reader) and the More menu (settings, About, Exit). It is built only from the toolkit's
// widgets and holds no rules: the use cases of internal/bible/application do.
//
// Memory: the screen keeps the list of books and of versions (a few KB) and, while reading,
// the text of one chapter. Nothing else is loaded; every page is rebuilt when it is shown.
package presentation

import (
	"embed"
	"fmt"
	"strings"
	"time"

	"go-pocketbook-base/internal/bible/application"
	"go-pocketbook-base/internal/bible/domain"
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/localization/i18n"
	"go-pocketbook-base/internal/toolkit/gfx"
	"go-pocketbook-base/internal/toolkit/widget"
)

//go:embed locales/*.json
var locales embed.FS

// NewTranslator loads the texts of the app (pt-BR first, English fallback) in lang.
func NewTranslator(lang string) *i18n.Bundle {
	tr := i18n.New()
	if err := tr.AddFS(locales, "locales"); err != nil {
		panic(err) // embedded files: an error is a programming mistake
	}
	tr.SetLang(lang)
	return tr
}

// Service is what the screen needs of the Bible context (application.Bible).
type Service interface {
	Books() ([]domain.Book, error)
	Versions() ([]domain.Version, error)
	Refresh() error
	Remove(slug string) error
	StartDownload(slug string) (*application.Download, error)
	Preferences(deviceLang string) (domain.Preferences, error)
	SavePreferences(p domain.Preferences) error
	Chapter(version string, book, chapter int) (domain.ChapterText, error)
	Seen(book, chapter int, verses []int) error
	Remember(p domain.Position) error
	Resume() (domain.Position, error)
	Progress(version string) (application.Status, error)
	VerseOfTheDay(version string, day time.Time) (application.Daily, bool, error)
	DailyRead(day time.Time) error
	Search(version, query string) ([]domain.Hit, error)
	ChapterPreviews(version string, book int) (map[int]string, error)
}

var _ Service = application.Bible{}

// Config is everything the screen needs.
type Config struct {
	Width, Height int
	// Doc is the document to draw on (a new one when nil).
	Doc *widget.Document
	// Chrome gives the About and Exit items of the More menu.
	Chrome   widget.AppChrome
	Keyboard widget.Keyboard
	Bible    Service
	// DeviceLang is the language of the reader: the interface starts in it.
	DeviceLang string
	// Speaker reads aloud (nil: the reader cannot).
	Speaker domain.Speaker
	// Flush paints pending changes now, before a slow call (nil: no flush).
	Flush func()
	// Now is the clock (time.Now when nil).
	Now func() time.Time
	// After runs a function on the event loop later (Env.After). A download is done in small steps
	// scheduled with it, so the screen stays usable; nil runs a task in one go.
	After func(d time.Duration, f func())
	// DeviceLangSource says where DeviceLang came from, and ClockSource where the time zone did
	// (for the Status window).
	DeviceLangSource, ClockSource string
}

// view is what the Bible page shows.
type view int

const (
	viewBooks view = iota
	viewChapters
	viewVerses
	viewReader
	viewSearch
)

// Screen is the Bible app's screen.
type Screen struct {
	Doc *widget.Document

	cfg   Config
	tr    *i18n.Bundle
	prefs domain.Preferences

	books    []domain.Book
	versions []domain.Version

	tabs   *widget.Tabs
	note   *widget.Label
	notice string

	// the task running (a download) and its strip above the footer
	job        *job
	strip      *widget.Panel
	stripLabel *widget.Label
	stripBar   *widget.Progress

	// the Bible page
	view      view
	book      int // the book open (id)
	chapter   int
	verse     int // the verse the reader opens at
	pickLang  string
	query     string
	hits      []domain.Hit
	bookPage  int
	extra     float64 // height the page gives up to a continue button and to the task strip
	listPage  int
	text      domain.ChapterText // the chapter being read: the only text kept in memory
	reader    *widget.ParagraphPager
	listening bool
}

// NewScreen builds the screen on the Home page.
func NewScreen(cfg Config) (*Screen, error) {
	s := &Screen{cfg: cfg}
	s.Doc = cfg.Doc
	if s.Doc == nil {
		s.Doc = widget.NewDocument(cfg.Width, cfg.Height)
		s.Doc.Keyboard = cfg.Keyboard
	}
	prefs, err := cfg.Bible.Preferences(cfg.DeviceLang)
	if err != nil {
		return nil, err
	}
	s.prefs = prefs.Normalize()
	s.tr = NewTranslator(s.uiLang())
	s.applyLook()
	if err := s.reload(); err != nil {
		return nil, err
	}
	s.build()
	return s, nil
}

// uiLang is the language of the interface: the one chosen, or the reader's.
func (s *Screen) uiLang() string {
	if s.prefs.Lang != "" {
		return s.prefs.Lang
	}
	return s.cfg.DeviceLang
}

func (s *Screen) t(key string, args ...interface{}) string { return s.tr.T(key, args...) }

// Prefs are the preferences in use.
func (s *Screen) Prefs() domain.Preferences { return s.prefs }

// Lang is the language of the interface.
func (s *Screen) Lang() string { return s.tr.Lang() }

func (s *Screen) now() time.Time {
	if s.cfg.Now != nil {
		return s.cfg.Now()
	}
	return time.Now()
}

// applyLook puts the typeface and the colors of the preferences in effect.
func (s *Screen) applyLook() {
	gfx.SetFamily(gfx.Family(s.prefs.Font))
	s.Doc.SetInvert(s.prefs.Invert)
}

// reload reads the books and versions again.
func (s *Screen) reload() error {
	books, err := s.cfg.Bible.Books()
	if err != nil {
		return err
	}
	versions, err := s.cfg.Bible.Versions()
	if err != nil {
		return err
	}
	s.books, s.versions = books, versions
	return nil
}

// currentVersion is the version in use (false when none is chosen or it is not in the list).
func (s *Screen) currentVersion() (domain.Version, bool) {
	for _, v := range s.versions {
		if v.Slug == s.prefs.Version {
			return v, true
		}
	}
	return domain.Version{}, false
}

// ready reports whether the current version has its text on the reader.
func (s *Screen) ready() bool {
	v, ok := s.currentVersion()
	return ok && v.Downloaded
}

// fail shows an error where the person can read it.
func (s *Screen) fail(err error) { s.say(s.t("bible.failed", err.Error())) }

// say shows a line in the footer.
func (s *Screen) say(text string) {
	s.notice = text
	if s.note != nil {
		s.note.Text = text
	}
	s.Doc.Invalidate()
}

func (s *Screen) flush() {
	if s.cfg.Flush != nil {
		s.cfg.Flush()
	}
}

// build makes the whole screen: the tabs, the More menu and the footer.
func (s *Screen) build() {
	s.tabs = widget.NewTabs(nil,
		&widget.Tab{ID: "home", Title: s.t("bible.menu.home"), Build: s.homePage, Mode: widget.TabRebuild},
		&widget.Tab{ID: "bible", Title: s.t("bible.menu.bible"), Build: s.biblePage, Mode: widget.TabRebuild})
	s.tabs.Bar.ID = "menubar"
	s.tabs.Bar.Menus[1].OnSelect = s.openBibleTab // a tap on the tab always starts at the books
	s.tabs.Bar.Menus = append(s.tabs.Bar.Menus, s.moreMenu())
	s.tabs.Body.Style.Padding = geom.XY(28, 20)
	s.tabs.Style.FlexGrow = 1

	s.note = widget.NewLabel(s.notice)
	s.note.Style.FontSize, s.note.Style.Color, s.note.Style.FlexGrow = widget.FontSmall, widget.Muted, 1
	version := widget.NewLabel("v" + s.cfg.Chrome.Version)
	version.Style.FontSize, version.Style.Color = widget.FontSmall, widget.Muted
	version.Style.TextAlign = gfx.AlignEnd
	footer := widget.NewRow(24, s.note, version)
	footer.Style.Padding = geom.XY(28, 8)

	root := widget.NewColumn(0, s.tabs, s.buildStrip(), footer)
	root.ID, root.Style.Background = "root", widget.Paper
	s.Doc.SetRoot(root)
}

// moreMenu is the More menu: Settings, then the About and Exit of every app.
func (s *Screen) moreMenu() *widget.Menu {
	items := []widget.MenuItem{
		{Label: s.t("bible.menu.settings"), Icon: gfx.IconMenu, OnSelect: s.OpenSettings},
		{Label: s.t("bible.menu.versions"), OnSelect: s.OpenVersions},
		{Label: s.t("bible.menu.status"), OnSelect: s.OpenStatus},
	}
	if chrome := s.cfg.Chrome; chrome.Enabled() {
		items = append(items, widget.MenuItem{Separator: true})
		items = append(items, chrome.Bar(s.Doc).Menus[0].Items...)
	}
	return &widget.Menu{Label: s.t("bible.menu.more"), Items: items}
}

// Show switches to the Home (0) or the Bible (1) page.
func (s *Screen) Show(i int) { s.tabs.Show(i) }

// Page is the page shown (0 Home, 1 Bible).
func (s *Screen) Page() int { return s.tabs.Current() }

// openBibleTab is what the Bible tab does when tapped: the list of books, whatever was open.
func (s *Screen) openBibleTab() {
	s.view, s.query, s.hits, s.listPage = viewBooks, "", nil, 0
	s.tabs.Show(1)
	s.Doc.Invalidate()
}

// rebuildBible shows the Bible page again after the navigation changed.
func (s *Screen) rebuildBible() {
	s.tabs.Show(1)
	s.Doc.Invalidate()
}

// Key handles the page keys: in the reader they turn the page of the text, anywhere else
// they move between the pages of the app. It returns false when it did nothing.
func (s *Screen) Key(previous bool) bool {
	if s.Doc.HasModal() {
		return false
	}
	if s.tabs.Current() == 1 && s.view == viewReader && s.reader != nil {
		if previous {
			s.readerPrevious()
		} else {
			s.readerNext()
		}
		return true
	}
	if previous {
		s.tabs.Move(-1)
	} else {
		s.tabs.Move(1)
	}
	return true
}

// bookByID finds a book of the list.
func (s *Screen) bookByID(id int) (domain.Book, bool) {
	for _, b := range s.books {
		if b.ID == id {
			return b, true
		}
	}
	return domain.Book{}, false
}

// bookName is a book's name in the language of the version being read, in the language of the
// interface when the version's has none, and in English at last.
func (s *Screen) bookName(b domain.Book) string {
	for _, lang := range []string{strings.ToLower(s.prefs.VersionLang), strings.ToLower(s.tr.Lang())} {
		if n := b.Names[lang]; n != "" {
			return n
		}
	}
	return b.Name("en")
}

// refText is "João 3:16".
func (s *Screen) refText(r domain.Ref) string {
	name := fmt.Sprint(r.Book)
	if b, ok := s.bookByID(r.Book); ok {
		name = s.bookName(b)
	}
	return fmt.Sprintf("%s %d:%d", name, r.Chapter, r.Verse)
}

// percentText writes a percentage with the decimal mark of the language.
func (s *Screen) percentText(p float64) string {
	text := fmt.Sprintf("%.1f%%", p)
	if s.tr.Lang() == "pt-BR" {
		text = strings.Replace(text, ".", ",", 1)
	}
	return text
}

// number writes an integer with thousands separators of the language.
func (s *Screen) number(n int) string {
	sep := ","
	if s.tr.Lang() == "pt-BR" {
		sep = "."
	}
	text := fmt.Sprint(n)
	for i := len(text) - 3; i > 0; i -= 3 {
		text = text[:i] + sep + text[i:]
	}
	return text
}

// muted is a small gray label.
func muted(text string) *widget.Label {
	l := widget.NewLabel(text)
	l.Style.Color, l.Style.FontSize = widget.Muted, widget.FontSmall
	return l
}

// heading is a bold label.
func heading(text string) *widget.Label {
	l := widget.NewLabel(text)
	l.Style.Bold, l.Style.FontSize = true, widget.FontBody+4
	return l
}
