// Package presentation draws the Calendar app: a month with dots on the days that
// have events, the events of the chosen day a page at a time, and one event to edit.
// It is built only from the toolkit's widgets and holds no rules: the Agenda use case does.
package presentation

import (
	"embed"
	"fmt"
	"time"

	"go-pocketbook-base/internal/calendar/application"
	"go-pocketbook-base/internal/calendar/domain"
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/localization/i18n"
	"go-pocketbook-base/internal/toolkit/gfx"
	"go-pocketbook-base/internal/toolkit/widget"
)

//go:embed locales/*.json
var locales embed.FS

// NewTranslator loads the Calendar app's texts (pt-BR first, English fallback) in lang.
func NewTranslator(lang string) *i18n.Bundle {
	tr := i18n.New()
	if err := tr.AddFS(locales, "locales"); err != nil {
		panic(err) // embedded files: an error is a programming mistake
	}
	tr.SetLang(lang)
	return tr
}

// Agenda is what the screen needs of the Calendar context (application.Agenda).
type Agenda interface {
	Day(day time.Time) ([]domain.Occurrence, error)
	Marks(year int, month time.Month) (map[int]bool, error)
	Add(in application.Entry) (domain.Event, error)
	Change(id string, in application.Entry) error
	Get(id string) (domain.Event, error)
	Delete(id string) error
	Today() time.Time
	Loc() *time.Location
}

var _ Agenda = application.Agenda{}

// Config is everything the screen needs.
type Config struct {
	Width, Height int
	// Doc is the document to draw on (a new one when nil).
	Doc *widget.Document
	// Chrome is the standard top bar (the app name with About and Exit), above every screen.
	Chrome   widget.AppChrome
	Keyboard widget.Keyboard
	Tr       *i18n.Bundle
	Agenda   Agenda
	Notice   string
}

const rowsPerPage = 5

// Screen is the Calendar app's screen.
type Screen struct {
	Doc *widget.Document

	cfg      Config
	selected time.Time // the day shown, a date at midnight UTC
	items    []domain.Occurrence
	marks    map[int]bool
	page     int
	notice   string
	editor   *editor
	locale   widget.Locale

	cal             *widget.Calendar
	body            *widget.Panel
	heading         *widget.Label
	note, indicator *widget.Label
	prev, next      *widget.Button
}

// NewScreen builds the screen on today.
func NewScreen(cfg Config) *Screen {
	s := &Screen{cfg: cfg, notice: cfg.Notice}
	s.Doc = cfg.Doc
	if s.Doc == nil {
		s.Doc = widget.NewDocument(cfg.Width, cfg.Height)
		s.Doc.Keyboard = cfg.Keyboard
	}
	s.locale = widget.LocaleEN
	if cfg.Tr.Lang() == "pt-BR" {
		s.locale = widget.LocalePTBR
	}
	t := cfg.Agenda.Today()
	s.selected = dateOf(t)
	s.showMonth()
	return s
}

