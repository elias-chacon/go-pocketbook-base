package presentation

import (
	"fmt"
	"strings"

	"go-pocketbook-base/internal/bible/domain"
	"go-pocketbook-base/internal/toolkit/gfx"
	"go-pocketbook-base/internal/toolkit/widget"
)

// readerChrome is the height the reader leaves to everything but the text: the menu bar, the
// header of the chapter, the footer and the paddings (the pager's own footer is counted apart).
const readerChrome = 370.0

// pagerFooter is the height of the footer of a ParagraphPager (page number and buttons).
const pagerFooter = 72.0

// OpenReader opens the text of a chapter at a verse. The verses on the screen count as read.
func (s *Screen) OpenReader(book, chapter, verse int) {
	s.view, s.book, s.chapter, s.verse = viewReader, book, chapter, verse
	if verse != endOfChapter { // that one is no place: the verses on the screen set it
		if err := s.cfg.Bible.Remember(domain.Position{Book: book, Chapter: chapter, Verse: verse}); err != nil {
			s.fail(err)
		}
	}
	s.rebuildBible()
	s.startReading()
}

// startReading turns to the verse asked for and counts what is on the screen as read. The page
// must be laid out first: where a verse falls depends on the width of the text.
func (s *Screen) startReading() {
	if s.reader == nil {
		return
	}
	s.Doc.Layout()
	last := len(s.text.Verses) - 1
	for i, v := range s.text.Verses {
		if v.Number >= s.verse {
			s.reader.GoToParagraph(i)
			s.markVisible()
			return
		}
	}
	s.reader.GoToParagraph(last) // past the last verse: the last page
	s.markVisible()
}

// markVisible records the verses that are on the screen now as read, and the place as the one
// to resume from.
func (s *Screen) markVisible() {
	if s.reader == nil || len(s.text.Verses) == 0 {
		return
	}
	first, last, ok := s.reader.Visible()
	if !ok {
		return
	}
	var numbers []int
	for i := first; i <= last && i < len(s.text.Verses); i++ {
		numbers = append(numbers, s.text.Verses[i].Number)
	}
	if len(numbers) > 0 {
		s.verse = numbers[0] // the place a new layout (a font change) goes back to
	}
	if err := s.cfg.Bible.Seen(s.book, s.chapter, numbers); err != nil {
		s.fail(err)
	}
}

// readerNext turns the page of the chapter; on its last page it goes on to the first page of the
// next chapter (crossing into the next book), so reading never needs the list again.
func (s *Screen) readerNext() {
	s.Doc.Layout()
	if s.reader.Page < s.reader.Pages()-1 {
		s.reader.Next()
		return
	}
	s.openNeighbor(1)
}

// readerPrevious turns the page back; on the first page it goes to the last page of the previous
// chapter.
func (s *Screen) readerPrevious() {
	if s.reader.Page > 0 {
		s.reader.Prev()
		return
	}
	if b, c, ok := s.neighbor(-1); ok {
		s.OpenReader(b, c, endOfChapter)
	}
}

// endOfChapter is a verse number past any chapter: the reader opens on the last page.
const endOfChapter = 1 << 20

// readerLines is how many lines of text the reader's page holds at the size chosen.
func (s *Screen) readerLines() int {
	lh := float64(s.prefs.FontSize) * 1.35
	room := s.Doc.H - readerChrome - pagerFooter
	if s.prefs.Speech {
		room -= widget.ControlHeight + 16
	}
	n := int(room / lh)
	if n < 6 {
		n = 6
	}
	return n
}

