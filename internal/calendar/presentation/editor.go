package presentation

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"go-pocketbook-base/internal/calendar/application"
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
	"go-pocketbook-base/internal/toolkit/widget"
)

const descriptionRows = 5

// editor is the screen of one event ("" id: a new one).
type editor struct {
	id              string
	day             time.Time
	title, location *widget.Input
	description     *widget.TextArea
	start, end      *widget.Input
	allDay          *widget.Toggle
	rule            string
	repeat          *widget.Dropdown
	remind          time.Duration
	reminder        *widget.Dropdown
	date, note      *widget.Label
}

// NewEvent opens an empty event on the selected day, at the next full hour.
func (s *Screen) NewEvent() {
	start := time.Now().In(s.cfg.Agenda.Loc()).Hour() + 1
	if start > 22 {
		start = 22
	}
	s.openEditor(&editor{day: s.selected}, application.Entry{Day: s.selected, Start: start * 60, End: (start + 1) * 60})
}

// NewEventAt opens an empty event starting at a time (a tap on an empty slot of the grid).
func (s *Screen) NewEventAt(at time.Time) {
	start := at.Hour()*60 + at.Minute()
	end := start + 60
	if end > 24*60-1 {
		end = 24*60 - 1
	}
	day := dateOf(at)
	s.openEditor(&editor{day: day}, application.Entry{Day: day, Start: start, End: end})
}

// Edit opens an event.
func (s *Screen) Edit(id string) {
	e, err := s.cfg.Agenda.Get(id)
	if err != nil {
		s.fail(err)
		return
	}
	in := application.Entry{Title: e.Title, Location: e.Location, Description: e.Description, AllDay: e.AllDay, Recurrence: e.Recurrence}
	if len(e.Reminders) > 0 {
		in.Reminder = e.Reminders[0]
	}
	if e.AllDay {
		in.Day = dateOf(e.Start)
	} else {
		loc := s.cfg.Agenda.Loc()
		start, end := e.Start.In(loc), e.End.In(loc)
		in.Day = dateOf(start)
		in.Start, in.End = start.Hour()*60+start.Minute(), end.Hour()*60+end.Minute()
	}
	s.openEditor(&editor{id: id, day: in.Day}, in)
}

func (s *Screen) field(id, placeholder, value string) *widget.Input {
	in := widget.NewInput(placeholder)
	in.ID, in.Value = id, value
	return in
}

func clock(minutes int) string { return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60) }

func (s *Screen) openEditor(e *editor, in application.Entry) {
	s.editor = e
	back := widget.NewButton(s.t("cal.back"))
	back.ID = "edit-back"
	back.OnClick = func(*widget.Event) { s.show() }
	save := widget.NewButton(s.t("cal.save"))
	save.ID = "edit-save"
	save.OnClick = func(*widget.Event) { s.SaveEdit() }
	bar := widget.NewRow(12, back, spacer())
	if e.id != "" {
		del := widget.NewButton(s.t("cal.delete"))
		del.ID = "edit-delete"
		del.OnClick = func(*widget.Event) { s.DeleteEdit() }
		bar.AppendChild(del)
	}
	bar.AppendChild(save)
	bar.Style.Padding = geom.XY(28, 28)
	paper(bar)

	e.title = s.field("edit-title", s.t("cal.field.title"), in.Title)
	e.location = s.field("edit-location", s.t("cal.field.location"), in.Location)
	e.description = widget.NewTextArea(s.t("cal.field.description"), descriptionRows)
	e.description.ID, e.description.Value = "edit-description", in.Description
	e.description.Style.Width = widget.Pct(100)
	e.start = s.field("edit-start", s.t("cal.field.start"), clock(in.Start))
	e.end = s.field("edit-end", s.t("cal.field.end"), clock(in.End))
	e.allDay = widget.NewSwitch(s.t("cal.field.allday"), in.AllDay)
	e.allDay.ID = "edit-allday"
	e.rule = in.Recurrence
	e.repeat = s.repeatMenu(e)
	e.remind = in.Reminder
	e.reminder = s.reminderMenu(e)

	e.date = widget.NewLabel("")
	s.showDate()
	before, after := widget.NewIconButton(gfx.IconChevronLeft), widget.NewIconButton(gfx.IconChevronRight)
	before.ID, after.ID = "edit-prevday", "edit-nextday"
	before.OnClick = func(*widget.Event) { s.moveDay(-1) }
	after.OnClick = func(*widget.Event) { s.moveDay(1) }
	e.date.Style.FlexGrow = 1
	dateRow := widget.NewRow(12, e.date, before, after)
	dateRow.Style.AlignItems = widget.AlignCenter

	e.note = widget.NewLabel(s.notice)
	e.note.Style.FontSize, e.note.Style.Color = widget.FontSmall, widget.Muted

	form := widget.NewColumn(16, e.title, e.location, dateRow, e.allDay, widget.NewRow(12, e.start, e.end), e.repeat, e.reminder, e.description, e.note)
	form.Style.Padding = geom.XY(28, 28)
	form.Style.FlexGrow = 1
	root := widget.NewColumn(0, bar, form)
	root.Style.Background = widget.Paper
	s.Doc.SetRoot(s.cfg.Chrome.Wrap(s.Doc, root))
	s.Doc.ResetFrame()
	s.Doc.Invalidate()
}

