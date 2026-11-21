// Package presentation draws the Calendar app like the calendar of Outlook: a toolbar
// (Today, previous and next, the period, the view and New) over the Day, Work week,
// Week or Month view, and a form to edit one event. It is built only from the
// toolkit's widgets (MonthView, TimeGrid) and holds no rules: the Agenda use case does.
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
	Span(first time.Time, n int) ([][]domain.Occurrence, error)
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
	// Importer brings in .ics files and addresses (nil: no Import button).
	Importer Importer
	// Flush paints pending changes now, before a slow call (nil: no flush).
	Flush func()
	// Connect opens the sign-in to an online account (Google or Microsoft); Account is the address
	// of the one already connected ("" when there is none). Nil Connect: no Account button.
	Connect func()
	Account string
	// Sync brings the events of the connected account in and says what it brought, already
	// translated (nil: no Sync button).
	Sync func() (string, error)
}

// View is how the schedule is shown.
type View int

// The views, as in Outlook.
const (
	MonthView View = iota
	DayView
	WorkWeekView
	WeekView
)

var views = []struct {
	view View
	key  string
}{{DayView, "cal.view.day"}, {WorkWeekView, "cal.view.workweek"}, {WeekView, "cal.view.week"}, {MonthView, "cal.view.month"}}

// Screen is the Calendar app's screen.
type Screen struct {
	Doc *widget.Document

	cfg      Config
	view     View
	selected time.Time // the day the view is about, a date at midnight UTC
	notice   string
	editor   *editor
	locale   widget.Locale

	title *widget.Label
	note  *widget.Label
}

// NewScreen builds the screen on today's month.
func NewScreen(cfg Config) *Screen {
	s := &Screen{cfg: cfg, notice: cfg.Notice, view: MonthView}
	s.Doc = cfg.Doc
	if s.Doc == nil {
		s.Doc = widget.NewDocument(cfg.Width, cfg.Height)
		s.Doc.Keyboard = cfg.Keyboard
	}
	s.locale = widget.LocaleEN
	if cfg.Tr.Lang() == "pt-BR" {
		s.locale = widget.LocalePTBR
	}
	s.selected = dateOf(cfg.Agenda.Today())
	s.show()
	return s
}

