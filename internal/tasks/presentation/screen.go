// Package presentation draws the Tasks app: the views (My Day, Important, Planned,
// Overdue, All), the tasks of a view a page at a time, and one task to edit. It is
// built only from the toolkit's widgets and holds no rules: the Checklist use case does.
package presentation

import (
	"embed"
	"strconv"
	"strings"
	"time"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/localization/i18n"
	"go-pocketbook-base/internal/tasks/application"
	"go-pocketbook-base/internal/tasks/domain"
	"go-pocketbook-base/internal/toolkit/gfx"
	"go-pocketbook-base/internal/toolkit/widget"
)

//go:embed locales/*.json
var locales embed.FS

// NewTranslator loads the Tasks app's texts (pt-BR first, English fallback) in lang.
func NewTranslator(lang string) *i18n.Bundle {
	tr := i18n.New()
	if err := tr.AddFS(locales, "locales"); err != nil {
		panic(err) // embedded files: an error is a programming mistake
	}
	tr.SetLang(lang)
	return tr
}

// Checklist is what the screen needs of the Tasks context (application.Checklist).
type Checklist interface {
	View(v domain.View) ([]domain.Task, domain.Counts, error)
	Add(title string, v domain.View) (domain.Task, error)
	Toggle(id string) error
	Get(id string) (domain.Task, error)
	Save(t domain.Task) error
	Delete(id string) error
	SetImportant(id string, on bool) error
	SetMyDay(id string, on bool) error
	SetDue(id string, day time.Time) error
	Today() time.Time
}

var _ Checklist = application.Checklist{}

// Config is everything the screen needs.
type Config struct {
	Width, Height int
	// Doc is the document to draw on (a new one when nil).
	Doc       *widget.Document
	Keyboard  widget.Keyboard
	Tr        *i18n.Bundle
	Checklist Checklist
	// Notice is shown at the first paint.
	Notice string
}

const rowsPerPage = 8

// The views, in the order of the chips.
var views = []domain.View{domain.ViewMyDay, domain.ViewImportant, domain.ViewPlanned, domain.ViewOverdue, domain.ViewAll}

// Screen is the Tasks app's screen.
type Screen struct {
	Doc *widget.Document

	cfg    Config
	view   domain.View
	counts domain.Counts
	tasks  []domain.Task // the tasks of the view, all pages
	page   int
	notice string
	editor *editor

	body            *widget.Panel
	note, indicator *widget.Label
	prev, next      *widget.Button
	add             *widget.Input
	chips           []*widget.Chip
}

// NewScreen builds the screen on the first view (My Day).
func NewScreen(cfg Config) *Screen {
	s := &Screen{cfg: cfg, view: domain.ViewMyDay, notice: cfg.Notice}
	s.Doc = cfg.Doc
	if s.Doc == nil {
		s.Doc = widget.NewDocument(cfg.Width, cfg.Height)
		s.Doc.Keyboard = cfg.Keyboard
	}
	s.showList()
	return s
}

func (s *Screen) t(key string, args ...interface{}) string { return s.cfg.Tr.T(key, args...) }

// View is the view being shown.
func (s *Screen) View() domain.View { return s.view }

// Page is the page of the list, counted from 0.
func (s *Screen) Page() int { return s.page }

// Editing is the id of the task being edited ("" while the list shows).
func (s *Screen) Editing() string {
	if s.editor == nil {
		return ""
	}
	return s.editor.task.ID
}

func paper(w widget.Widget) widget.Widget {
	w.Elem().Style.Background = widget.Paper
	return w
}

func (s *Screen) fail(err error) {
	s.notice = s.t("tasks.failed", err.Error())
	if s.note != nil {
		s.note.Text = s.notice
	}
	s.Doc.Invalidate()
}

func (s *Screen) count(v domain.View) int {
	switch v {
	case domain.ViewMyDay:
		return s.counts.MyDay
	case domain.ViewImportant:
		return s.counts.Important
	case domain.ViewPlanned:
		return s.counts.Planned
	case domain.ViewOverdue:
		return s.counts.Overdue
	}
	return 0
}

func (s *Screen) chipLabel(v domain.View) string {
	label := s.t("tasks.view." + string(v))
	if n := s.count(v); n > 0 {
		label += " (" + strconv.Itoa(n) + ")"
	}
	return label
}