// dateOf is the calendar date of t as a midnight UTC (the widgets work with dates).
func dateOf(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func (s *Screen) t(key string, args ...interface{}) string { return s.cfg.Tr.T(key, args...) }

// Selected is the day shown.
func (s *Screen) Selected() time.Time { return s.selected }

// Page is the page of the day's events, counted from 0.
func (s *Screen) Page() int { return s.page }

// Editing is the id of the event being edited ("" while the month shows).
func (s *Screen) Editing() string {
	if s.editor == nil {
		return ""
	}
	return s.editor.id
}

func paper(w widget.Widget) widget.Widget {
	w.Elem().Style.Background = widget.Paper
	return w
}

func (s *Screen) fail(err error) {
	s.notice = s.t("cal.failed", err.Error())
	if s.note != nil {
		s.note.Text = s.notice
	}
	s.Doc.Invalidate()
}

func (s *Screen) heading1() string {
	d := s.selected
	return fmt.Sprintf("%s, %02d/%02d/%d", s.locale.Weekdays[d.Weekday()], d.Day(), int(d.Month()), d.Year())
}

func (s *Screen) showMonth() {
	s.editor = nil
	title := widget.NewLabel(s.t("cal.title"))
	title.Style.FontSize, title.Style.Bold, title.Style.FlexGrow = widget.FontTitle, true, 1
	today := widget.NewButton(s.t("cal.today"))
	today.ID = "cal-today"
	today.OnClick = func(*widget.Event) { s.Select(dateOf(s.cfg.Agenda.Today())) }
	add := widget.NewButton(s.t("cal.new"))
	add.ID = "cal-new"
	add.OnClick = func(*widget.Event) { s.NewEvent() }
	head := widget.NewRow(12, title, today, add)
	head.Style.Padding = geom.XY(28, 28)
	head.Style.AlignItems = widget.AlignCenter
	paper(head)

	s.cal = widget.NewCalendar(s.selected)
	s.cal.ID = "cal-month"
	s.cal.Locale = s.locale
	s.cal.Today = dateOf(s.cfg.Agenda.Today())
	s.cal.Marked = func(day time.Time) bool {
		return day.Month() == s.cal.Month && day.Year() == s.cal.Year && s.marks[day.Day()]
	}
	s.cal.OnChange = func(*widget.Event) { s.Select(dateOf(s.cal.Selected)) }
	s.cal.OnMonth = func(*widget.Event) { s.loadMarks() }
	calRow := widget.NewRow(0, s.cal)
	calRow.Style.Padding = geom.XY(14, 28)
	paper(calRow)

	s.heading = widget.NewLabel("")
	s.heading.ID = "cal-day"
	s.heading.Style.Bold = true
	s.heading.Style.Padding = geom.XY(8, 28)
	paper(s.heading)

	s.body = widget.NewColumn(8)
	s.body.ID = "cal-body"
	s.body.Style.FlexGrow, s.body.Style.Padding = 1, geom.XY(8, 28)

	s.note = widget.NewLabel(s.notice)
	s.note.Style.FontSize, s.note.Style.Color, s.note.Style.FlexGrow = widget.FontSmall, widget.Muted, 1
	s.indicator = widget.NewLabel("")
	s.indicator.Style.FontSize, s.indicator.Style.Bold = widget.FontSmall, true
	s.indicator.Style.TextAlign, s.indicator.Style.Width = gfx.AlignCenter, widget.Px(110)
	s.prev, s.next = widget.NewIconButton(gfx.IconChevronLeft), widget.NewIconButton(gfx.IconChevronRight)
	s.prev.ID, s.next.ID = "cal-prev", "cal-next"
	s.prev.OnClick = func(*widget.Event) { s.GoTo(s.page - 1) }
	s.next.OnClick = func(*widget.Event) { s.GoTo(s.page + 1) }
	foot := widget.NewRow(12, s.note, s.prev, s.indicator, s.next)
	foot.Style.Padding = geom.XY(14, 28)
	paper(foot)

	root := widget.NewColumn(0, head, calRow, s.heading, s.body, foot)
	root.Style.Background = widget.Paper
	s.Doc.SetRoot(s.cfg.Chrome.Wrap(s.Doc, root))
	s.loadMarks()
	s.reload()
	s.Doc.ResetFrame()
	s.Doc.Invalidate()
}

// loadMarks reads which days of the shown month have events.
func (s *Screen) loadMarks() {
	marks, err := s.cfg.Agenda.Marks(s.cal.Year, s.cal.Month)
	if err != nil {
		s.fail(err)
		return
	}
	s.marks = marks
}

// Select shows another day (and its month).
func (s *Screen) Select(day time.Time) {
	s.selected, s.page = dateOf(day), 0
	if s.cal != nil && (s.cal.Month != s.selected.Month() || s.cal.Year != s.selected.Year()) {
		s.cal.Year, s.cal.Month = s.selected.Year(), s.selected.Month()
		s.loadMarks()
	}
	if s.cal != nil {
		s.cal.Selected = s.selected
	}
	s.reload()
	s.Doc.Invalidate()
}

// GoTo shows a page of the day's events (clamped).
func (s *Screen) GoTo(page int) {
	if last := s.pages() - 1; page > last {
		page = last
	}
	if page < 0 {
		page = 0
	}
	if page == s.page {
		return
	}
	s.page = page
	s.fill()
	s.Doc.Invalidate()
}

func (s *Screen) pages() int {
	if len(s.items) == 0 {
		return 1
	}
	return (len(s.items) + rowsPerPage - 1) / rowsPerPage
}

func (s *Screen) reload() {
	items, err := s.cfg.Agenda.Day(s.selected)
	if err != nil {
		s.fail(err)
		items = nil
	}
	s.items = items
	if s.page >= s.pages() {
		s.page = s.pages() - 1
	}
	s.fill()
}

func (s *Screen) fill() {
	s.heading.Text = s.heading1()
	s.body.ClearChildren()
	from := s.page * rowsPerPage
	to := from + rowsPerPage
	if to > len(s.items) {
		to = len(s.items)
	}
	if len(s.items) == 0 {
		empty := widget.NewLabel(s.t("cal.empty"))
		empty.Style.Color, empty.Style.Padding = widget.Muted, geom.All(28)
		s.body.AppendChild(empty)
	}
	for _, o := range s.items[from:to] {
		s.body.AppendChild(s.row(o))
	}
	s.indicator.Text = s.t("cal.page", s.page+1, s.pages())
	s.prev.Disabled = s.page == 0
	s.next.Disabled = s.page >= s.pages()-1
}

func (s *Screen) row(o domain.Occurrence) widget.Widget {
	when := s.t("cal.allday")
	if !o.Event.AllDay {
		loc := s.cfg.Agenda.Loc()
		when = o.Start.In(loc).Format("15:04") + "–" + o.End.In(loc).Format("15:04")
	}
	title := o.Event.Title
	if title == "" {
		title = s.t("cal.untitled")
	}
	time1 := widget.NewLabel(when)
	time1.Style.Bold, time1.Style.Width = true, widget.Px(260)
	name := widget.NewLabel(title)
	name.Style.FlexGrow = 1
	cells := []widget.Widget{time1, name}
	if o.Event.Location != "" {
		loc := widget.NewLabel(o.Event.Location)
		loc.Style.FontSize, loc.Style.Color = widget.FontSmall, widget.Muted
		cells = append(cells, loc)
	}
	row := widget.NewRow(10, cells...)
	row.ID = "event-" + o.Event.ID
	row.Style.Padding = geom.XY(12, 18)
	row.Style.AlignItems = widget.AlignCenter
	row.Style.BorderColor, row.Style.BorderWidth, row.Style.BorderRadius = widget.Mid, 2, widget.Radius
	paper(row)
	id := o.Event.ID
	row.OnClick = func(*widget.Event) { s.Edit(id) }
	return row
}

// Key turns the pages of the day's events with the reader's page keys.
func (s *Screen) Key(previous bool) bool {
	if s.editor != nil {
		return false
	}
	if previous {
		s.GoTo(s.page - 1)
	} else {
		s.GoTo(s.page + 1)
	}
	return true
}