// buildReader fills the page with the chapter: a header to go back and to the neighbour
// chapters, and the verses a page at a time.
func (s *Screen) buildReader(page *widget.Panel) {
	s.reader, s.text, s.listening = nil, domain.ChapterText{}, false
	b, ok := s.bookByID(s.book)
	if !ok {
		s.view = viewBooks
		page.AppendChild(s.booksList())
		return
	}
	title := s.t("bible.chapter.ref", s.bookName(b), s.chapter)

	back := widget.NewButton("")
	back.ID = "back"
	back.Icon = gfx.IconChevronLeft
	back.Style.MinWidth = widget.ControlHeight
	back.OnClick = func(*widget.Event) { s.view = viewVerses; s.listPage = 0; s.rebuildBible() }
	label := heading(title)
	label.ID = "reader-title"
	label.Style.FlexGrow = 1
	prev := widget.NewIconButton(gfx.IconChevronLeft)
	prev.ID, prev.Disabled = "chapter-prev", !s.hasNeighbor(-1)
	prev.OnClick = func(*widget.Event) { s.openNeighbor(-1) }
	next := widget.NewIconButton(gfx.IconChevronRight)
	next.ID, next.Disabled = "chapter-next", !s.hasNeighbor(1)
	next.OnClick = func(*widget.Event) { s.openNeighbor(1) }
	header := widget.NewRow(12, back, label, prev, next)
	header.Style.AlignItems = widget.AlignCenter
	page.AppendChild(header)

	if !s.ready() {
		page.AppendChild(s.needsDownload())
		return
	}
	text, err := s.cfg.Bible.Chapter(s.prefs.Version, s.book, s.chapter)
	if err != nil {
		s.fail(err)
	}
	if len(text.Verses) == 0 {
		page.AppendChild(widget.NewLabel(s.t("bible.nochapter")))
		return
	}
	s.text = text
	texts, numbers := make([]string, len(text.Verses)), make([]string, len(text.Verses))
	for i, v := range text.Verses {
		texts[i], numbers[i] = v.Text, fmt.Sprint(v.Number)
	}
	pager := widget.NewParagraphPager(texts, numbers)
	pager.ID = "reader"
	pager.Style.FontSize = float64(s.prefs.FontSize)
	pager.Lines = s.readerLines()
	pager.OnPage = func(*widget.Event) { s.markVisible() }
	s.reader = pager
	page.AppendChild(pager)
	if s.prefs.Speech {
		page.AppendChild(s.listenRow())
	}
}

// listenRow is the button that reads the page aloud.
func (s *Screen) listenRow() widget.Widget {
	b := widget.NewButton(s.t("bible.reader.listen"))
	b.ID = "listen"
	b.OnClick = func(*widget.Event) { s.toggleListening(b) }
	return b
}

// toggleListening starts or stops reading the verses on the page aloud.
func (s *Screen) toggleListening(b *widget.Button) {
	sp := s.cfg.Speaker
	if sp == nil || !sp.Available() {
		s.say(s.t("bible.tts.unavailable"))
		return
	}
	if s.listening {
		sp.Stop()
		s.listening, b.Text = false, s.t("bible.reader.listen")
		s.Doc.Invalidate()
		return
	}
	first, last, ok := s.reader.Visible()
	if !ok {
		return
	}
	var parts []string
	for i := first; i <= last && i < len(s.text.Verses); i++ {
		parts = append(parts, s.text.Verses[i].Text)
	}
	if err := sp.Speak(strings.Join(parts, " ")); err != nil {
		s.fail(err)
		return
	}
	s.listening, b.Text = true, s.t("bible.reader.stop")
	s.Doc.Invalidate()
}

// neighbor is the chapter delta places away, crossing from one book to the next.
func (s *Screen) neighbor(delta int) (book, chapter int, ok bool) {
	b, found := s.bookByID(s.book)
	if !found {
		return 0, 0, false
	}
	chapter = s.chapter + delta
	switch {
	case chapter >= 1 && chapter <= b.Chapters:
		return b.ID, chapter, true
	case chapter < 1 && delta < 0:
		if p, ok := s.adjacentBook(b.ID, -1); ok {
			return p.ID, p.Chapters, true
		}
	case chapter > b.Chapters && delta > 0:
		if n, ok := s.adjacentBook(b.ID, 1); ok {
			return n.ID, 1, true
		}
	}
	return 0, 0, false
}

// adjacentBook is the book before or after one in the canonical order.
func (s *Screen) adjacentBook(id, delta int) (domain.Book, bool) {
	ordered := domain.InCanonicalOrder(s.books)
	for i, b := range ordered {
		if b.ID == id {
			if j := i + delta; j >= 0 && j < len(ordered) {
				return ordered[j], true
			}
		}
	}
	return domain.Book{}, false
}

func (s *Screen) hasNeighbor(delta int) bool { _, _, ok := s.neighbor(delta); return ok }

// openNeighbor reads the chapter before or after the one open, from its first verse.
func (s *Screen) openNeighbor(delta int) {
	if b, c, ok := s.neighbor(delta); ok {
		s.OpenReader(b, c, 1)
	}
}
