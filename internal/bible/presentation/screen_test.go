package presentation

import (
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go-pocketbook-base/internal/bible/application"
	"go-pocketbook-base/internal/bible/domain"
	"go-pocketbook-base/internal/bible/infrastructure/sqlstore"
	"go-pocketbook-base/internal/storage"
	"go-pocketbook-base/internal/toolkit/gfx"
	"go-pocketbook-base/internal/toolkit/widget"
)

// fakeCatalog is a small Bible: three books of three chapters with 30 long verses each.
type fakeCatalog struct {
	fail error
	// chapters is how many chapters each book has (3 when zero); more than 20 books-worth makes a
	// download take several requests.
	chapters int
}

func (f *fakeCatalog) per() int {
	if f.chapters > 0 {
		return f.chapters
	}
	return 3
}

func (f *fakeCatalog) Books() ([]domain.Book, error) {
	if f.fail != nil {
		return nil, f.fail
	}
	return []domain.Book{
		{ID: 1, Slug: "genesis", Names: map[string]string{"en": "Genesis", "pt-br": "Gênesis"}, Chapters: f.per()},
		{ID: 19, Slug: "psalms", Names: map[string]string{"en": "Psalms", "pt-br": "Salmos"}, Chapters: f.per()},
		{ID: 43, Slug: "john", Names: map[string]string{"en": "John", "pt-br": "João"}, Chapters: f.per(), New: true},
	}, nil
}

func (f *fakeCatalog) Versions() ([]domain.Version, error) {
	return []domain.Version{
		{Slug: "almeida-livre", Name: "Bíblia Livre", ShortName: "BL", Language: "pt-br", Books: 3, Chapters: 9},
		{Slug: "kjv", Name: "King James Version", ShortName: "KJV", Language: "en", Books: 3, Chapters: 9},
		{Slug: "web", Name: "World English Bible", ShortName: "WEB", Language: "en", Books: 3, Chapters: 9},
	}, nil
}

func (f *fakeCatalog) Chapters(version string, _ []domain.Book, refs []domain.ChapterRef) ([]domain.ChapterText, error) {
	if f.fail != nil {
		return nil, f.fail
	}
	var out []domain.ChapterText
	for _, r := range refs {
		c := domain.ChapterText{Version: version, Book: r.Book, Chapter: r.Chapter}
		for v := 1; v <= 30; v++ {
			c.Verses = append(c.Verses, domain.Verse{Number: v, Text: fmt.Sprintf("Versículo %d do capítulo %d do livro %d na versão %s, com palavras bastantes para ocupar mais de uma linha da página de leitura.", v, r.Chapter, r.Book, version)})
		}
		out = append(out, c)
	}
	return out, nil
}

type fakeSpeaker struct {
	available bool
	spoke     string
	stopped   int
}

func (f *fakeSpeaker) Available() bool       { return f.available }
func (f *fakeSpeaker) Speak(t string) error  { f.spoke = t; return nil }
func (f *fakeSpeaker) Stop()                 { f.stopped++ }
func (f *fakeSpeaker) wasAskedToSpeak() bool { return f.spoke != "" }

var today = time.Date(2026, 11, 19, 8, 0, 0, 0, time.UTC)

// world is everything a screen is built over.
type world struct {
	t       *testing.T
	db      *storage.Store
	bible   application.Bible
	catalog *fakeCatalog
	speaker *fakeSpeaker
	flushes int
	async   bool     // the screen gets a timer: a task runs a step at a time
	later   []func() // what the screen asked to run later
}

// after is the Config.After of an async world: it only queues.
func (w *world) after(_ time.Duration, f func()) { w.later = append(w.later, f) }

// runNext runs the oldest queued function and reports whether there was one.
func (w *world) runNext() bool {
	if len(w.later) == 0 {
		return false
	}
	f := w.later[0]
	w.later = w.later[1:]
	f()
	return true
}

