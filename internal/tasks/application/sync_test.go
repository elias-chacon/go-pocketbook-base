package application

import (
	"errors"
	"testing"
	"time"

	"go-pocketbook-base/internal/tasks/domain"
)

// fakeBackend is a service in memory.
type fakeBackend struct {
	lists    []domain.RemoteList
	tasks    map[string][]domain.Task // by remote list id
	created  []domain.Task
	updated  []domain.Task
	deleted  []string
	listsErr error
	tasksErr error
	sendErr  error
	delErr   error
}

func (f *fakeBackend) Lists() ([]domain.RemoteList, error) { return f.lists, f.listsErr }
func (f *fakeBackend) Tasks(id string) ([]domain.Task, error) {
	return f.tasks[id], f.tasksErr
}
func (f *fakeBackend) Create(_ string, t domain.Task) (string, error) {
	if f.sendErr != nil {
		return "", f.sendErr
	}
	f.created = append(f.created, t)
	return "R" + t.Title, nil
}
func (f *fakeBackend) Update(_ string, t domain.Task) error {
	if f.sendErr != nil {
		return f.sendErr
	}
	f.updated = append(f.updated, t)
	return nil
}
func (f *fakeBackend) Delete(_, id string) error {
	f.deleted = append(f.deleted, id)
	return f.delErr
}

type fakeLog struct {
	changes []domain.Change
	err     error
}

func (l *fakeLog) Pending() ([]domain.Change, error) { return l.changes, l.err }
func (l *fakeLog) Resolve(id string) error {
	var keep []domain.Change
	for _, c := range l.changes {
		if c.TaskID != id {
			keep = append(keep, c)
		}
	}
	l.changes = keep
	return nil
}

func syncFixture() (Sync, *memStore, *fakeBackend, *fakeLog) {
	st, be, lg := newMem(), &fakeBackend{tasks: map[string][]domain.Task{}}, &fakeLog{}
	st.lists = []domain.List{{ID: "mine", Name: "Local"}}
	return Sync{AccountID: "local", Store: st, Log: lg, Backend: be}, st, be, lg
}

func TestSyncBringsTheListsAndTasksOfTheService(t *testing.T) {
	s, st, be, _ := syncFixture()
	be.lists = []domain.RemoteList{{ID: "L1", Name: "Tasks"}}
	due := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)
	be.tasks["L1"] = []domain.Task{{RemoteID: "T1", Title: "Pay", Due: due, Important: true}}
	res, err := s.Run()
	if err != nil || res.Received != 1 || len(res.Errors) != 0 {
		t.Fatalf("res = %+v, %v", res, err)
	}
	if len(st.lists) != 2 || st.lists[0].ID != "mine" || st.lists[1].ID != "r:L1" {
		t.Fatalf("lists = %+v", st.lists)
	}
	got := st.tasks["r:T1"]
	if got.ListID != "r:L1" || got.Title != "Pay" || !got.Important || got.Due != due || got.RemoteID != "T1" {
		t.Fatalf("task = %+v", got)
	}
}

func TestSyncKeepsWhatTheServiceHasNoFieldFor(t *testing.T) {
	s, st, be, _ := syncFixture()
	be.lists = []domain.RemoteList{{ID: "L1", Name: "Tasks"}}
	st.tasks["r:T1"] = domain.Task{ID: "r:T1", ListID: "r:L1", RemoteID: "T1", Title: "Old", Recurrence: "FREQ=DAILY",
		Steps: []domain.Step{{Title: "a"}}, MailID: "m1", MyDay: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)}
	be.tasks["L1"] = []domain.Task{{RemoteID: "T1", Title: "New", Done: true}}
	if _, err := s.Run(); err != nil {
		t.Fatal(err)
	}
	got := st.tasks["r:T1"]
	if got.Title != "New" || !got.Done || got.Recurrence != "FREQ=DAILY" || len(got.Steps) != 1 || got.MailID != "m1" || got.MyDay.IsZero() {
		t.Fatalf("task = %+v", got)
	}
}

