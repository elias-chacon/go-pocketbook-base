package sqlstore

import (
	"strings"
	"testing"
	"time"

	"go-pocketbook-base/internal/storage"
	"go-pocketbook-base/internal/tasks/domain"
)

func day(m time.Month, d int) time.Time { return time.Date(2026, m, d, 0, 0, 0, 0, time.UTC) }

func newStore(t *testing.T) (*Store, *storage.Store) {
	t.Helper()
	db, err := storage.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	s, err := New(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveLists("ana", []domain.List{{ID: "work", Name: "Trabalho"}, {ID: "home", Name: "Casa"}}); err != nil {
		t.Fatal(err)
	}
	return s, db
}

func titles(tasks []domain.Task) string {
	var s []string
	for _, t := range tasks {
		s = append(s, t.Title)
	}
	return strings.Join(s, "|")
}

func TestATaskRoundTripsWithEveryField(t *testing.T) {
	s, _ := newStore(t)
	in := domain.Task{
		ID: "t1", ListID: "work", RemoteID: "r1", Title: "Relatório", Notes: "Entregar até sexta", Done: true, DoneAt: day(3, 6),
		Important: true, Due: day(3, 13), MyDay: day(3, 10), Reminder: time.Date(2026, 3, 12, 9, 0, 0, 0, time.UTC),
		Recurrence: "FREQ=WEEKLY;BYDAY=FR", MailID: "ana:inbox:42", CreatedAt: day(3, 1),
		Steps: []domain.Step{{Title: "rascunho", Done: true}, {Title: "revisão"}},
	}
	if err := s.Save(in); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("t1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != in.Title || got.Notes != in.Notes || got.RemoteID != "r1" || got.ListID != "work" || got.MailID != in.MailID || got.Recurrence != in.Recurrence {
		t.Errorf("%+v", got)
	}
	if !got.Done || !got.Important || !got.DoneAt.Equal(in.DoneAt) || !got.Due.Equal(in.Due) || !got.MyDay.Equal(in.MyDay) ||
		!got.Reminder.Equal(in.Reminder) || !got.CreatedAt.Equal(in.CreatedAt) {
		t.Errorf("%+v", got)
	}
	if len(got.Steps) != 2 || !got.Steps[0].Done || got.Steps[1].Title != "revisão" || got.StepsDone() != 1 {
		t.Errorf("steps: %+v", got.Steps)
	}
	if _, err := s.Get("ghost"); err != domain.ErrNotFound {
		t.Errorf("%v", err)
	}
}

func TestADateThatIsNotSetComesBackNotSet(t *testing.T) {
	s, _ := newStore(t)
	s.Save(domain.Task{ID: "a", ListID: "work", Title: "Simples"})
	got, _ := s.Get("a")
	if !got.Due.IsZero() || !got.MyDay.IsZero() || !got.Reminder.IsZero() || !got.DoneAt.IsZero() || len(got.Steps) != 0 {
		t.Errorf("%+v", got)
	}
}

func TestSavingAssignsAnIDReplacesAndRefusesAnUnknownList(t *testing.T) {
	s, _ := newStore(t)
	if err := s.Save(domain.Task{ListID: "work", Title: "Sem id", CreatedAt: day(3, 1)}); err != nil {
		t.Fatal(err)
	}
	all, _ := s.Tasks(domain.TaskQuery{AccountID: "ana"})
	if len(all) != 1 || len(all[0].ID) != 16 {
		t.Fatalf("%+v", all)
	}
	changed := all[0]
	changed.Title = "Com id"
	s.Save(changed)
	if all, _ := s.Tasks(domain.TaskQuery{AccountID: "ana"}); len(all) != 1 || all[0].Title != "Com id" {
		t.Errorf("replaced, not duplicated: %s", titles(all))
	}
	if err := s.Save(domain.Task{ListID: "nowhere", Title: "X"}); err == nil || !strings.Contains(err.Error(), "nowhere") {
		t.Errorf("%v", err)
	}
}

func TestTasksAreListedNewestFirstOpenOnesByDefault(t *testing.T) {
	s, _ := newStore(t)
	s.Save(
		domain.Task{ID: "1", ListID: "work", Title: "Antiga", CreatedAt: day(3, 1)},
		domain.Task{ID: "2", ListID: "work", Title: "Nova", CreatedAt: day(3, 3)},
		domain.Task{ID: "3", ListID: "home", Title: "De casa", CreatedAt: day(3, 2)},
		domain.Task{ID: "4", ListID: "work", Title: "Feita", CreatedAt: day(3, 4), Done: true},
	)
	if got, _ := s.Tasks(domain.TaskQuery{AccountID: "ana"}); titles(got) != "Nova|De casa|Antiga" {
		t.Errorf("%s", titles(got))
	}
	if got, _ := s.Tasks(domain.TaskQuery{AccountID: "ana", ListID: "work"}); titles(got) != "Nova|Antiga" {
		t.Errorf("one list: %s", titles(got))
	}
	if got, _ := s.Tasks(domain.TaskQuery{AccountID: "ana", IncludeDone: true}); titles(got) != "Feita|Nova|De casa|Antiga" {
		t.Errorf("with the done ones: %s", titles(got))
	}
	if got, _ := s.Tasks(domain.TaskQuery{AccountID: "bia"}); len(got) != 0 {
		t.Error("another account's tasks")
	}
}

func TestCompletingARepeatingTaskThroughTheStoreLeavesTheNextOne(t *testing.T) {
	s, _ := newStore(t)
	weekly := domain.Task{ID: "w", ListID: "work", Title: "Relatório", Due: day(3, 6), Recurrence: "FREQ=WEEKLY;BYDAY=FR", Steps: []domain.Step{{Title: "a", Done: true}}, CreatedAt: day(3, 1)}
	s.Save(weekly)

	loaded, _ := s.Get("w")
	done, next, err := loaded.Complete(day(3, 6))
	if err != nil || next == nil {
		t.Fatalf("%v %v", next, err)
	}
	if err := s.Save(done, *next); err != nil {
		t.Fatal(err)
	}
	open, _ := s.Tasks(domain.TaskQuery{AccountID: "ana"})
	if len(open) != 1 || !open[0].Due.Equal(day(3, 13)) || open[0].ID == "w" || open[0].Steps[0].Done {
		t.Errorf("only the next one is open, due on the 13th, with fresh steps: %+v", open)
	}
	all, _ := s.Tasks(domain.TaskQuery{AccountID: "ana", IncludeDone: true})
	if len(all) != 2 {
		t.Errorf("and the done one is kept: %d", len(all))
	}
	// the views work over what the store returns
	if got := domain.Filter(open, domain.ViewPlanned, day(3, 7)); len(got) != 1 {
		t.Error("Planned")
	}
}

func TestListsFollowTheServerAndTakeTheirTasksWithThem(t *testing.T) {
	s, db := newStore(t)
	s.Save(domain.Task{ID: "1", ListID: "work", Title: "A"}, domain.Task{ID: "2", ListID: "home", Title: "B"})
	lists, _ := s.Lists("ana")
	if len(lists) != 2 || lists[0].Name != "Casa" {
		t.Errorf("by name: %+v", lists)
	}
	if err := s.SaveLists("ana", []domain.List{{ID: "work", Name: "Trabalho 2"}}); err != nil {
		t.Fatal(err)
	}
	lists, _ = s.Lists("ana")
	if len(lists) != 1 || lists[0].Name != "Trabalho 2" {
		t.Errorf("%+v", lists)
	}
	if got, _ := s.Tasks(domain.TaskQuery{AccountID: "ana"}); titles(got) != "A" {
		t.Errorf("the other list took its tasks: %s", titles(got))
	}
	if err := db.Erase("ana"); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Tasks(domain.TaskQuery{AccountID: "ana", IncludeDone: true}); len(got) != 0 {
		t.Error("erasing the account clears its tasks")
	}
	if lists, _ := s.Lists("ana"); len(lists) != 0 {
		t.Error("and its lists")
	}
	if err := s.SaveLists("ana", nil); err != nil {
		t.Errorf("%v", err)
	}
}

func TestDeleteAndFailures(t *testing.T) {
	s, db := newStore(t)
	s.Save(domain.Task{ID: "1", ListID: "work", Title: "A"})
	if err := s.Delete("1"); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("1"); err != domain.ErrNotFound {
		t.Errorf("%v", err)
	}

	s.Save(domain.Task{ID: "2", ListID: "work", Title: "B"})
	db.DB().Exec(`UPDATE tasks_tasks SET steps = 'not json'`)
	if _, err := s.Get("2"); err == nil || !strings.Contains(err.Error(), "damaged") {
		t.Errorf("%v", err)
	}
	if _, err := s.Tasks(domain.TaskQuery{AccountID: "ana"}); err == nil {
		t.Error("a list with a damaged row is an error")
	}

	db.Close()
	if err := s.Save(domain.Task{ListID: "work", Title: "C"}); err == nil {
		t.Error("Save")
	}
	if err := s.SaveLists("ana", nil); err == nil {
		t.Error("SaveLists")
	}
	if _, err := s.Lists("ana"); err == nil {
		t.Error("Lists")
	}
	if _, err := s.Tasks(domain.TaskQuery{}); err == nil {
		t.Error("Tasks")
	}
	if err := s.Delete("2"); err == nil || err == domain.ErrNotFound {
		t.Errorf("Delete: %v", err)
	}
}
