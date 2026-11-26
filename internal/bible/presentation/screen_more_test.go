package presentation

import (
	"strings"
	"testing"
	"time"

	"go-pocketbook-base/internal/toolkit/widget"
)

func labelText(t *testing.T, s *Screen, id string) string {
	t.Helper()
	return byID(t, s, id).(*widget.Label).Text
}

func TestTheStatusWindowShowsLocaleTimeProgressAndTheLastVerseWithAWayToGoThere(t *testing.T) {
	w := newWorld(t)
	w.ready()
	s := w.screen("pt-BR")
	s.OpenStatus()
	if got := labelText(t, s, "status-locale"); !strings.Contains(got, "pt-BR") || !strings.Contains(got, "language=pt") {
		t.Errorf("locale and where it came from: %q", got)
	}
	if got := labelText(t, s, "status-lang"); !strings.Contains(got, "segue o aparelho") {
		t.Errorf("app language: %q", got)
	}
	if got := labelText(t, s, "status-time"); !strings.Contains(got, "19/11/2026 08:00") || !strings.Contains(got, "UTC+00:00") || !strings.Contains(got, "Buenos Aires") {
		t.Errorf("time: %q", got)
	}
	if got := labelText(t, s, "status-progress"); !strings.Contains(got, "0,0%") {
		t.Errorf("progress: %q", got)
	}
	if findID(s, "status-goto") != nil || !strings.Contains(labelText(t, s, "status-last"), "Nada lido") {
		t.Error("nothing read yet: no button to go to")
	}
	byID(t, s, "modal-status").(*widget.Modal).Close()

	s.OpenReader(43, 2, 5)
	s.Show(0)
	s.OpenStatus()
	if got := labelText(t, s, "status-last"); !strings.Contains(got, "João 2:") {
		t.Errorf("the last verse: %q", got)
	}
	if got := labelText(t, s, "status-last-text"); !strings.Contains(got, "Versículo") {
		t.Errorf("the verse's text: %q", got)
	}
	tap(t, s, "status-goto")
	if s.Doc.HasModal() || s.Page() != 1 || findID(s, "reader") == nil {
		t.Errorf("the button goes straight to the reader (modal=%v page=%d)", s.Doc.HasModal(), s.Page())
	}
	s.Show(0)
	p := s.Prefs()
	p.Lang = "en"
	s.ApplyPreferences(p)
	s.OpenStatus()
	if got := labelText(t, s, "status-time"); !strings.Contains(got, "Nov 19, 2026 8:00 AM") {
		t.Errorf("English time: %q", got)
	}
	if got := labelText(t, s, "status-lang"); !strings.Contains(got, "chosen in Settings") {
		t.Errorf("the language was chosen: %q", got)
	}
}

func TestTheStatusWindowSaysWhenTheLanguageWasNotDetected(t *testing.T) {
	w := newWorld(t)
	w.ready()
	s, err := NewScreen(Config{Width: 1404, Height: 1872, Bible: w.bible, DeviceLang: "pt-BR", Now: func() time.Time { return today }})
	if err != nil {
		t.Fatal(err)
	}
	s.OpenStatus()
	if got := labelText(t, s, "status-locale"); !strings.Contains(got, "não detectado") {
		t.Errorf("no source: %q", got)
	}
}

func TestTheHomeCardOfTheLastVerseHasItsTextAndTheContinueButtonInside(t *testing.T) {
	w := newWorld(t)
	w.ready()
	s := w.screen("pt-BR")
	if findID(s, "last") != nil || findID(s, "continue") != nil {
		t.Fatal("nothing read: no card and no button")
	}
	s.OpenReader(43, 2, 7)
	s.Show(0)
	card := byID(t, s, "last").(*widget.Panel)
	if got := labelText(t, s, "last-text"); !strings.Contains(got, "capítulo 2 do livro 43") {
		t.Errorf("the verse as content: %q", got)
	}
	cont := byID(t, s, "continue")
	if cont.Elem().Parent() != card {
		t.Error("the button is inside the card")
	}
	if !strings.Contains(cont.(*widget.Button).Text, "João 2:") {
		t.Errorf("Continue at: %q", cont.(*widget.Button).Text)
	}
	tap(t, s, "continue")
	if findID(s, "reader") == nil {
		t.Error("it opens the reader")
	}
}