func newWorld(t *testing.T) *world {
	t.Helper()
	t.Cleanup(func() { gfx.SetFamily(gfx.FamilyGo) })
	db, err := storage.Open(filepath.Join(t.TempDir(), "bible.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	lib, err := sqlstore.New(db)
	if err != nil {
		t.Fatal(err)
	}
	cat := &fakeCatalog{}
	return &world{t: t, db: db, catalog: cat, speaker: &fakeSpeaker{},
		bible: application.Bible{Catalog: cat, Library: lib, Now: func() time.Time { return today }}}
}

// screen opens the app over the world (again, as the next run would).
func (w *world) screen(lang string) *Screen {
	w.t.Helper()
	var after func(time.Duration, func())
	if w.async {
		after = w.after
	}
	s, err := NewScreen(Config{
		Width: 1404, Height: 1872, Bible: w.bible, DeviceLang: lang, Speaker: w.speaker, After: after,
		DeviceLangSource: "InkView global.cfg language=pt", ClockSource: "Buenos Aires",
		Chrome: widget.AppChrome{Title: "Bíblia", Version: "1.2.3", LabelAbout: "Sobre", LabelExit: "Sair", LabelVersion: "versão %s", Exit: func() {}},
		Flush:  func() { w.flushes++ }, Now: func() time.Time { return today },
	})
	if err != nil {
		w.t.Fatal(err)
	}
	s.Doc.Layout()
	return s
}

// ready is a world with the Portuguese version already downloaded.
func (w *world) ready() {
	w.t.Helper()
	if err := w.bible.Refresh(); err != nil {
		w.t.Fatal(err)
	}
	if err := w.bible.Download("almeida-livre", nil); err != nil {
		w.t.Fatal(err)
	}
}

func byID(t *testing.T, s *Screen, id string) widget.Widget {
	t.Helper()
	w := findID(s, id)
	if w == nil {
		t.Fatalf("no element %q", id)
	}
	return w
}

func findID(s *Screen, id string) widget.Widget {
	var found widget.Widget
	s.Doc.Walk(func(w widget.Widget) bool {
		if w.Elem().ID == id {
			found = w
		}
		return true
	})
	return found
}

func tap(t *testing.T, s *Screen, id string) {
	t.Helper()
	s.Doc.Layout()
	c := byID(t, s, id).Elem().Bounds.Center()
	s.Doc.HandlePointer(widget.PointerDown, c.X*s.Doc.Scale, c.Y*s.Doc.Scale)
	s.Doc.HandlePointer(widget.PointerUp, c.X*s.Doc.Scale, c.Y*s.Doc.Scale)
	s.Doc.Layout()
}

func progressOf(t *testing.T, s *Screen) float64 {
	t.Helper()
	return byID(t, s, "progress").(*widget.Progress).Value
}

func snap(t *testing.T, s *Screen, name string) {
	t.Helper()
	dir := os.Getenv("BIBLE_PREVIEW_DIR")
	if dir == "" {
		return
	}
	s.Doc.Layout()
	f, err := os.Create(filepath.Join(dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, s.Doc.Render()); err != nil {
		t.Fatal(err)
	}
}

func TestEveryTextIsTranslatedInBothLanguages(t *testing.T) {
	tr := NewTranslator("pt-BR")
	for _, lang := range []string{"pt-BR", "en"} {
		if missing := tr.Missing(lang); len(missing) != 0 {
			t.Errorf("%s lacks: %v", lang, missing)
		}
	}
}

func TestTheFirstRunOffersToDownloadTheBibleOfTheReadersLanguageInOneTap(t *testing.T) {
	w := newWorld(t)
	s := w.screen("pt-BR")
	snap(t, s, "01-first-run")
	if s.Page() != 0 || s.Lang() != "pt-BR" || s.Prefs().Version != "almeida-livre" {
		t.Fatalf("page=%d lang=%s prefs=%+v", s.Page(), s.Lang(), s.Prefs())
	}
	if findID(s, "progress") != nil || findID(s, "get-bible") == nil {
		t.Fatal("with nothing downloaded the Home asks to download")
	}
	tap(t, s, "get-bible")
	if !s.ready() {
		t.Fatal("the default version is downloaded and in use")
	}
	if progressOf(t, s) != 0 || findID(s, "votd") == nil {
		t.Error("the Home now shows the progress (0%) and the verse of the day")
	}
	snap(t, s, "02-home")

	// a second run opens straight on the Home with everything remembered
	again := w.screen("pt-BR")
	if !again.ready() || findID(again, "get-bible") != nil {
		t.Error("the version stays downloaded")
	}
}

func TestADownloadFailureIsShownAndKeepsTheScreenUsable(t *testing.T) {
	w := newWorld(t)
	w.catalog.fail = errors.New("sem rede")
	s := w.screen("pt-BR")
	tap(t, s, "get-bible")
	if !strings.Contains(s.notice, "sem rede") || s.ready() {
		t.Errorf("notice = %q ready=%v", s.notice, s.ready())
	}
	if s.Doc.HasModal() {
		t.Error("the progress modal must close on failure")
	}
	tap(t, s, "open-bible")
	if findID(s, "refresh-versions") == nil {
		t.Error("the Bible page offers to fetch the list again")
	}
	w.catalog.fail = nil
	tap(t, s, "refresh-versions")
	if findID(s, "combo-version") == nil || len(s.versions) != 3 {
		t.Errorf("after the refresh the pickers show: %d versions", len(s.versions))
	}
}

func TestTheVerseOfTheDayIsReadWithACheckAndCountsInTheProgress(t *testing.T) {
	w := newWorld(t)
	w.ready()
	s := w.screen("pt-BR")
	if findID(s, "votd-done") != nil || findID(s, "votd-read") == nil {
		t.Fatal("not read yet: the check box is there")
	}
	card := byID(t, s, "votd").Elem()
	if card.Bounds.H == 0 {
		t.Error("the card is laid out")
	}
	tap(t, s, "votd-read")
	if findID(s, "votd-read") != nil || findID(s, "votd-done") == nil {
		t.Error("the check box is replaced by the done mark")
	}
	if progressOf(t, s) <= 0 {
		t.Error("the verse counts in the progress")
	}
	snap(t, s, "03-home-read")
	if again := w.screen("pt-BR"); findID(again, "votd-done") == nil {
		t.Error("it stays read after reopening")
	}
}

func TestReadingFlowBookChapterVerseThenTheVersesOnScreenCountAsRead(t *testing.T) {
	w := newWorld(t)
	w.ready()
	s := w.screen("pt-BR")
	tap(t, s, "menubar") // harmless: the bar itself
	s.Show(1)
	s.Doc.Layout()
	snap(t, s, "04-books")
	if findID(s, "book-43") == nil {
		t.Fatal("the books are listed")
	}
	tap(t, s, "book-43")
	snap(t, s, "05-chapters")
	if findID(s, "chapter-2") == nil {
		t.Fatal("the chapters of John")
	}
	if b := byID(t, s, "chapter-2").(*widget.Button); !strings.Contains(b.Text, "Versículo 1") {
		t.Errorf("the chapter shows a preview of its start: %q", b.Text)
	}
	tap(t, s, "chapter-2")
	snap(t, s, "06-verses")
	if findID(s, "verse-17") == nil || findID(s, "read-chapter") == nil {
		t.Fatal("the verse numbers and the read button")
	}
	before, _ := w.bible.Progress("almeida-livre")
	tap(t, s, "verse-17")
	snap(t, s, "07-reader")
	if s.reader == nil {
		t.Fatal("the reader is open")
	}
	after, _ := w.bible.Progress("almeida-livre")
	first, last, _ := s.reader.Visible()
	shown := last - first + 1
	if after.Read-before.Read != shown {
		t.Errorf("the %d verses on the screen count as read: %d -> %d", shown, before.Read, after.Read)
	}
	if first+1 > 17 || last+1 < 17 {
		t.Errorf("verse 17 is on the page that opened: %d..%d", first+1, last+1)
	}
	pos, _ := w.bible.Resume()
	if pos.Book != 43 || pos.Chapter != 2 {
		t.Errorf("the place is remembered: %+v", pos)
	}

	// turning the page counts the verses of the next page too
	s.OpenReader(1, 3, 1)
	read, _ := w.bible.Progress("almeida-livre")
	s.reader.Next()
	if now, _ := w.bible.Progress("almeida-livre"); s.reader.Page != 1 || now.Read <= read.Read {
		t.Errorf("the next page counts too: page %d, %d -> %d", s.reader.Page, read.Read, now.Read)
	}

	// a new run goes on from where the person stopped
	s.OpenReader(43, 2, 17)
	again := w.screen("pt-BR")
	if findID(again, "continue") == nil {
		t.Fatal("the Home offers to continue")
	}
	tap(t, again, "continue")
	if again.reader == nil || again.book != 43 || again.chapter != 2 {
		t.Errorf("resumed at %d:%d", again.book, again.chapter)
	}
	// neighbour chapters, across books
	tap(t, again, "chapter-next")
	if again.chapter != 3 {
		t.Errorf("next chapter: %d", again.chapter)
	}
	if byID(t, again, "chapter-next").Elem().Disabled != true {
		t.Error("John 3 is the last chapter of this little Bible")
	}
	tap(t, again, "chapter-prev")
	tap(t, again, "chapter-prev")
	tap(t, again, "chapter-prev")
	if again.book != 19 || again.chapter != 3 {
		t.Errorf("before John 1 comes the last chapter of Psalms: %d:%d", again.book, again.chapter)
	}
	tap(t, again, "back")
	if again.view != viewVerses {
		t.Error("back leaves the reader for the verse numbers")
	}
}

func TestTheProgressFollowsTheVersionInUse(t *testing.T) {
	w := newWorld(t)
	w.ready()
	if err := w.bible.Download("kjv", nil); err != nil {
		t.Fatal(err)
	}
	if err := w.bible.Seen(1, 1, []int{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	s := w.screen("pt-BR")
	pt := progressOf(t, s)
	s.pickVersion("kjv")
	s.Show(0)
	if en := progressOf(t, s); en != pt || pt <= 0 {
		t.Errorf("a verse read in one version counts in the other: %v vs %v", pt, en)
	}
}

func TestBooksCanBeListedAlphabeticallyAndTheChoiceIsKept(t *testing.T) {
	w := newWorld(t)
	w.ready()
	s := w.screen("pt-BR")
	s.Show(1)
	s.Doc.Layout()
	order := func() []string {
		var ids []string
		s.Doc.Walk(func(x widget.Widget) bool {
			if id := x.Elem().ID; strings.HasPrefix(id, "book-") {
				ids = append(ids, id)
			}
			return true
		})
		return ids
	}
	if got := order(); len(got) != 3 || got[0] != "book-1" {
		t.Fatalf("original order: %v", got)
	}
	tap(t, s, "books-order")
	if got := order(); got[0] != "book-1" || got[1] != "book-43" || got[2] != "book-19" {
		t.Errorf("Gênesis, João, Salmos: %v", got)
	}
	if again := w.screen("pt-BR"); !again.Prefs().Alphabetical {
		t.Error("the order is remembered")
	}
	tap(t, s, "books-order")
	if got := order(); got[1] != "book-19" {
		t.Errorf("back to the original order: %v", got)
	}
}

func TestPickingALanguageAndANotDownloadedVersionDownloadsItAfterAsking(t *testing.T) {
	w := newWorld(t)
	w.ready()
	s := w.screen("pt-BR")
	s.Show(1)
	snap(t, s, "08-bar")
	lang := byID(t, s, "combo-language").(*widget.ComboBox)
	if len(lang.Options) != 2 || lang.Value() != "pt-br" {
		t.Fatalf("languages: %+v value=%q", lang.Options, lang.Value())
	}
	s.pickLanguage("en")
	versions := byID(t, s, "combo-version").(*widget.ComboBox)
	if len(versions.Options) != 2 || versions.Value() != "" || !strings.Contains(versions.Options[0].Label, "baixar") {
		t.Fatalf("English versions, none chosen, not downloaded: %+v value=%q", versions.Options, versions.Value())
	}
	if !s.ready() || s.prefs.Version != "almeida-livre" {
		t.Error("the version in use does not change until another is chosen")
	}
	s.pickVersion("kjv")
	if !s.Doc.HasModal() {
		t.Fatal("it asks before downloading")
	}
	modal := byID(t, s, "modal-download").(*widget.Modal)
	modal.Confirm()
	if s.prefs.Version != "kjv" || !s.ready() || s.prefs.VersionLang != "en" {
		t.Errorf("the version is downloaded and in use: %+v", s.prefs)
	}
	if v := byID(t, s, "version-current").(*widget.Label).Text; !strings.Contains(v, "KJV") {
		t.Errorf("the bar names the version: %q", v)
	}
	// refusing a download keeps the version in use
	s.pickVersion("web")
	byID(t, s, "modal-download").(*widget.Modal).Cancel()
	if s.prefs.Version != "kjv" {
		t.Errorf("cancelled: %s", s.prefs.Version)
	}
	s.pickVersion("kjv") // already on the reader: just used, no question
	s.pickVersion("nope")
	if s.Doc.HasModal() {
		t.Error("no question for a version that is on the reader or unknown")
	}
	if got := w.screen("pt-BR").Prefs().Version; got != "kjv" {
		t.Errorf("the choice survives a restart: %s", got)
	}
}

func TestSearchFindsVersesAndOpensThemInTheReader(t *testing.T) {
	w := newWorld(t)
	w.ready()
	s := w.screen("pt-BR")
	s.Show(1)
	s.Search("LIVRO 19")
	snap(t, s, "09-search")
	if len(s.hits) != application.SearchLimit {
		t.Fatalf("every verse of Psalms has the words, the search stops at its limit: %d", len(s.hits))
	}
	tap(t, s, "hit-4")
	if s.reader == nil || s.book != 19 || s.chapter != 1 {
		t.Fatalf("opened at %d:%d", s.book, s.chapter)
	}
	s.Search("")
	if s.view != viewBooks {
		t.Error("an empty search goes back to the books")
	}
	s.Search("palavra que não existe")
	tap(t, s, "search-clear")
	if s.view != viewBooks || s.query != "" {
		t.Errorf("clear: view=%v query=%q", s.view, s.query)
	}
	s.Search("x")
	if len(s.hits) != 0 {
		t.Error("no hits expected")
	}
}

func TestSettingsChangeTheLookAndAreKeptBetweenRuns(t *testing.T) {
	w := newWorld(t)
	w.ready()
	s := w.screen("pt-BR")
	s.OpenSettings()
	snap(t, s, "10-settings")
	m := byID(t, s, "modal-settings").(*widget.Modal)
	size := byID(t, s, "settings-size").(*widget.Slider)
	size.Value = 46
	size.OnInput(&widget.Event{})
	if l := byID(t, s, "settings-size-label").(*widget.Label).Text; !strings.Contains(l, "46") {
		t.Errorf("the label follows the slider: %q", l)
	}
	byID(t, s, "settings-font").(*widget.ComboBox).SetValue("mono")
	byID(t, s, "settings-invert").(*widget.Toggle).Checked = true
	byID(t, s, "settings-tts").(*widget.Toggle).Checked = true
	byID(t, s, "settings-titles").(*widget.Toggle).Checked = false
	m.Confirm()

	p := s.Prefs()
	if p.FontSize != 46 || p.Font != "mono" || !p.Invert || !p.Speech || p.ChapterTitles {
		t.Fatalf("prefs: %+v", p)
	}
	if gfx.CurrentFamily() != gfx.FamilyMono || !s.Doc.Invert {
		t.Error("the typeface and the high contrast are in effect")
	}
	if img := s.Doc.Render(); img.Pix[0] != 0 {
		t.Errorf("inverted paper is black: %d", img.Pix[0])
	}

	gfx.SetFamily(gfx.FamilyGo)
	again := w.screen("pt-BR")
	if q := again.Prefs(); q != p || gfx.CurrentFamily() != gfx.FamilyMono || !again.Doc.Invert {
		t.Errorf("after reopening: %+v family=%s invert=%v", q, gfx.CurrentFamily(), again.Doc.Invert)
	}

	// without previews the chapters are plain numbers
	again.Show(1)
	tap(t, again, "book-43")
	if b := byID(t, again, "chapter-1").(*widget.Button); b.Text != "1" {
		t.Errorf("a plain number: %q", b.Text)
	}
}

func TestTheLanguageOfTheAppFollowsTheReaderAndCanBeChanged(t *testing.T) {
	w := newWorld(t)
	w.ready()
	s := w.screen("en")
	if s.Lang() != "en" || s.tabs.Bar.Menus[0].Label != "Home" || s.tabs.Bar.Menus[2].Label != "More" {
		t.Fatalf("the reader's language: %s / %q", s.Lang(), s.tabs.Bar.Menus[0].Label)
	}
	if s.Prefs().Version != "almeida-livre" {
		t.Logf("a portuguese version was already chosen in this world: %s", s.Prefs().Version)
	}
	p := s.Prefs()
	p.Lang = "pt-BR"
	s.ApplyPreferences(p)
	if s.Lang() != "pt-BR" || s.tabs.Bar.Menus[0].Label != "Início" || s.tabs.Bar.Menus[1].Label != "Bíblia" || s.tabs.Bar.Menus[2].Label != "Mais" {
		t.Errorf("rebuilt in Portuguese: %s %q", s.Lang(), s.tabs.Bar.Menus[0].Label)
	}
	if again := w.screen("en"); again.Lang() != "pt-BR" {
		t.Errorf("the choice beats the reader's language: %s", again.Lang())
	}
}

func TestTheMoreMenuKeepsSettingsAboutAndExit(t *testing.T) {
	w := newWorld(t)
	s := w.screen("pt-BR")
	items := s.tabs.Bar.Menus[2].Items
	var labels []string
	for _, it := range items {
		labels = append(labels, it.Label)
	}
	if strings.Join(labels, "|") != "Configurações|Versões baixadas|Status||Sobre||Sair" {
		t.Errorf("the More menu: %q", labels)
	}
	items[4].OnSelect() // About
	if !s.Doc.HasModal() {
		t.Error("About opens a modal")
	}
	byID(t, s, "modal-about").(*widget.Modal).Close()
	items[0].OnSelect()
	if findID(s, "modal-settings") == nil {
		t.Error("Settings opens its modal")
	}
}

func TestReadingAloudNeedsTheSettingAndASpeaker(t *testing.T) {
	w := newWorld(t)
	w.ready()
	s := w.screen("pt-BR")
	s.OpenReader(43, 1, 1)
	if findID(s, "listen") != nil {
		t.Fatal("the button shows only when reading aloud is on")
	}
	p := s.Prefs()
	p.Speech = true
	s.ApplyPreferences(p)
	if findID(s, "listen") == nil {
		t.Fatal("reading aloud is on: the button is there")
	}
	tap(t, s, "listen")
	if !strings.Contains(s.notice, "voz alta") || w.speaker.wasAskedToSpeak() {
		t.Errorf("without a speech engine it says so: %q", s.notice)
	}
	w.speaker.available = true
	tap(t, s, "listen")
	if !strings.Contains(w.speaker.spoke, "Versículo 1 do capítulo 1") {
		t.Errorf("it speaks the verses on the page: %q", w.speaker.spoke)
	}
	if b := byID(t, s, "listen").(*widget.Button); b.Text != "Parar" {
		t.Errorf("the button turns into Stop: %q", b.Text)
	}
	tap(t, s, "listen")
	if w.speaker.stopped != 1 {
		t.Error("stopped")
	}
}

func TestThePageKeysTurnThePagesOfTheReaderOrOfTheApp(t *testing.T) {
	w := newWorld(t)
	w.ready()
	s := w.screen("pt-BR")
	if !s.Key(false) || s.Page() != 1 || !s.Key(true) || s.Page() != 0 {
		t.Error("outside the reader the keys move between the pages")
	}
	s.OpenReader(1, 1, 1)
	if !s.Key(false) || s.reader.Page != 1 || !s.Key(true) || s.reader.Page != 0 {
		t.Errorf("in the reader they turn the page: %d", s.reader.Page)
	}
	m := widget.NewModal("x", widget.NewLabel("x"))
	m.Open(s.Doc)
	if s.Key(false) {
		t.Error("with a modal open the keys are left to the platform")
	}
}

func TestAChapterTheVersionLacksSaysSo(t *testing.T) {
	w := newWorld(t)
	w.ready()
	s := w.screen("pt-BR")
	s.OpenReader(43, 9, 1)
	if s.reader != nil {
		t.Error("nothing to read")
	}
	s.OpenChapter(43, 9)
	s.Show(1)
	if findID(s, "read-chapter") != nil {
		t.Error("no button to read a chapter that is not there")
	}
}

func TestTheScreenWithoutTheVersionAsksForTheDownloadInTheBiblePage(t *testing.T) {
	w := newWorld(t)
	if err := w.bible.Refresh(); err != nil {
		t.Fatal(err)
	}
	s := w.screen("pt-BR")
	s.Show(1)
	tap(t, s, "book-19")
	if findID(s, "download-current") == nil {
		t.Fatal("the chapters page offers the download of the version in use")
	}
	tap(t, s, "download-current")
	byID(t, s, "modal-download").(*widget.Modal).Confirm()
	if !s.ready() {
		t.Error("downloaded")
	}
	s.Search("palavras")
	if len(s.hits) == 0 {
		t.Error("now it searches")
	}
}

// checkElement is the touch-accuracy rule of the toolkit: what is touchable is reached by a touch at
// its center, is big enough and stays on the screen.
func checkElement(t *testing.T, s *Screen, where string, w widget.Widget, screenH float64) {
	t.Helper()
	e := w.Elem()
	if !e.Visible() {
		return
	}
	if e.Bounds.Y+e.Bounds.H > screenH+0.5 {
		t.Errorf("%s: %s (%T) overflows the screen: %v of %.0f", where, e.ID, w, e.Bounds, screenH)
	}
	if _, ok := w.(widget.PointerHandler); !ok || e.Disabled || e.Bounds.W <= 0 {
		return
	}
	if got := s.Doc.TargetAt(e.Bounds.Center()); got != w {
		t.Errorf("%s: %s (%T) does not get its own touch", where, e.ID, w)
	}
	if e.Bounds.H < 56 {
		t.Errorf("%s: %s is only %.0f px high", where, e.ID, e.Bounds.H)
	}
}

func TestEveryPageFitsTheScreenAndEveryElementGetsItsTouch(t *testing.T) {
	w := newWorld(t)
	s := w.screen("pt-BR")
	check := func(where string) {
		t.Helper()
		s.Doc.Layout()
		h := s.Doc.Root.Elem().Bounds.H
		s.Doc.Walk(func(x widget.Widget) bool {
			if id := x.Elem().ID; s.Doc.HasModal() && !strings.HasPrefix(id, "settings-") && !strings.HasPrefix(id, "modal-") && !strings.HasPrefix(id, "status-") && !strings.HasPrefix(id, "combo-") || strings.HasPrefix(id, "combo-") && !strings.Contains(id, "-picker") && !strings.Contains(id, "-search") && !strings.Contains(id, "-opt-") {
				return true // under the modal: not touchable now
			}
			checkElement(t, s, where, x, h)
			return true
		})
	}
	check("first run home")
	s.Show(1)
	check("no versions")
	if err := w.bible.Refresh(); err != nil {
		t.Fatal(err)
	}
	_ = s.reload()
	s.rebuildBible()
	check("bar without a version")
	w.ready()
	s = w.screen("pt-BR")
	check("home")
	s.Show(1)
	check("books")
	s.OpenBook(19)
	check("chapters with previews")
	s.OpenChapter(19, 2)
	check("verses")
	for _, size := range []int{domain.MinFontSize, domain.DefaultFontSize, domain.MaxFontSize} {
		p := s.Prefs()
		p.FontSize, p.Speech = size, true
		s.ApplyPreferences(p)
		s.OpenReader(19, 2, 1)
		check(fmt.Sprintf("reader at %d", size))
	}
	s.Search("livro")
	check("search")
	s.OpenStatus()
	check("status")
	byID(t, s, "modal-status").(*widget.Modal).Close()
	s.openBibleTab()
	check("books with the continue button")
	s.Show(0)
	check("home with the last verse")
	s.Show(1)
	tap(t, s, "combo-language") // a popup covers part of the page: its own tests check it
	s.Doc.ClosePopups()
	tap(t, s, "combo-version")
	s.Doc.ClosePopups()
	s.OpenSettings()
	check("settings")
	byID(t, s, "modal-settings").(*widget.Modal).Cancel()
}
