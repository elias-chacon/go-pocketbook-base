package presentation

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"go-pocketbook-base/internal/tasks/application"
	"go-pocketbook-base/internal/tasks/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

var now = time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)

type memStore struct {
	lists []domain.List
	tasks map[string]domain.Task
	next  int
	err   error
}

func (m *memStore) SaveLists(_ string, l []domain.List) error { m.lists = l; return m.err }
func (m *memStore) Lists(string) ([]domain.List, error)       { return m.lists, m.err }
func (m *memStore) Save(ts ...domain.Task) error {
	if m.err != nil {
		return m.err
	}
	for _, t := range ts {
		if t.ID == "" {
			m.next++
			t.ID = "t" + strconv.Itoa(m.next)
		}
		m.tasks[t.ID] = t
	}
	return nil
}
func (m *memStore) Tasks(domain.TaskQuery) ([]domain.Task, error) {
	var out []domain.Task
	for _, t := range m.tasks {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, m.err
}
func (m *memStore) Get(id string) (domain.Task, error) {
	if t, ok := m.tasks[id]; ok {
		return t, nil
	}
	return domain.Task{}, domain.ErrNotFound
}
func (m *memStore) Delete(id string) error {
	if m.err != nil {
		return m.err
	}
	delete(m.tasks, id)
	return nil
}

func newScreen(titles ...string) (*Screen, *memStore) {
	st := &memStore{tasks: map[string]domain.Task{}}
	todo := application.Checklist{AccountID: "local", Store: st, Now: func() time.Time { return now }}
	for _, title := range titles {
		todo.Add(title, domain.ViewMyDay)
	}
	s := NewScreen(Config{Width: 1404, Height: 1872, Tr: NewTranslator("pt-BR"), Checklist: todo})
	s.Doc.Layout()
	return s, st
}

func byID(t *testing.T, s *Screen, id string) widget.Widget {
	t.Helper()
	var found widget.Widget
	s.Doc.Walk(func(w widget.Widget) bool {
		if w.Elem().ID == id {
			found = w
		}
		return true
	})
	if found == nil {
		t.Fatalf("no element %q", id)
	}
	return found
}

func tap(t *testing.T, s *Screen, id string) {
	t.Helper()
	s.Doc.Layout()
	w := byID(t, s, id)
	c := w.Elem().Bounds.Center()
	s.Doc.HandlePointer(widget.PointerDown, c.X*s.Doc.Scale, c.Y*s.Doc.Scale)
	s.Doc.HandlePointer(widget.PointerUp, c.X*s.Doc.Scale, c.Y*s.Doc.Scale)
	s.Doc.Layout()
}

func TestEveryTextIsTranslatedInBothLanguages(t *testing.T) {
	tr := NewTranslator("pt-BR")
	for _, lang := range []string{"pt-BR", "en"} {
		if missing := tr.Missing(lang); len(missing) != 0 {
			t.Errorf("%s lacks: %v", lang, missing)
		}
	}
}

func TestTheScreenStartsOnMyDayWithCounts(t *testing.T) {
	s, _ := newScreen("a", "b")
	if s.View() != domain.ViewMyDay {
		t.Errorf("view = %q", s.View())
	}
	if got := byID(t, s, "view-my-day").(*widget.Chip).Label; got != "Meu dia (2)" {
		t.Errorf("chip = %q", got)
	}
	byID(t, s, "task-t1")
	byID(t, s, "task-t2")
}

func TestAddingATaskPutsItInTheView(t *testing.T) {
	s, st := newScreen()
	s.Add("   ")
	if len(st.tasks) != 0 {
		t.Error("an empty title was added")
	}
	s.Add("Pagar a conta")
	if len(st.tasks) != 1 || st.tasks["t1"].MyDay.IsZero() {
		t.Fatalf("tasks = %+v", st.tasks)
	}
	byID(t, s, "task-t1")
	s.Show(domain.ViewImportant)
	s.Add("Urgente")
	if !st.tasks["t2"].Important {
		t.Error("a task added in Important is not starred")
	}
}

func TestTheCheckboxTicksAndTheViewsFollow(t *testing.T) {
	s, st := newScreen("a")
	tap(t, s, "done-t1")
	if !st.tasks["t1"].Done {
		t.Fatal("not done")
	}
	if len(s.tasks) != 0 {
		t.Errorf("a done task is still in My Day: %d", len(s.tasks))
	}
	s.Show(domain.ViewAll)
	tap(t, s, "done-t1")
	if st.tasks["t1"].Done {
		t.Error("not reopened")
	}
}

func TestPagesAndKeys(t *testing.T) {
	titles := make([]string, 11)
	for i := range titles {
		titles[i] = "tarefa " + strconv.Itoa(i)
	}
	s, _ := newScreen(titles...)
	if s.pages() != 2 {
		t.Fatalf("pages = %d", s.pages())
	}
	tap(t, s, "tasks-next")
	if s.Page() != 1 {
		t.Errorf("page = %d", s.Page())
	}
	if !s.Key(true) || s.Page() != 0 {
		t.Errorf("previous key: page = %d", s.Page())
	}
	s.GoTo(50)
	if s.Page() != 1 {
		t.Errorf("clamped = %d", s.Page())
	}
}

func TestEditingChangesAndSavesATask(t *testing.T) {
	s, st := newScreen("a")
	tap(t, s, "task-t1")
	if s.Editing() != "t1" {
		t.Fatal("not editing")
	}
	byID(t, s, "edit-title").(*widget.Input).Value = "Novo título"
	byID(t, s, "edit-notes").(*widget.TextArea).Value = "uma nota"
	tap(t, s, "edit-important")
	tap(t, s, "edit-myday")
	tap(t, s, "due-tomorrow")
	tap(t, s, "edit-save")
	got := st.tasks["t1"]
	if got.Title != "Novo título" || got.Notes != "uma nota" || !got.Important || !got.MyDay.IsZero() {
		t.Errorf("task = %+v", got)
	}
	if !got.Due.Equal(domain.Day(now).AddDate(0, 0, 1)) {
		t.Errorf("due = %v", got.Due)
	}
	if s.Editing() != "" {
		t.Error("still editing")
	}
}

func TestTheDueButtonsAndDelete(t *testing.T) {
	s, st := newScreen("a")
	s.Edit("t1")
	for _, id := range []string{"due-today", "due-week", "due-clear"} {
		tap(t, s, id)
	}
	if !s.editor.task.Due.IsZero() {
		t.Error("clear did not clear")
	}
	tap(t, s, "edit-delete")
	if len(st.tasks) != 0 || s.Editing() != "" {
		t.Errorf("delete: %d tasks, editing %q", len(st.tasks), s.Editing())
	}
	s.Edit("t1") // gone: a notice, no panic
	if !strings.Contains(s.notice, "Não foi possível") {
		t.Errorf("notice = %q", s.notice)
	}
	var none Screen
	none.SaveEdit()
	none.DeleteEdit()
}

func TestAnInvalidEditAndStoreErrorsAreExplained(t *testing.T) {
	s, st := newScreen("a")
	s.Edit("t1")
	byID(t, s, "edit-title").(*widget.Input).Value = ""
	tap(t, s, "edit-save")
	if s.Editing() == "" || !strings.Contains(s.editor.note.Text, "title") {
		t.Errorf("an empty title was saved: %q", s.editor.note.Text)
	}
	st.err = errors.New("disco")
	tap(t, s, "edit-delete")
	if !strings.Contains(s.editor.note.Text, "disco") {
		t.Errorf("note = %q", s.editor.note.Text)
	}
	s.showList()
	st.err = errors.New("falhou")
	s.Toggle("t1")
	if !strings.Contains(s.notice, "falhou") {
		t.Errorf("notice = %q", s.notice)
	}
	s.notice = ""
	s.Add("x")
	if !strings.Contains(s.notice, "falhou") {
		t.Errorf("notice = %q", s.notice)
	}
}

func checkElement(t *testing.T, s *Screen, where string, w widget.Widget, screenH float64) {
	t.Helper()
	e := w.Elem()
	if !e.Visible() {
		return
	}
	if e.Bounds.Y+e.Bounds.H > screenH+0.5 && e.ID != "" {
		t.Errorf("%s: %s overflows the screen (%v)", where, e.ID, e.Bounds)
	}
	if _, ok := w.(widget.PointerHandler); !ok || e.Disabled || e.Bounds.W <= 0 {
		return
	}
	if got := s.Doc.TargetAt(e.Bounds.Center()); got != w {
		t.Errorf("%s: %s (%T) does not get its own touch", where, e.ID, w)
	}
	if e.Bounds.H < 56 {
		t.Errorf("%s: %s is only %.0f px high", where, e.ID, e.Bounds.H)
	}
}

func TestEveryInteractiveElementReceivesItsOwnTouchAndNothingOverflows(t *testing.T) {
	titles := make([]string, 9)
	for i := range titles {
		titles[i] = "tarefa " + strconv.Itoa(i)
	}
	s, _ := newScreen(titles...)
	check := func(where string) {
		t.Helper()
		s.Doc.Layout()
		h := s.Doc.Root.Elem().Bounds.H
		s.Doc.Walk(func(w widget.Widget) bool {
			checkElement(t, s, where, w, h)
			return true
		})
	}
	check("list")
	s.Edit("t1")
	check("editor")
}

func TestStepsAreAddedTickedRemovedAndSaved(t *testing.T) {
	s, st := newScreen("a")
	s.Edit("t1")
	byID(t, s, "edit-title").(*widget.Input).Value = "editado"
	add := byID(t, s, "step-add").(*widget.Input)
	add.Value = "primeira"
	add.OnSubmit(nil)
	add = byID(t, s, "step-add").(*widget.Input)
	add.Value = "segunda"
	add.OnSubmit(nil)
	if got := byID(t, s, "edit-title").(*widget.Input).Value; got != "editado" {
		t.Errorf("the typed title was lost: %q", got)
	}
	tap(t, s, "step-0")
	tap(t, s, "step-del-1")
	byID(t, s, "step-add").(*widget.Input).OnSubmit(nil) // empty: nothing
	tap(t, s, "edit-save")
	got := st.tasks["t1"]
	if got.Title != "editado" || len(got.Steps) != 1 || got.Steps[0].Title != "primeira" || !got.Steps[0].Done {
		t.Errorf("task = %+v", got)
	}
	if !strings.Contains(s.info(got), "1/1") {
		t.Errorf("info = %q", s.info(got))
	}
}
