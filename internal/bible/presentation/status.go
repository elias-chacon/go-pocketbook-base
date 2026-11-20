package presentation

import (
	"time"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/widget"
)

// OpenStatus opens the Status window: the language the reader reported and where it came from, the
// language of the app, the time, the reading progress and the last verse read, with a button to go
// there. It is also what to look at when the reader's language or clock do not look right.
func (s *Screen) OpenStatus() {
	lines := widget.NewColumn(10)
	add := func(id, text string) {
		l := widget.NewLabel(text)
		l.ID = id
		lines.AppendChild(l)
	}

	source := s.cfg.DeviceLangSource
	if source == "" {
		source = s.t("bible.status.unknown")
	}
	add("status-locale", s.t("bible.status.locale", s.cfg.DeviceLang, source))
	add("status-lang", s.t("bible.status.lang", s.tr.Lang(), s.uiLangSource()))

	now := s.now()
	clock := s.t("bible.status.time", s.timeText(now))
	if s.cfg.ClockSource != "" {
		clock += " · " + s.cfg.ClockSource
	}
	add("status-time", clock)

	if s.ready() {
		if st, err := s.cfg.Bible.Progress(s.prefs.Version); err == nil {
			add("status-progress", s.t("bible.status.progress", s.percentText(st.Percent), s.number(st.Read), s.number(st.Total)))
		}
	} else {
		add("status-progress", s.t("bible.status.noversion"))
	}

	body := []widget.Widget{lines}
	if at, ok := s.resumePoint(); ok {
		add("status-last", s.t("bible.status.last", s.refText(at)))
		if text := s.verseText(at); text != "" {
			quote := muted("“" + oneLine(text, 160) + "”")
			quote.ID = "status-last-text"
			lines.AppendChild(quote)
		}
		goTo := widget.NewButton(s.t("bible.status.goto", s.refText(at)))
		goTo.ID = "status-goto"
		goTo.SetVariant(widget.VariantPrimary)
		goTo.Style.Padding = geom.XY(0, 20)
		body = append(body, goTo)
		m := s.statusModal(body)
		goTo.OnClick = func(*widget.Event) {
			m.Close()
			s.Show(1)
			s.OpenReader(at.Book, at.Chapter, at.Verse)
		}
		m.Open(s.Doc)
		return
	}
	add("status-last", s.t("bible.status.nolast"))
	s.statusModal(body).Open(s.Doc)
}

func (s *Screen) statusModal(body []widget.Widget) *widget.Modal {
	m := widget.NewModal(s.t("bible.status.title"), body...)
	m.ID = "modal-status"
	m.OKText = s.t("bible.combo.close")
	m.SetButtons(widget.ModalOK)
	m.SetControls(widget.ModalClose)
	return m
}

// uiLangSource says why the interface is in the language it is.
func (s *Screen) uiLangSource() string {
	if s.prefs.Lang != "" && s.prefs.Lang != s.cfg.DeviceLang {
		return s.t("bible.status.chosen")
	}
	return s.t("bible.status.fromreader")
}

// timeText writes a date and time the way the language does, with the offset from UTC.
func (s *Screen) timeText(t time.Time) string {
	layout := "Jan 2, 2006 3:04 PM"
	if s.tr.Lang() == "pt-BR" {
		layout = "02/01/2006 15:04"
	}
	return t.Format(layout) + " (UTC" + t.Format("-07:00") + ")"
}
