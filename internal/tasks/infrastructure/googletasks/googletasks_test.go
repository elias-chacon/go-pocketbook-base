package googletasks

import (
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

func server(t *testing.T, calls *[]call, handler http.HandlerFunc) Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*calls = append(*calls, call{r.Method, r.URL.RequestURI(), string(b)})
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	return Client{HTTP: srv.Client(), Token: func() (string, error) { return "tok", nil }, Base: srv.URL}
}

func TestListsAndTasksFollowThePageTokens(t *testing.T) {
	var calls []call
	c := server(t, &calls, func(w http.ResponseWriter, r *http.Request) {
		second := r.URL.Query().Get("pageToken") == "p2"
		switch {
		case strings.HasSuffix(r.URL.Path, "/lists") && !second:
			_, _ = w.Write([]byte(`{"items":[{"id":"A","title":"My"}],"nextPageToken":"p2"}`))
		case strings.HasSuffix(r.URL.Path, "/lists"):
			_, _ = w.Write([]byte(`{"items":[{"id":"B","title":"Work"}]}`))
		case !second:
			_, _ = w.Write([]byte(`{"items":[{"id":"1","title":"Pay","notes":"n","status":"completed","due":"2026-05-20T00:00:00.000Z","completed":"2026-05-19T10:00:00.000Z"}],"nextPageToken":"p2"}`))
		default:
			_, _ = w.Write([]byte(`{"items":[{"id":"2","title":"Gone","deleted":true},{"id":"3","title":"Open","status":"needsAction"}]}`))
		}
	})
	lists, err := c.Lists()
	if err != nil || len(lists) != 2 || lists[1].Name != "Work" {
		t.Fatalf("lists = %+v, %v", lists, err)
	}
	tasks, err := c.Tasks("A")
	if err != nil || len(tasks) != 2 {
		t.Fatalf("tasks = %+v, %v", tasks, err)
	}
	if a := tasks[0]; !a.Done || a.Notes != "n" || a.Due != time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC) || a.DoneAt.IsZero() {
		t.Fatalf("task = %+v", a)
	}
	if tasks[1].RemoteID != "3" || tasks[1].Done {
		t.Fatalf("open = %+v", tasks[1])
	}
}

func TestCreateUpdateDeleteAndNotFound(t *testing.T) {
	var calls []call
	c := server(t, &calls, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"id":"NEW"}`))
		case strings.Contains(r.URL.Path, "gone"):
			w.WriteHeader(http.StatusNotFound)
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	})
	task := domain.Task{Title: "Pay", Due: time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)}
	if id, err := c.Create("L", task); err != nil || id != "NEW" || !strings.Contains(calls[0].body, `"due":"2026-05-20T00:00:00.000Z"`) {
		t.Fatalf("create = %q %v %s", id, err, calls[0].body)
	}
	task.RemoteID, task.Due = "NEW", time.Time{}
	if err := c.Update("L", task); err != nil || !strings.Contains(calls[1].body, `"due":null`) || !strings.Contains(calls[1].body, `"completed":null`) {
		t.Fatalf("update = %v %s", err, calls[1].body)
	}
	task.Done = true
	if err := c.Update("L", task); err != nil || !strings.Contains(calls[2].body, `"status":"completed"`) {
		t.Fatalf("update done = %v %s", err, calls[2].body)
	}
	if err := c.Delete("L", "NEW"); err != nil {
		t.Fatal(err)
	}
	if err := c.Delete("L", "gone"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
	if _, err := c.Tasks("gone"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestCreateWithoutAnIdFailsAndAServerErrorIsReturned(t *testing.T) {
	var calls []call
	c := server(t, &calls, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write([]byte(`{}`))
	})
	if _, err := c.Create("L", domain.Task{Title: "x"}); err == nil {
		t.Fatal("no id must fail")
	}
	if _, err := c.Lists(); err == nil {
		t.Fatal("a 500 must fail")
	}
	c.Base = ""
	if c.web().Base != DefaultBase {
		t.Fatal("default base")
	}
}
