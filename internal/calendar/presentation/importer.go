package presentation

import (
	"strconv"
	"strings"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/widget"
)

// maxSources is how many files the import screen lists.
const maxSources = 8

// Importer brings events in from a file of the reader or from an address (an iCalendar
// link: the secret address of a Google calendar, the published link of an Outlook one,
// any webcal). It returns how many events were kept and what it could not understand.
type Importer interface {
	// Files lists the .ics files found on the reader.
	Files() []string
	// Import reads a file path or an address.
	Import(source string) (count int, warnings []string, err error)
}

// ShowImport shows the import screen: an address field and the files found.
func (s *Screen) ShowImport() {
	s.editor = nil
	title := widget.NewLabel(s.t("cal.import.title"))
	title.Style.FontSize, title.Style.Bold = widget.FontTitle, true
	hint := widget.NewLabel(s.t("cal.import.hint"))
	hint.Style.FontSize, hint.Style.Color = widget.FontSmall, widget.Muted

	url := widget.NewInput(s.t("cal.import.url"))
	url.ID = "import-url"
	fetch := widget.NewButton(s.t("cal.import.fetch"))
	fetch.ID = "import-fetch"
	status := widget.NewLabel("")
	status.ID = "import-status"
	status.Style.Bold = true
	fetch.OnClick = func(*widget.Event) { s.runImport(strings.TrimSpace(url.Value), status) }

	children := []widget.Widget{title, hint, url, fetch, s.filesHeading()}
	files := s.cfg.Importer.Files()
	if len(files) == 0 {
		none := widget.NewLabel(s.t("cal.import.nofiles"))
		none.Style.FontSize, none.Style.Color = widget.FontSmall, widget.Muted
		children = append(children, none)
	}
	for i, f := range files {
		if i == maxSources {
			break
		}
		f := f
		b := widget.NewButton(f)
		b.ID = "import-file-" + strconv.Itoa(i)
		b.OnClick = func(*widget.Event) { s.runImport(f, status) }
		children = append(children, b)
	}
	back := widget.NewButton(s.t("cal.import.back"))
	back.ID = "import-back"
	back.OnClick = func(*widget.Event) { s.show() }
	children = append(children, status, back)

	form := widget.NewColumn(14, children...)
	form.Style.Padding = geom.XY(28, 28)
	form.Style.FlexGrow = 1
	root := widget.NewColumn(0, form)
	root.Style.Background = widget.Paper
	s.Doc.SetRoot(s.cfg.Chrome.Wrap(s.Doc, root))
	s.Doc.ResetFrame()
	s.Doc.Invalidate()
}

func (s *Screen) filesHeading() widget.Widget {
	l := widget.NewLabel(s.t("cal.import.files"))
	l.Style.Bold = true
	return l
}

// runImport imports one source and says what happened on the status line.
func (s *Screen) runImport(source string, status *widget.Label) {
	if source == "" {
		status.Text = s.t("cal.import.empty")
		s.Doc.Invalidate()
		return
	}
	status.Text = s.t("cal.import.working")
	if s.cfg.Flush != nil {
		s.cfg.Flush()
	}
	n, warnings, err := s.cfg.Importer.Import(source)
	switch {
	case err != nil:
		status.Text = s.t("cal.import.failed", err.Error())
	case len(warnings) > 0:
		status.Text = s.t("cal.import.partial", n, len(warnings), warnings[0])
	default:
		status.Text = s.t("cal.import.done", n)
	}
	s.Doc.Invalidate()
}