// dateOf is the calendar date of t as a midnight UTC (the widgets work with dates).
func dateOf(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func (s *Screen) t(key string, args ...interface{}) string { return s.cfg.Tr.T(key, args...) }

// Selected is the day the view is about.
func (s *Screen) Selected() time.Time { return s.selected }

// CurrentView is the view being shown.
func (s *Screen) CurrentView() View { return s.view }

// Editing is the id of the event being edited ("" while the schedule shows).
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

// ----- the period of each view -----

func (s *Screen) weekStart(first time.Weekday) time.Time {
	offset := (int(s.selected.Weekday()) - int(first) + 7) % 7
	return s.selected.AddDate(0, 0, -offset)
}

// days are the columns of a day, work week or week view.
func (s *Screen) days() []time.Time {
	var start time.Time
	n := 1
	switch s.view {
	case DayView:
		start = s.selected
	case WorkWeekView:
		start, n = s.weekStart(time.Monday), 5
	default:
		start, n = s.weekStart(s.locale.FirstWeekday), 7
	}
	out := make([]time.Time, n)
	for i := range out {
		out[i] = start.AddDate(0, 0, i)
	}
	return out
}

// monthStart is the first cell of the month grid that shows the selected day.
func (s *Screen) monthStart() time.Time {
	first := time.Date(s.selected.Year(), s.selected.Month(), 1, 0, 0, 0, 0, time.UTC)
	offset := (int(first.Weekday()) - int(s.locale.FirstWeekday) + 7) % 7
	return first.AddDate(0, 0, -offset)
}

func (s *Screen) period() string {
	switch s.view {
	case MonthView:
		return fmt.Sprintf("%s %d", s.locale.Months[s.selected.Month()-1], s.selected.Year())
	case DayView:
		d := s.selected
		return fmt.Sprintf("%s, %02d/%02d/%d", s.locale.Weekdays[d.Weekday()], d.Day(), int(d.Month()), d.Year())
	}
	days := s.days()
	a, b := days[0], days[len(days)-1]
	return fmt.Sprintf("%02d/%02d – %02d/%02d/%d", a.Day(), int(a.Month()), b.Day(), int(b.Month()), b.Year())
}

// ----- drawing -----

func (s *Screen) viewName(v View) string {
	for _, x := range views {
		if x.view == v {
			return s.t(x.key)
		}
	}
	return ""
}

func (s *Screen) toolbar() widget.Widget {
	today := widget.NewButton(s.t("cal.today"))
	today.ID = "cal-today"
	today.OnClick = func(*widget.Event) { s.Select(dateOf(s.cfg.Agenda.Today())) }
	prev, next := widget.NewIconButton(gfx.IconChevronLeft), widget.NewIconButton(gfx.IconChevronRight)
	prev.ID, next.ID = "cal-prev", "cal-next"
	prev.OnClick = func(*widget.Event) { s.Move(-1) }
	next.OnClick = func(*widget.Event) { s.Move(1) }

	s.title = widget.NewLabel(s.period())
	s.title.ID = "cal-title"
	s.title.Style.Bold, s.title.Style.FlexGrow = true, 1
	s.title.Style.TextAlign = gfx.AlignCenter

	var items []widget.MenuItem
	for _, v := range views {
		v := v
		items = append(items, widget.MenuItem{Label: s.t(v.key), OnSelect: func() { s.SetView(v.view) }})
	}
	menu := widget.NewDropdown(s.viewName(s.view), items...)
	menu.ID = "cal-views"
	add := widget.NewButton(s.t("cal.new"))
	add.ID = "cal-new"
	add.OnClick = func(*widget.Event) { s.NewEvent() }

	bar := widget.NewRow(10, today, prev, next, s.title, menu, add)
	bar.Style.Padding = geom.XY(20, 20)
	bar.Style.AlignItems = widget.AlignCenter
	return paper(bar)
}

// show draws the toolbar and the current view.
func (s *Screen) show() {
	s.editor = nil
	body, err := s.body()
	if err != nil {
		s.fail(err)
		body = widget.NewColumn(0)
	}
	s.note = widget.NewLabel(s.notice)
	s.note.Style.FontSize, s.note.Style.Color = widget.FontSmall, widget.Muted
	s.note.Style.FlexGrow = 1
	foot := widget.NewRow(12, s.note)
	if s.cfg.Importer != nil {
		imp := widget.NewButton(s.t("cal.import"))
		imp.ID = "cal-import"
		imp.OnClick = func(*widget.Event) { s.ShowImport() }
		foot.AppendChild(imp)
	}
	if s.cfg.Connect != nil {
		label := s.t("cal.account.connect")
		if s.cfg.Account != "" {
			label = s.t("cal.account.connected", s.cfg.Account)
		}
		acc := widget.NewButton(label)
		acc.ID = "cal-account"
		acc.OnClick = func(*widget.Event) { s.cfg.Connect() }
		foot.AppendChild(acc)
	}
	if s.cfg.Sync != nil {
		sync := widget.NewButton(s.t("cal.sync"))
		sync.ID = "cal-sync"
		sync.OnClick = func(*widget.Event) { s.SyncNow() }
		foot.AppendChild(sync)
	}
	foot.Style.AlignItems = widget.AlignCenter
	foot.Style.Padding = geom.XY(8, 20)
	root := widget.NewColumn(0, s.toolbar(), body, foot)
	root.Style.Background = widget.Paper
	s.Doc.SetRoot(s.cfg.Chrome.Wrap(s.Doc, root))
	s.Doc.ResetFrame()
	s.Doc.Invalidate()
}

func (s *Screen) body() (widget.Widget, error) {
	if s.view == MonthView {
		return s.monthBody()
	}
	return s.gridBody()
}

func eventText(o domain.Occurrence, loc *time.Location) string {
	if o.Event.AllDay {
		return o.Event.Title
	}
	return o.Start.In(loc).Format("15:04") + " " + o.Event.Title
}

func (s *Screen) monthBody() (widget.Widget, error) {
	start := s.monthStart()
	span, err := s.cfg.Agenda.Span(start, 42)
	if err != nil {
		return nil, err
	}
	loc := s.cfg.Agenda.Loc()
	mv := widget.NewMonthView(s.selected)
	mv.ID = "cal-month"
	mv.Locale = s.locale
	mv.Today = dateOf(s.cfg.Agenda.Today())
	mv.Entries = func(day time.Time) []string {
		i := int(day.Sub(start).Hours() / 24)
		if i < 0 || i >= len(span) {
			return nil
		}
		out := make([]string, len(span[i]))
		for k, o := range span[i] {
			out[k] = eventText(o, loc)
		}
		return out
	}
	// a tap on a day selects it; a second tap opens it in the Day view
	mv.OnChange = func(*widget.Event) { s.selected = dateOf(mv.Selected); s.title.Text = s.period() }
	mv.OnOpen = func(*widget.Event) { s.selected = dateOf(mv.Selected); s.SetView(DayView) }
	mv.OnMonth = func(*widget.Event) { s.selected = dateOf(mv.Selected); s.show() }
	return mv, nil
}

func (s *Screen) gridBody() (widget.Widget, error) {
	days := s.days()
	span, err := s.cfg.Agenda.Span(days[0], len(days))
	if err != nil {
		return nil, err
	}
	loc := s.cfg.Agenda.Loc()
	g := widget.NewTimeGrid(days...)
	g.ID = "cal-grid"
	g.Locale = s.locale
	g.Today = dateOf(s.cfg.Agenda.Today())
	g.Now = func() time.Time { return time.Now().In(loc) }
	g.Events, g.FirstHour, g.LastHour = gridEvents(span, loc)
	g.OnEvent = func(id string) { s.Edit(id) }
	g.OnSlot = func(at time.Time) { s.NewEventAt(at) }
	g.OnDay = func(day time.Time) { s.selected = dateOf(day); s.SetView(DayView) }
	return g, nil
}

// ----- navigation -----

// SetView switches the view (keeping the selected day).
func (s *Screen) SetView(v View) {
	s.view = v
	s.show()
}

// Select makes a day the one the view is about.
func (s *Screen) Select(day time.Time) {
	s.selected = dateOf(day)
	s.show()
}

// Move goes to the next (1) or the previous (-1) period of the view.
func (s *Screen) Move(delta int) {
	switch s.view {
	case MonthView:
		first := time.Date(s.selected.Year(), s.selected.Month()+time.Month(delta), 1, 0, 0, 0, 0, time.UTC)
		day := s.selected.Day()
		if last := first.AddDate(0, 1, -1).Day(); day > last {
			day = last
		}
		s.selected = first.AddDate(0, 0, day-1)
	case DayView:
		s.selected = s.selected.AddDate(0, 0, delta)
	default:
		s.selected = s.selected.AddDate(0, 0, 7*delta)
	}
	s.show()
}

// Key turns the period with the reader's page keys.
func (s *Screen) Key(previous bool) bool {
	if s.editor != nil {
		return false
	}
	if previous {
		s.Move(-1)
	} else {
		s.Move(1)
	}
	return true
}

// gridEvents turns the occurrences of the shown days into blocks of the grid, and the
// hours to draw: 7 to 21, widened to take in an event that is earlier or later.
func gridEvents(span [][]domain.Occurrence, loc *time.Location) ([]widget.GridEvent, int, int) {
	first, last := 7, 21
	var events []widget.GridEvent
	seen := map[string]bool{}
	for _, list := range span {
		for _, o := range list {
			key := o.Event.ID + o.Start.String()
			if seen[key] {
				continue
			}
			seen[key] = true
			ev := gridEvent(o, loc)
			first, last = widenHours(ev, first, last)
			events = append(events, ev)
		}
	}
	return events, first, last
}

// widenHours stretches the hours to draw so a timed event that is earlier or later fits.
func widenHours(ev widget.GridEvent, first, last int) (int, int) {
	if ev.AllDay {
		return first, last
	}
	if h := ev.Start.Hour(); h < first {
		first = h
	}
	if h := ev.End.Hour() + 1; h > last && h <= 24 {
		last = h
	}
	return first, last
}

// gridEvent is one occurrence as a block: all-day ones keep their dates, the others
// the wall clock of the person's zone.
func gridEvent(o domain.Occurrence, loc *time.Location) widget.GridEvent {
	ev := widget.GridEvent{ID: o.Event.ID, Title: o.Event.Title, AllDay: o.Event.AllDay}
	if o.Event.AllDay {
		ev.Start, ev.End = o.Start, o.End
		return ev
	}
	ev.Start, ev.End = o.Start.In(loc), o.End.In(loc)
	ev.Detail = ev.Start.Format("15:04") + "–" + ev.End.Format("15:04")
	if o.Event.Location != "" {
		ev.Detail += " " + o.Event.Location
	}
	return ev
}

// SyncNow brings the events of the connected account in, telling first that it is working (the
// call blocks on the network), and shows the calendar again with the result.
func (s *Screen) SyncNow() {
	if s.cfg.Sync == nil {
		return
	}
	s.notice = s.t("cal.sync.working")
	if s.note != nil {
		s.note.Text = s.notice
		s.Doc.Invalidate()
	}
	if s.cfg.Flush != nil {
		s.cfg.Flush()
	}
	msg, err := s.cfg.Sync()
	if err != nil {
		msg = s.t("cal.failed", err.Error())
	}
	s.notice = msg
	s.show()
}

// Importer is the importer the screen was given (nil when there is none).
func (s *Screen) Importer() Importer { return s.cfg.Importer }