// The repetitions the form offers, by their rule ("" is none).
var repeatRules = []struct{ rule, key string }{
	{"", "cal.repeat.none"}, {"FREQ=DAILY", "cal.repeat.daily"}, {"FREQ=WEEKLY", "cal.repeat.weekly"},
	{"FREQ=MONTHLY", "cal.repeat.monthly"}, {"FREQ=YEARLY", "cal.repeat.yearly"},
}

// repeatLabel names the repetition of a rule; a rule the form does not offer (it
// came from a server or a file) is shown as custom and kept until another is chosen.
func (s *Screen) repeatLabel(rule string) string {
	for _, r := range repeatRules {
		if r.rule == rule {
			return s.t("cal.repeat") + ": " + s.t(r.key)
		}
	}
	return s.t("cal.repeat") + ": " + s.t("cal.repeat.custom")
}

func (s *Screen) repeatMenu(e *editor) *widget.Dropdown {
	var items []widget.MenuItem
	for _, r := range repeatRules {
		r := r
		items = append(items, widget.MenuItem{Label: s.t(r.key), Value: r.rule, OnSelect: func() {
			e.rule = r.rule
			e.repeat.Text = s.repeatLabel(r.rule)
		}})
	}
	d := widget.NewDropdown(s.repeatLabel(e.rule), items...)
	d.ID = "edit-repeat"
	return d
}

// The reminders the form offers (0 is none).
var reminderChoices = []struct {
	before time.Duration
	key    string
}{
	{0, "cal.remind.none"}, {5 * time.Minute, "cal.remind.5m"}, {15 * time.Minute, "cal.remind.15m"},
	{30 * time.Minute, "cal.remind.30m"}, {time.Hour, "cal.remind.1h"}, {24 * time.Hour, "cal.remind.1d"},
}

// reminderLabel names a reminder; one the form does not offer shows its minutes.
func (s *Screen) reminderLabel(before time.Duration) string {
	for _, r := range reminderChoices {
		if r.before == before {
			return s.t("cal.remind") + ": " + s.t(r.key)
		}
	}
	return s.t("cal.remind") + ": " + s.t("cal.remind.minutes", int(before.Minutes()))
}

func (s *Screen) reminderMenu(e *editor) *widget.Dropdown {
	var items []widget.MenuItem
	for _, r := range reminderChoices {
		r := r
		items = append(items, widget.MenuItem{Label: s.t(r.key), OnSelect: func() {
			e.remind = r.before
			e.reminder.Text = s.reminderLabel(r.before)
		}})
	}
	d := widget.NewDropdown(s.reminderLabel(e.remind), items...)
	d.ID = "edit-reminder"
	return d
}

func spacer() *widget.Label {
	l := widget.NewLabel("")
	l.Style.FlexGrow = 1
	return l
}

func (s *Screen) showDate() {
	d := s.editor.day
	s.editor.date.Text = s.t("cal.date", fmt.Sprintf("%s, %02d/%02d/%d", s.locale.Weekdays[d.Weekday()], d.Day(), int(d.Month()), d.Year()))
}

func (s *Screen) moveDay(delta int) {
	s.editor.day = s.editor.day.AddDate(0, 0, delta)
	s.showDate()
	s.Doc.Invalidate()
}

// parseClock reads "H:MM" or "HH:MM" as minutes after midnight.
func parseClock(text string) (int, bool) {
	h, m, ok := splitClock(strings.TrimSpace(text))
	if !ok || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

func splitClock(text string) (int, int, bool) {
	parts := strings.Split(text, ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	return h, m, err1 == nil && err2 == nil
}

// entry reads the form.
func (s *Screen) entry() (application.Entry, bool) {
	e := s.editor
	in := application.Entry{Title: e.title.Value, Location: e.location.Value, Description: e.description.Value, Day: e.day, AllDay: e.allDay.Checked, Recurrence: e.rule, Reminder: e.remind}
	if in.AllDay {
		return in, true
	}
	var ok1, ok2 bool
	in.Start, ok1 = parseClock(e.start.Value)
	in.End, ok2 = parseClock(e.end.Value)
	return in, ok1 && ok2
}

// SaveEdit keeps the event and shows its day.
func (s *Screen) SaveEdit() {
	e := s.editor
	if e == nil {
		return
	}
	in, ok := s.entry()
	if !ok {
		e.note.Text = s.t("cal.badtime")
		s.Doc.Invalidate()
		return
	}
	var err error
	if e.id == "" {
		_, err = s.cfg.Agenda.Add(in)
	} else {
		err = s.cfg.Agenda.Change(e.id, in)
	}
	if err != nil {
		e.note.Text = s.t("cal.failed", err.Error())
		s.Doc.Invalidate()
		return
	}
	s.notice, s.selected = "", dateOf(in.Day)
	s.show()
}

// DeleteEdit deletes the event and shows its day.
func (s *Screen) DeleteEdit() {
	e := s.editor
	if e == nil || e.id == "" {
		return
	}
	if err := s.cfg.Agenda.Delete(e.id); err != nil {
		e.note.Text = s.t("cal.failed", err.Error())
		s.Doc.Invalidate()
		return
	}
	s.notice = ""
	s.show()
}