func TestSyncSendsLocalChangesFirstAndTheyWinOverTheService(t *testing.T) {
	s, st, be, lg := syncFixture()
	be.lists = []domain.RemoteList{{ID: "L1", Name: "Tasks"}}
	st.tasks["new"] = domain.Task{ID: "new", ListID: "r:L1", Title: "Buy"}
	st.tasks["r:T1"] = domain.Task{ID: "r:T1", ListID: "r:L1", RemoteID: "T1", Title: "Edited here"}
	lg.changes = []domain.Change{{TaskID: "new", ListID: "r:L1"}, {TaskID: "r:T1", ListID: "r:L1", RemoteID: "T1"}, {TaskID: "r:T2", ListID: "r:L1", RemoteID: "T2", Deleted: true}, {TaskID: "ghost", ListID: "r:L1"}}
	be.tasks["L1"] = []domain.Task{{RemoteID: "T1", Title: "Edited there"}, {RemoteID: "RBuy", Title: "Buy"}}
	res, err := s.Run()
	if err != nil || len(res.Errors) != 0 || res.Sent != 4 {
		t.Fatalf("res = %+v, %v", res, err)
	}
	if len(be.created) != 1 || len(be.updated) != 1 || len(be.deleted) != 1 || be.deleted[0] != "T2" {
		t.Fatalf("created %d updated %d deleted %v", len(be.created), len(be.updated), be.deleted)
	}
	if st.tasks["new"].RemoteID != "RBuy" {
		t.Fatalf("the new task did not keep the id of the service: %+v", st.tasks["new"])
	}
	if be.updated[0].Title != "Edited here" {
		t.Fatalf("the local change was not what was sent: %+v", be.updated[0])
	}
	if len(lg.changes) != 0 {
		t.Fatalf("changes left: %+v", lg.changes)
	}
}

func TestSyncRemovesWhatWasDeletedAtTheServiceButNotWhatIsWaiting(t *testing.T) {
	s, st, be, lg := syncFixture()
	be.lists = []domain.RemoteList{{ID: "L1", Name: "Tasks"}}
	st.tasks["r:A"] = domain.Task{ID: "r:A", ListID: "r:L1", RemoteID: "A", Title: "gone at the service"}
	st.tasks["r:B"] = domain.Task{ID: "r:B", ListID: "r:L1", RemoteID: "B", Title: "edited, waiting"}
	st.tasks["only-here"] = domain.Task{ID: "only-here", ListID: "mine", Title: "local list"}
	lg.changes = []domain.Change{{TaskID: "r:B", ListID: "r:L1", RemoteID: "B"}}
	be.sendErr = errors.New("offline") // B could not be sent: it stays
	res, err := s.Run()
	if err != nil || res.Removed != 1 || len(res.Errors) != 1 {
		t.Fatalf("res = %+v, %v", res, err)
	}
	if _, ok := st.tasks["r:A"]; ok {
		t.Fatal("a task deleted at the service stayed")
	}
	if _, ok := st.tasks["r:B"]; !ok {
		t.Fatal("a task with a change waiting was removed")
	}
	if _, ok := st.tasks["only-here"]; !ok {
		t.Fatal("a task of a local list was touched")
	}
}

func TestSyncKeepsTheTasksOfAListItCouldNotRead(t *testing.T) {
	s, st, be, _ := syncFixture()
	be.lists = []domain.RemoteList{{ID: "L1", Name: "Tasks"}}
	st.tasks["r:A"] = domain.Task{ID: "r:A", ListID: "r:L1", RemoteID: "A", Title: "kept"}
	be.tasksErr = errors.New("timeout")
	res, err := s.Run()
	if err != nil || len(res.Errors) != 1 {
		t.Fatalf("res = %+v, %v", res, err)
	}
	if _, ok := st.tasks["r:A"]; !ok {
		t.Fatal("an unreadable list was emptied")
	}
}

func TestSyncStopsWhenTheServiceCannotBeReached(t *testing.T) {
	s, _, be, lg := syncFixture()
	be.listsErr = errors.New("401")
	if _, err := s.Run(); err == nil {
		t.Fatal("the lists failing must stop the run")
	}
	be.listsErr, lg.err = nil, errors.New("disk")
	if _, err := s.Run(); err == nil {
		t.Fatal("the log failing must stop the run")
	}
}

func TestSyncIgnoresADeleteTheServiceAlreadyDid(t *testing.T) {
	s, _, be, lg := syncFixture()
	be.delErr = domain.ErrNotFound
	lg.changes = []domain.Change{{TaskID: "x", ListID: "r:L1", RemoteID: "X", Deleted: true}}
	res, _ := s.Run()
	if res.Sent != 1 || len(lg.changes) != 0 {
		t.Fatalf("res = %+v changes %+v", res, lg.changes)
	}
}