func TestTheBiblePageHasAContinueButtonAndTappingItsTabGoesBackToTheBooks(t *testing.T) {
	w := newWorld(t)
	w.ready()
	s := w.screen("pt-BR")
	s.Show(1)
	if findID(s, "continue") != nil {
		t.Error("nothing read yet")
	}
	s.OpenReader(19, 1, 1)
	if findID(s, "reader") == nil {
		t.Fatal("reading")
	}
	s.tabs.Bar.Menus[1].OnSelect() // a tap on the Bible tab
	if findID(s, "reader") != nil || findID(s, "book-19") == nil {
		t.Error("the tab goes back to the list of books")
	}
	if b := findID(s, "continue"); b == nil || !strings.Contains(b.(*widget.Button).Text, "Salmos 1:") {
		t.Error("the Bible page offers to continue where it stopped")
	}
	tap(t, s, "continue")
	if findID(s, "reader") == nil {
		t.Error("the button opens the reader")
	}
	s.OpenBook(43)
	s.tabs.Bar.Menus[1].OnSelect()
	if findID(s, "book-43") == nil || findID(s, "chapter-1") != nil {
		t.Error("from the chapters too")
	}
	s.Search("livro")
	s.tabs.Bar.Menus[1].OnSelect()
	if s.view != viewBooks || s.query != "" {
		t.Errorf("and from a search: view=%d query=%q", s.view, s.query)
	}
}

func TestADownloadRunsInStepsWithABarAndTheScreenStaysUsable(t *testing.T) {
	w := newWorld(t)
	w.async = true
	w.catalog.chapters = 15 // 45 chapters: three requests
	s := w.screen("pt-BR")
	tap(t, s, "get-bible")
	if !s.Busy() || s.strip.Hidden || s.Doc.HasModal() {
		t.Fatalf("the list is fetched in the strip, no modal: busy=%v modal=%v", s.Busy(), s.Doc.HasModal())
	}
	if !strings.Contains(s.stripLabel.Text, "lista de versões") {
		t.Errorf("strip: %q", s.stripLabel.Text)
	}
	s.Show(1) // the person browses meanwhile
	if findID(s, "bible") == nil {
		t.Fatal("the Bible page opens while the task runs")
	}
	w.drain(t, func(step int) {
		if step == 3 {
			s.Show(0) // and goes Home in the middle of the download
			checkTheBarFromHome(t, s)
		}
	})
	if s.Busy() || !s.strip.Hidden {
		t.Error("the strip goes away at the end")
	}
	if !s.ready() || s.Prefs().Version != "almeida-livre" || findID(s, "progress") == nil {
		t.Errorf("downloaded, in use, and the Home that was open shows the progress: ready=%v %+v", s.ready(), s.Prefs())
	}
}

func TestADownloadCanBeCancelledAndGoesOnLater(t *testing.T) {
	w := newWorld(t)
	w.async = true
	w.catalog.chapters = 15
	w.ready()
	s := w.screen("pt-BR")
	s.Show(1)
	s.pickLanguage("en")
	s.pickVersion("kjv")
	byID(t, s, "modal-download").(*widget.Modal).Confirm()
	if !s.Busy() {
		t.Fatal("the download started")
	}
	w.runNext()
	s.pickVersion("web") // a second one while the first runs
	byID(t, s, "modal-download").(*widget.Modal).Confirm()
	if !strings.Contains(s.notice, "Outro download") {
		t.Errorf("only one at a time: %q", s.notice)
	}
	tap(t, s, "job-cancel")
	if s.Busy() || !s.strip.Hidden || !strings.Contains(s.notice, "cancelado") {
		t.Errorf("cancelled: busy=%v %q", s.Busy(), s.notice)
	}
	for w.runNext() { // the steps still queued do nothing
	}
	if v, _ := s.currentVersion(); v.Slug == "kjv" && v.Downloaded {
		t.Error("it did not finish")
	}
	s.pickVersion("kjv")
	byID(t, s, "modal-download").(*widget.Modal).Confirm()
	for w.runNext() {
	}
	if s.prefs.Version != "kjv" || !s.ready() {
		t.Errorf("started again, it finishes: %+v", s.prefs)
	}
}

func TestADownloadThatEndsWhileReadingDoesNotPullTheChapterAway(t *testing.T) {
	w := newWorld(t)
	w.async = true
	w.catalog.chapters = 15
	w.ready()
	s := w.screen("pt-BR")
	s.Show(1)
	s.pickLanguage("en")
	s.pickVersion("kjv")
	byID(t, s, "modal-download").(*widget.Modal).Confirm()
	s.OpenReader(43, 1, 1)
	for w.runNext() {
	}
	if findID(s, "reader") == nil || s.prefs.Version != "almeida-livre" {
		t.Errorf("still reading the same chapter in the same version: %s", s.prefs.Version)
	}
	if !strings.Contains(s.notice, "KJV") {
		t.Errorf("but it says the version is ready: %q", s.notice)
	}
}

