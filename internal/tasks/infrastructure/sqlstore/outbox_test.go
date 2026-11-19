package sqlstore

import (
	"testing"

	"go-pocketbook-base/internal/tasks/domain"
)

func TestChangesToRemoteTasksAreKeptUntilResolved(t *testing.T) {
	s, _ := newStore(t)
	if err := s.SaveLists("ana", []domain.List{{ID: "r:L1", Name: "Sync"}, {ID: "work", Name: "Local"}}); err != nil {
		t.Fatal(err)
	}
	local := domain.Task{ID: "l1", ListID: "work", Title: "only here"}
	remote := domain.Task{ID: "r:a", ListID: "r:L1", RemoteID: "a", Title: "synced"}
	fresh := domain.Task{ID: "n1", ListID: "r:L1", Title: "new"}
	if err := s.Save(local, remote, fresh); err != nil {
		t.Fatal(err)
	}
	got, err := s.Pending()
	if err != nil || len(got) != 2 || got[0].TaskID != "r:a" || got[1].TaskID != "n1" {
		t.Fatalf("pending = %+v, %v", got, err)
	}
	for _, id := range []string{"r:a", "n1", "l1"} { // n1 was never sent: nothing to tell
		if err := s.Delete(id); err != nil {
			t.Fatal(err)
		}
	}
	got, _ = s.Pending()
	if len(got) != 1 || !got[0].Deleted || got[0].RemoteID != "a" || got[0].ListID != "r:L1" {
		t.Fatalf("pending after deletes = %+v", got)
	}
	if err := s.Resolve("r:a"); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Pending(); len(got) != 0 {
		t.Fatalf("pending = %+v", got)
	}
	if err := s.Delete("missing"); err == nil {
		t.Fatal("deleting a missing task must fail")
	}
}

func TestTheRawStoreDoesNotRecordChanges(t *testing.T) {
	s, _ := newStore(t)
	if err := s.SaveLists("ana", []domain.List{{ID: "r:L1", Name: "Sync"}}); err != nil {
		t.Fatal(err)
	}
	raw := s.Raw()
	if err := raw.Save(domain.Task{ID: "r:a", ListID: "r:L1", RemoteID: "a", Title: "from the service"}); err != nil {
		t.Fatal(err)
	}
	if err := raw.Delete("r:a"); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Pending(); len(got) != 0 {
		t.Fatalf("the writes of the service became changes: %+v", got)
	}
}
