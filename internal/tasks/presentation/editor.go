package presentation

import (
	"time"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/tasks/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

const notesRows = 6

// editor is the screen of one task: the changes live in task until Save.
type editor struct {
	task             domain.Task
	title            *widget.Input
	notes            *widget.TextArea
	dueLabel, note   *widget.Label
	important, myDay *widget.Toggle
	today            time.Time
}

// Edit opens a task.
func (s *Screen) Edit(id string) {
	t, err := s.cfg.Checklist.Get(id)
	if err != nil {
		s.fail(err)
		return
	}
	e := &editor{task: t, today: s.cfg.Checklist.Today()}
	s.editor = e

	back := widget.NewButton(s.t("tasks.back"))
	back.ID = "edit-back"
	back.OnClick = func(*widget.Event) { s.showList() }
	save := widget.NewButton(s.t("tasks.save"))
	save.ID = "edit-save"
	save.OnClick = func(*widget.Event) { s.SaveEdit() }
	del := widget.NewButton(s.t("tasks.delete"))
	del.ID = "edit-delete"
	del.OnClick = func(*widget.Event) { s.DeleteEdit() }
	spacer := widget.NewLabel("")
	spacer.Style.FlexGrow = 1
	bar := widget.NewRow(12, back, spacer, del, save)
	bar.Style.Padding = geom.XY(28, 28)
	paper(bar)

	e.title = widget.NewInput(s.t("tasks.title.field"))
	e.title.ID, e.title.Value = "edit-title", t.Title
	e.title.Style.FlexGrow = 1
	e.notes = widget.NewTextArea(s.t("tasks.notes"), notesRows)
	e.notes.ID, e.notes.Value = "edit-notes", t.Notes
	e.notes.Style.Width = widget.Pct(100)

	e.important = widget.NewSwitch(s.t("tasks.important"), t.Important)
	e.important.ID = "edit-important"
	e.important.OnChange = func(*widget.Event) { e.task.Important = e.important.Checked }
	e.myDay = widget.NewSwitch(s.t("tasks.myday"), !t.MyDay.IsZero() && domain.Day(t.MyDay).Equal(e.today))
	e.myDay.ID = "edit-myday"
	e.myDay.OnChange = func(*widget.Event) {
		e.task.MyDay = time.Time{}
		if e.myDay.Checked {
			e.task.MyDay = e.today
		}
	}

	e.dueLabel = widget.NewLabel("")
	s.showDue()
	days := widget.NewRow(10,
		s.dueButton("due-today", "tasks.due.today", 0),
		s.dueButton("due-tomorrow", "tasks.due.tomorrow", 1),
		s.dueButton("due-week", "tasks.due.week", 7),
		s.dueButton("due-clear", "tasks.due.clear", -1))
	days.Style.FlexWrap = true

	e.note = widget.NewLabel(s.notice)
	e.note.Style.FontSize, e.note.Style.Color = widget.FontSmall, widget.Muted

	form := widget.NewColumn(16, e.title, e.notes, e.important, e.myDay, e.dueLabel, days, e.note)
	form.Style.Padding = geom.XY(28, 28)
	form.Style.FlexGrow = 1
	root := widget.NewColumn(0, bar, form)
	root.Style.Background = widget.Paper
	s.Doc.SetRoot(root)
	s.Doc.ResetFrame()
	s.Doc.Invalidate()
}

// showDue writes the due date of the task being edited under its label.
func (s *Screen) showDue() {
	e := s.editor
	text := s.t("tasks.due.none")
	if !e.task.Due.IsZero() {
		text = e.task.Due.Format("02/01/2006")
	}
	e.dueLabel.Text = s.t("tasks.due.label") + ": " + text
}

// dueButton sets the due date to some days from today (a negative number clears it).
func (s *Screen) dueButton(id, key string, days int) *widget.Button {
	b := widget.NewButton(s.t(key))
	b.ID = id
	b.OnClick = func(*widget.Event) {
		e := s.editor
		if days < 0 {
			e.task.Due = time.Time{}
		} else {
			e.task.Due = e.today.AddDate(0, 0, days)
		}
		s.showDue()
		s.Doc.Invalidate()
	}
	return b
}

// SaveEdit keeps the changes and goes back to the list.
func (s *Screen) SaveEdit() {
	e := s.editor
	if e == nil {
		return
	}
	e.task.Title, e.task.Notes = e.title.Value, e.notes.Value
	if err := s.cfg.Checklist.Save(e.task); err != nil {
		e.note.Text = s.t("tasks.failed", err.Error())
		s.Doc.Invalidate()
		return
	}
	s.notice = ""
	s.showList()
}

// DeleteEdit deletes the task and goes back to the list.
func (s *Screen) DeleteEdit() {
	e := s.editor
	if e == nil {
		return
	}
	if err := s.cfg.Checklist.Delete(e.task.ID); err != nil {
		e.note.Text = s.t("tasks.failed", err.Error())
		s.Doc.Invalidate()
		return
	}
	s.notice = ""
	s.showList()
}
