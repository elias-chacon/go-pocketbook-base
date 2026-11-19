package application

import (
	"errors"
	"sort"
	"strconv"
	"testing"
	"time"

	"go-pocketbook-base/internal/tasks/domain"
)

// memStore is a TaskStore in memory.
type memStore struct {
	lists []domain.List
	tasks map[string]domain.Task
	next  int
	err   error
}

func newMem() *memStore { return &memStore{tasks: map[string]domain.Task{}} }

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
func (m *memStore) Delete(id string) error { delete(m.tasks, id); return m.err }

var now = time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC)

func todo() (Checklist, *memStore) {
	st := newMem()
	return Checklist{AccountID: "local", Store: st, Now: func() time.Time { return now }}, st
}

func TestAddPutsTheTaskInTheViewItWasAddedIn(t *testing.T) {
	u, st := todo()
	cases := []struct {
		view  domain.View
		check func(domain.Task) bool
	}{
		{domain.ViewMyDay, func(t domain.Task) bool { return domain.Day(t.MyDay).Equal(domain.Day(now)) }},
		{domain.ViewImportant, func(t domain.Task) bool { return t.Important }},
		{domain.ViewPlanned, func(t domain.Task) bool { return domain.Day(t.Due).Equal(domain.Day(now)) }},
		{domain.ViewAll, func(t domain.Task) bool { return t.MyDay.IsZero() && !t.Important && t.Due.IsZero() }},
	}
	for _, c := range cases {
		if _, err := u.Add("  Comprar pão ", c.view); err != nil {
			t.Fatal(err)
		}
	}
	if len(st.lists) != 1 || st.lists[0].ID != DefaultListID {
		t.Errorf("the default list was not created: %+v", st.lists)
	}
	all, _ := u.All()
	for i, c := range cases {
		if all[i].Title != "Comprar pão" || !c.check(all[i]) {
			t.Errorf("%s: %+v", c.view, all[i])
		}
	}
	if _, err := u.Add("   ", domain.ViewAll); !errors.Is(err, ErrEmptyTitle) {
		t.Errorf("err = %v", err)
	}
}

func TestViewsAndCountsFollowTheClock(t *testing.T) {
	u, _ := todo()
	u.Add("a", domain.ViewMyDay)
	u.Add("b", domain.ViewImportant)
	u.Add("c", domain.ViewPlanned)
	u.SetDue("t3", now.AddDate(0, 0, -2))
	list, counts, err := u.View(domain.ViewOverdue)
	if err != nil || len(list) != 1 || list[0].Title != "c" {
		t.Fatalf("overdue = %+v %v", list, err)
	}
	if counts.MyDay != 1 || counts.Important != 1 || counts.Planned != 1 || counts.Overdue != 1 {
		t.Errorf("counts = %+v", counts)
	}
}

func TestToggleTicksReopensAndRepeats(t *testing.T) {
	u, st := todo()
	u.Add("simples", domain.ViewAll)
	if err := u.Toggle("t1"); err != nil || !st.tasks["t1"].Done {
		t.Fatalf("not done: %v", err)
	}
	if err := u.Toggle("t1"); err != nil || st.tasks["t1"].Done {
		t.Fatalf("not reopened: %v", err)
	}
	u.Add("semanal", domain.ViewPlanned)
	u.Change("t2", func(x *domain.Task) { x.Recurrence = "FREQ=WEEKLY" })
	if err := u.Toggle("t2"); err != nil {
		t.Fatal(err)
	}
	if len(st.tasks) != 3 {
		t.Fatalf("the next occurrence was not created: %d tasks", len(st.tasks))
	}
	if next := st.tasks["t3"]; next.Done || !next.Due.After(domain.Day(now)) {
		t.Errorf("next = %+v", next)
	}
	if err := u.Toggle("nope"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("err = %v", err)
	}
}

func TestStarMyDayDueSaveAndDelete(t *testing.T) {
	u, st := todo()
	u.Add("x", domain.ViewAll)
	u.SetImportant("t1", true)
	u.SetMyDay("t1", true)
	u.SetDue("t1", now.AddDate(0, 0, 3))
	got := st.tasks["t1"]
	if !got.Important || got.MyDay.IsZero() || got.Due.IsZero() {
		t.Fatalf("task = %+v", got)
	}
	u.SetMyDay("t1", false)
	u.SetDue("t1", time.Time{})
	if got = st.tasks["t1"]; !got.MyDay.IsZero() || !got.Due.IsZero() {
		t.Errorf("not cleared: %+v", got)
	}
	got.Title = "  "
	if err := u.Save(got); err == nil {
		t.Error("a task with no title was saved")
	}
	if err := u.Change("t1", func(x *domain.Task) { x.Title = "" }); err == nil {
		t.Error("an invalid change was kept")
	}
	if g, err := u.Get("t1"); err != nil || g.Title != "x" {
		t.Errorf("get = %+v %v", g, err)
	}
	if err := u.Delete("t1"); err != nil || len(st.tasks) != 0 {
		t.Errorf("delete: %v %d", err, len(st.tasks))
	}
}

func TestStoreErrorsArePassedOn(t *testing.T) {
	u, st := todo()
	st.err = errors.New("disco")
	if _, err := u.Add("x", domain.ViewAll); err == nil {
		t.Error("add ignored the store error")
	}
	if _, _, err := u.View(domain.ViewAll); err == nil {
		t.Error("view ignored the store error")
	}
	if _, err := (Checklist{AccountID: "a", Store: st}).EnsureList("x"); err == nil {
		t.Error("ensure ignored the store error")
	}
	if (Checklist{}).now().IsZero() {
		t.Error("the default clock gave zero")
	}
}