func TestThePickersOfLanguageAndVersionAreComboBoxesThatOpenAPopupNotAWindow(t *testing.T) {
	w := newWorld(t)
	w.ready()
	s := w.screen("pt-BR")
	s.Show(1)
	tap(t, s, "combo-language")
	if s.Doc.HasModal() || !s.Doc.HasPopup() {
		t.Error("the language list is the combo box's popup under the field, not a window")
	}
	s.Doc.ClosePopups()
	s.pickLanguage("en")
	tap(t, s, "combo-version")
	if s.Doc.HasModal() || !s.Doc.HasPopup() {
		t.Error("the version list is a popup too")
	}
	s.Doc.ClosePopups()
	box := byID(t, s, "combo-version").(*widget.ComboBox)
	box.Index = 1 // what a tap on the second row does
	box.OnChange(&widget.Event{})
	if findID(s, "modal-download") == nil {
		t.Error("the version is not on the reader: it asks first")
	}
	if box.SearchPlaceholder != "Buscar" || box.NoMatchText == "" {
		t.Errorf("the search texts of a long list are the app's: %q %q", box.SearchPlaceholder, box.NoMatchText)
	}
}

// drain runs what the screen queued until nothing is left, calling each with the step number.
func (w *world) drain(t *testing.T, each func(step int)) {
	t.Helper()
	for steps := 1; w.runNext(); steps++ {
		if each != nil {
			each(steps)
		}
		if steps > 50 {
			t.Fatal("never ends")
		}
	}
}

// checkTheBarFromHome is what the strip shows while a download runs and the Home is open.
func checkTheBarFromHome(t *testing.T, s *Screen) {
	t.Helper()
	if !s.Busy() || s.strip.Hidden {
		t.Fatal("the strip is on every page")
	}
	if s.stripBar.Value <= 0 || !strings.Contains(s.stripLabel.Text, "Baixando") {
		t.Errorf("the bar moves: %v %q", s.stripBar.Value, s.stripLabel.Text)
	}
}

// lastPage turns the pages of the reader until the last one.
func lastPage(t *testing.T, s *Screen) {
	t.Helper()
	s.Doc.Layout()
	for i := 0; s.reader.Page < s.reader.Pages()-1; i++ {
		if i > 100 {
			t.Fatal("the chapter never ends")
		}
		if !s.Key(false) {
			t.Fatal("the key was not taken")
		}
	}
}

func TestTheNextKeyOnTheLastPageGoesToTheNextChapterAndThePreviousOnTheFirstToTheOneBefore(t *testing.T) {
	w := newWorld(t)
	w.ready()
	s := w.screen("pt-BR")
	s.OpenReader(43, 1, 1)
	lastPage(t, s)
	if s.chapter != 1 {
		t.Fatalf("still in the chapter: %d", s.chapter)
	}
	s.Key(false)
	if s.book != 43 || s.chapter != 2 || s.reader == nil || s.reader.Page != 0 {
		t.Fatalf("the next key on the last page opens the next chapter: %d:%d page %d", s.book, s.chapter, s.reader.Page)
	}
	pos, _ := w.bible.Resume()
	if pos.Book != 43 || pos.Chapter != 2 || pos.Verse != 1 {
		t.Errorf("the place follows: %+v", pos)
	}
	s.Key(true)
	if s.chapter != 1 || s.reader.Page != s.reader.Pages()-1 {
		t.Errorf("the previous key on the first page opens the end of the chapter before: %d page %d of %d", s.chapter, s.reader.Page, s.reader.Pages())
	}
	if pos, _ = w.bible.Resume(); pos.Verse >= endOfChapter || pos.Verse < 1 {
		t.Errorf("the place is a real verse: %+v", pos)
	}
}

func TestTheKeysCrossBooksAndStopAtTheEndsOfTheBible(t *testing.T) {
	w := newWorld(t)
	w.ready()
	s := w.screen("pt-BR")
	s.OpenReader(1, 3, 1) // the last chapter of Genesis
	lastPage(t, s)
	s.Key(false)
	if s.book != 19 || s.chapter != 1 {
		t.Errorf("after Genesis comes Psalms: %d:%d", s.book, s.chapter)
	}
	s.Key(true)
	if s.book != 1 || s.chapter != 3 {
		t.Errorf("back to the end of Genesis: %d:%d", s.book, s.chapter)
	}
	s.OpenReader(1, 1, 1)
	if !s.Key(true) || s.book != 1 || s.chapter != 1 {
		t.Errorf("nothing before Genesis 1: %d:%d", s.book, s.chapter)
	}
	s.OpenReader(43, 3, 1)
	lastPage(t, s)
	if !s.Key(false) || s.book != 43 || s.chapter != 3 {
		t.Errorf("nothing after the last chapter: %d:%d", s.book, s.chapter)
	}
}
