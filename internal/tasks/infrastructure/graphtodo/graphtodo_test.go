package graphtodo

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-pocketbook-base/internal/tasks/domain"
)

type call struct{ method, path, body string }

func server(t *testing.T, calls *[]call, handler func(w http.ResponseWriter, r *http.Request)) Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*calls = append(*calls, call{r.Method, r.URL.RequestURI(), string(b)})
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	return Client{HTTP: srv.Client(), Token: func() (string, error) { return "tok", nil }, Base: srv.URL}
}

func TestListsFollowsThePages(t *testing.T) {
	var calls []call
	var base string
	c := server(t, &calls, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("p") == "2" {
			_, _ = w.Write([]byte(`{"value":[{"id":"L2","displayName":"Work"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"value":[{"id":"L1","displayName":"Tasks"}],"@odata.nextLink":"` + base + `/me/todo/lists?p=2"}`))
	})
	base = c.Base
	got, err := c.Lists()
	if err != nil || len(got) != 2 || got[1].ID != "L2" || got[0].Name != "Tasks" {
		t.Fatalf("lists = %+v, %v", got, err)
	}
}

func TestTasksAreReadWithTheirFields(t *testing.T) {
	var calls []call
	c := server(t, &calls, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"value":[
		 {"id":"T1","title":"Pay","importance":"high","status":"completed","body":{"content":"notes"},
		  "dueDateTime":{"dateTime":"2026-05-20T00:00:00.0000000","timeZone":"UTC"},
		  "completedDateTime":{"dateTime":"2026-05-19T10:00:00.0000000","timeZone":"UTC"},
		  "reminderDateTime":{"dateTime":"2026-05-20T09:00:00.0000000","timeZone":"UTC"},
		  "createdDateTime":"2026-05-01T08:00:00Z"},
		 {"id":"T2","title":"Open","importance":"normal","status":"notStarted"}]}`))
	})
	got, err := c.Tasks("L 1")
	if err != nil || len(got) != 2 {
		t.Fatalf("tasks = %+v, %v", got, err)
	}
	a := got[0]
	if a.RemoteID != "T1" || !a.Done || !a.Important || a.Notes != "notes" || a.Due != time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC) ||
		a.DoneAt.IsZero() || a.Reminder.IsZero() || a.CreatedAt.IsZero() {
		t.Fatalf("task = %+v", a)
	}
	if got[1].Done || got[1].Important || !got[1].Due.IsZero() {
		t.Fatalf("open task = %+v", got[1])
	}
	if !strings.Contains(calls[0].path, "/me/todo/lists/L%201/tasks") {
		t.Fatalf("path = %s", calls[0].path)
	}
}

func TestCreateUpdateAndDeleteSendTheTask(t *testing.T) {
	var calls []call
	c := server(t, &calls, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			_, _ = w.Write([]byte(`{"id":"NEW"}`))
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	})
	task := domain.Task{Title: "Pay", Notes: "n", Important: true, Due: time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)}
	id, err := c.Create("L1", task)
	if err != nil || id != "NEW" {
		t.Fatalf("create = %q, %v", id, err)
	}
	var sent map[string]interface{}
	if err := json.Unmarshal([]byte(calls[0].body), &sent); err != nil || sent["importance"] != "high" || sent["status"] != "notStarted" || sent["dueDateTime"] == nil {
		t.Fatalf("sent = %v, %v", sent, err)
	}
	task.RemoteID, task.Done, task.Due = "NEW", true, time.Time{}
	if err := c.Update("L1", task); err != nil || calls[1].method != http.MethodPatch || !strings.Contains(calls[1].body, `"dueDateTime":null`) || !strings.Contains(calls[1].body, `"completed"`) {
		t.Fatalf("update = %v %+v", err, calls[1])
	}
	if err := c.Delete("L1", "NEW"); err != nil || calls[2].method != http.MethodDelete {
		t.Fatalf("delete = %v %+v", err, calls[2])
	}
}

func TestAnUnknownTaskIsNotFoundAndAnErrorKeepsItsWords(t *testing.T) {
	var calls []call
	c := server(t, &calls, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "gone") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"no access"}}`))
	})
	if err := c.Delete("L", "gone"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
	if err := c.Update("L", domain.Task{RemoteID: "x"}); err == nil || !strings.Contains(err.Error(), "no access") {
		t.Fatalf("err = %v", err)
	}
	if _, err := c.Tasks("gone"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
	c.Token = func() (string, error) { return "", errors.New("no token") }
	if _, err := c.Lists(); err == nil {
		t.Fatal("a missing token must fail")
	}
}

func TestCreateWithoutAnIdFails(t *testing.T) {
	var calls []call
	c := server(t, &calls, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{}`)) })
	if _, err := c.Create("L", domain.Task{Title: "x"}); err == nil {
		t.Fatal("a task with no id must fail")
	}
}
