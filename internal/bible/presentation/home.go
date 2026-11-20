package presentation

import (
	"go-pocketbook-base/internal/bible/domain"
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
	"go-pocketbook-base/internal/toolkit/widget"
)

// homePage is the first page: the reading progress, the verse of the day and a way to go on
// reading where the person stopped.
func (s *Screen) homePage() widget.Widget {
	page := widget.NewColumn(24)
	page.ID = "home"

	if !s.ready() {
		page.AppendChild(s.homeEmpty())
		return page
	}

	st, err := s.cfg.Bible.Progress(s.prefs.Version)
	if err != nil {
		s.fail(err)
	}
	bar := widget.NewProgress(st.Percent)
	bar.ID = "progress"
	bar.Text = s.percentText(st.Percent)
	bar.Style.Width = widget.Pct(100)
	bar.Style.MinHeight = 56
	detail := muted(s.t("bible.home.progress.detail", s.number(st.Read), s.number(st.Total)))
	page.AppendChild(widget.NewCard(s.t("bible.home.progress"), "", bar, detail))

	page.AppendChild(s.dailyCard())
	if last := s.lastCard(); last != nil {
		page.AppendChild(last)
	}
	return page
}

// homeEmpty is what the Home page says while no version is on the reader.
func (s *Screen) homeEmpty() widget.Widget {
	get := widget.NewButton(s.t("bible.home.get"))
	get.ID = "get-bible"
	get.SetVariant(widget.VariantPrimary)
	get.OnClick = func(*widget.Event) { s.getFirstVersion() }
	open := widget.NewButton(s.t("bible.home.open"))
	open.ID = "open-bible"
	open.OnClick = func(*widget.Event) { s.Show(1) }
	return widget.NewCard(s.t("bible.menu.bible"), "", widget.NewLabel(s.t("bible.home.empty")), get, open)
}

// dailyCard is the card of the verse of the day, with its "I read it" check box.
func (s *Screen) dailyCard() widget.Widget {
	today := s.now()
	d, ok, err := s.cfg.Bible.VerseOfTheDay(s.prefs.Version, today)
	if err != nil {
		s.fail(err)
	}
	subtitle := ""
	if ok && d.Festive {
		subtitle = s.t("bible.home.votd.festive")
	}
	card := widget.NewCard(s.t("bible.home.votd"), subtitle)
	card.ID = "votd"
	if !ok {
		card.AppendChild(widget.NewLabel(s.t("bible.home.novotd")))
		return card
	}
	ref := widget.NewLabel(s.refText(d.Ref))
	ref.Style.Bold = true
	text := widget.NewLabel("“" + d.Text + "”")
	text.Style.FontSize = float64(s.prefs.FontSize)
	card.AppendChild(ref)
	card.AppendChild(text)

	if d.Read {
		card.AppendChild(doneRow(s.t("bible.home.done")))
		return card
	}
	check := widget.NewCheckbox(s.t("bible.home.read"), false)
	check.ID = "votd-read"
	check.OnChange = func(e *widget.Event) {
		if !check.Checked {
			return
		}
		if err := s.cfg.Bible.DailyRead(today); err != nil {
			s.fail(err)
			return
		}
		s.Show(0) // the check box goes away, the done mark takes its place and the progress grows
		s.Doc.Invalidate()
	}
	card.AppendChild(check)
	return card
}

// doneRow is the mark of something finished: a check in a filled square and its word. The screen
// is drawn in grays, so "green" is the dark fill.
func doneRow(label string) widget.Widget {
	mark := widget.NewIconButton(gfx.IconCheck)
	mark.ID = "votd-done"
	mark.SetVariant(widget.VariantPrimary)
	mark.Style.Color = widget.Paper
	mark.Style.MinHeight, mark.Style.MinWidth = 64, 64
	text := widget.NewLabel(label)
	text.Style.Bold = true
	row := widget.NewRow(16, mark, text)
	row.Style.AlignItems = widget.AlignCenter
	return row
}

// resumePoint is the place the reading stopped at (false when there is none, or the book is not known).
func (s *Screen) resumePoint() (domain.Ref, bool) {
	pos, err := s.cfg.Bible.Resume()
	if err != nil || !pos.Valid() {
		return domain.Ref{}, false
	}
	if _, ok := s.bookByID(pos.Book); !ok {
		return domain.Ref{}, false
	}
	verse := pos.Verse
	if verse < 1 {
		verse = 1
	}
	return domain.At(pos.Book, pos.Chapter, verse), true
}

// continueButton resumes the reading at the place saved, nil when there is none.
func (s *Screen) continueButton() widget.Widget {
	at, ok := s.resumePoint()
	if !ok {
		return nil
	}
	b := widget.NewButton(s.t("bible.home.continue.at", s.refText(at)))
	b.ID = "continue"
	b.SetVariant(widget.VariantPrimary)
	b.Style.Padding = geom.XY(0, 24)
	b.OnClick = func(*widget.Event) { s.OpenReader(at.Book, at.Chapter, at.Verse) }
	return b
}

// lastCard is the card of the last verse read: where it is, its text and the button to go on
// from there. Nil when nothing was read yet.
func (s *Screen) lastCard() widget.Widget {
	at, ok := s.resumePoint()
	if !ok {
		return nil
	}
	card := widget.NewCard(s.t("bible.home.last"), "")
	card.ID = "last"
	ref := widget.NewLabel(s.refText(at))
	ref.Style.Bold = true
	card.AppendChild(ref)
	if text := s.verseText(at); text != "" {
		quote := widget.NewLabel("“" + oneLine(text, lastVerseLetters) + "”")
		quote.ID = "last-text"
		quote.Style.FontSize = float64(s.prefs.FontSize)
		card.AppendChild(quote)
	}
	card.AppendChild(s.continueButton())
	return card
}

// lastVerseLetters is the longest text of a verse the Home card shows (the reader has all of it).
const lastVerseLetters = 280

// verseText is the text of a verse in the version in use ("" when it is not on the reader).
func (s *Screen) verseText(r domain.Ref) string {
	if !s.ready() {
		return ""
	}
	ch, err := s.cfg.Bible.Chapter(s.prefs.Version, r.Book, r.Chapter)
	if err != nil {
		return ""
	}
	for _, v := range ch.Verses {
		if v.Number == r.Verse {
			return v.Text
		}
	}
	return ""
}