func (s *Screen) showList() {
	s.editor = nil
	title := widget.NewLabel(s.t("tasks.title"))
	title.Style.FontSize, title.Style.Bold = widget.FontTitle, true
	title.Style.Padding = geom.XY(28, 28)
	paper(title)

	s.chips = s.chips[:0]
	row := widget.NewRow(10)
	row.Style.FlexWrap = true
	for _, v := range views {
		v := v
		c := widget.NewChip(s.chipLabel(v))
		c.ID = "view-" + string(v)
		c.Selected = v == s.view
		c.OnChange = func(*widget.Event) { s.Show(v) }
		s.chips = append(s.chips, c)
		row.AppendChild(c)
	}
	row.Style.Padding = geom.XY(0, 28)

	s.add = widget.NewInput(s.t("tasks.add"))
	s.add.ID = "task-add"
	s.add.Style.FlexGrow = 1
	s.add.OnSubmit = func(*widget.Event) { s.Add(s.add.Value) }
	addRow := widget.NewRow(0, s.add)
	addRow.Style.Padding = geom.XY(16, 28)

	s.body = widget.NewColumn(8)
	s.body.ID = "tasks-body"
	s.body.Style.FlexGrow, s.body.Style.Padding = 1, geom.XY(8, 28)

	s.note = widget.NewLabel(s.notice)
	s.note.Style.FontSize, s.note.Style.Color, s.note.Style.FlexGrow = widget.FontSmall, widget.Muted, 1
	s.indicator = widget.NewLabel("")
	s.indicator.Style.FontSize, s.indicator.Style.Bold = widget.FontSmall, true
	s.indicator.Style.TextAlign, s.indicator.Style.Width = gfx.AlignCenter, widget.Px(110)
	s.prev, s.next = widget.NewIconButton(gfx.IconChevronLeft), widget.NewIconButton(gfx.IconChevronRight)
	s.prev.ID, s.next.ID = "tasks-prev", "tasks-next"
	s.prev.OnClick = func(*widget.Event) { s.GoTo(s.page - 1) }
	s.next.OnClick = func(*widget.Event) { s.GoTo(s.page + 1) }
	foot := widget.NewRow(12, s.note, s.prev, s.indicator, s.next)
	foot.Style.Padding = geom.XY(14, 28)
	paper(foot)

	root := widget.NewColumn(0, title, row, addRow, s.body, foot)
	root.Style.Background = widget.Paper
	s.Doc.SetRoot(root)
	s.reload()
	s.Doc.ResetFrame()
	s.Doc.Invalidate()
}

// Show switches to a view, from its first page.
func (s *Screen) Show(v domain.View) {
	s.view, s.page = v, 0
	s.showList()
}

// GoTo shows a page of the list (clamped).
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
	if len(s.tasks) == 0 {
		return 1
	}
	return (len(s.tasks) + rowsPerPage - 1) / rowsPerPage
}

// reload reads the tasks of the view and shows the current page.
func (s *Screen) reload() {
	tasks, counts, err := s.cfg.Checklist.View(s.view)
	if err != nil {
		s.fail(err)
		return
	}
	s.tasks, s.counts = tasks, counts
	for i, v := range views {
		s.chips[i].Label = s.chipLabel(v)
	}
	if s.page >= s.pages() {
		s.page = s.pages() - 1
	}
	s.fill()
}

func (s *Screen) fill() {
	s.body.ClearChildren()
	from := s.page * rowsPerPage
	to := from + rowsPerPage
	if to > len(s.tasks) {
		to = len(s.tasks)
	}
	if len(s.tasks) == 0 {
		empty := widget.NewLabel(s.t("tasks.empty"))
		empty.Style.Color, empty.Style.Padding = widget.Muted, geom.All(28)
		s.body.AppendChild(empty)
	}
	for _, t := range s.tasks[from:to] {
		s.body.AppendChild(s.row(t))
	}
	s.indicator.Text = s.t("tasks.page", s.page+1, s.pages())
	s.prev.Disabled = s.page == 0
	s.next.Disabled = s.page >= s.pages()-1
}

func (s *Screen) row(t domain.Task) widget.Widget {
	check := widget.NewCheckbox("", t.Done)
	check.ID = "done-" + t.ID
	id := t.ID
	check.OnChange = func(*widget.Event) { s.Toggle(id) }

	name := t.Title
	if t.Important {
		name = "! " + name
	}
	title := widget.NewLabel(name)
	title.Style.Bold = t.Important
	if t.Done {
		title.Style.Color = widget.Muted
	}
	lines := []widget.Widget{title}
	if info := s.info(t); info != "" {
		l := widget.NewLabel(info)
		l.Style.FontSize, l.Style.Color = widget.FontSmall, widget.Muted
		lines = append(lines, l)
	}
	text := widget.NewColumn(2, lines...)
	text.Style.FlexGrow = 1
	text.ID = "task-" + t.ID
	text.OnClick = func(*widget.Event) { s.Edit(id) }

	row := widget.NewRow(8, check, text)
	row.Style.Padding = geom.XY(8, 14)
	row.Style.AlignItems = widget.AlignCenter
	row.Style.BorderColor, row.Style.BorderWidth, row.Style.BorderRadius = widget.Mid, 2, widget.Radius
	paper(row)
	return row
}

// info is the small line under a title: the due date and the steps.
func (s *Screen) info(t domain.Task) string {
	var parts []string
	if !t.Due.IsZero() {
		key := "tasks.due"
		if t.Overdue(s.cfg.Checklist.Today()) {
			key = "tasks.overdue"
		}
		parts = append(parts, s.t(key, t.Due.Format("02/01")))
	}
	if t.StepsTotal() > 0 {
		parts = append(parts, s.t("tasks.steps", t.StepsDone(), t.StepsTotal()))
	}
	return strings.Join(parts, "  ·  ")
}

// Add creates a task in the view being shown.
func (s *Screen) Add(title string) {
	if strings.TrimSpace(title) == "" {
		return
	}
	if _, err := s.cfg.Checklist.Add(title, s.view); err != nil {
		s.fail(err)
		return
	}
	s.notice, s.page = "", 0
	s.showList()
}

// Toggle ticks or reopens a task.
func (s *Screen) Toggle(id string) {
	if err := s.cfg.Checklist.Toggle(id); err != nil {
		s.fail(err)
		return
	}
	s.notice = ""
	s.showList()
}

// Key turns the pages of the list with the reader's page keys.
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
