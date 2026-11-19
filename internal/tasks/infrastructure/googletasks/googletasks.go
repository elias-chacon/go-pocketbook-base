// Package googletasks is the domain.Backend for Google Tasks over its REST API. Google Tasks
// has no importance, reminder or repetition: they stay on the reader. Its due date is a day
// (the time of day is ignored by Google).
package googletasks

import (
	"errors"
	"net/http"
	"net/url"
	"time"

	"go-pocketbook-base/internal/kernel/webjson"
	"go-pocketbook-base/internal/tasks/domain"
)

// DefaultBase is the address of the Google Tasks API.
const DefaultBase = "https://tasks.googleapis.com/tasks/v1"

// listsPath is the address of the lists of tasks.
const listsPath = "/lists/"

// pageSize is how many items one request asks for (the most Google allows).
const pageSize = "100"

// Client implements domain.Backend over Google Tasks.
type Client struct {
	HTTP  webjson.Doer
	Token func() (string, error)
	Base  string // DefaultBase when empty
}

var _ domain.Backend = Client{}

func (c Client) web() webjson.Client {
	base := c.Base
	if base == "" {
		base = DefaultBase
	}
	return webjson.Client{HTTP: c.HTTP, Token: c.Token, Base: base}
}

func notFound(err error) error {
	var se *webjson.StatusError
	if errors.As(err, &se) && se.Code == http.StatusNotFound {
		return domain.ErrNotFound
	}
	return err
}

// Lists implements domain.Backend.
func (c Client) Lists() ([]domain.RemoteList, error) {
	var out []domain.RemoteList
	token := ""
	for {
		var page struct {
			Items []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"items"`
			Next string `json:"nextPageToken"`
		}
		q := url.Values{"maxResults": {pageSize}}
		if token != "" {
			q.Set("pageToken", token)
		}
		if err := c.web().Do(http.MethodGet, "/users/@me/lists?"+q.Encode(), nil, &page); err != nil {
			return nil, err
		}
		for _, l := range page.Items {
			out = append(out, domain.RemoteList{ID: l.ID, Name: l.Title})
		}
		if page.Next == "" {
			return out, nil
		}
		token = page.Next
	}
}

type remoteTask struct {
	ID        string `json:"id,omitempty"`
	Title     string `json:"title"`
	Notes     string `json:"notes"`
	Status    string `json:"status"`
	Due       string `json:"due,omitempty"`
	Completed string `json:"completed,omitempty"`
	Deleted   bool   `json:"deleted,omitempty"`
	Updated   string `json:"updated,omitempty"`
}

// Tasks implements domain.Backend: open and completed ones.
func (c Client) Tasks(listID string) ([]domain.Task, error) {
	var out []domain.Task
	token := ""
	for {
		var page struct {
			Items []remoteTask `json:"items"`
			Next  string       `json:"nextPageToken"`
		}
		q := url.Values{"maxResults": {pageSize}, "showCompleted": {"true"}, "showHidden": {"true"}}
		if token != "" {
			q.Set("pageToken", token)
		}
		if err := c.web().Do(http.MethodGet, listsPath+url.PathEscape(listID)+"/tasks?"+q.Encode(), nil, &page); err != nil {
			return nil, notFound(err)
		}
		for _, r := range page.Items {
			if !r.Deleted {
				out = append(out, taskOf(r))
			}
		}
		if page.Next == "" {
			return out, nil
		}
		token = page.Next
	}
}

func parse(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func taskOf(r remoteTask) domain.Task {
	t := domain.Task{RemoteID: r.ID, Title: r.Title, Notes: r.Notes, Done: r.Status == "completed"}
	if d := parse(r.Due); !d.IsZero() {
		t.Due = domain.Day(d)
	}
	if t.Done {
		t.DoneAt = parse(r.Completed)
	}
	return t
}

// payload is what is sent: Google ignores the time of the due date, so it is sent at midnight UTC.
func payload(t domain.Task) map[string]interface{} {
	p := map[string]interface{}{"title": t.Title, "notes": t.Notes, "status": "needsAction"}
	if t.Done {
		p["status"] = "completed"
	}
	if t.Due.IsZero() {
		p["due"] = nil
	} else {
		p["due"] = domain.Day(t.Due).Format("2006-01-02T15:04:05.000Z")
	}
	if !t.Done {
		p["completed"] = nil // reopening a task must clear its completion
	}
	return p
}

func taskPath(listID, remoteID string) string {
	return listsPath + url.PathEscape(listID) + "/tasks/" + url.PathEscape(remoteID)
}

// Create implements domain.Backend.
func (c Client) Create(listID string, t domain.Task) (string, error) {
	var made remoteTask
	if err := c.web().Do(http.MethodPost, listsPath+url.PathEscape(listID)+"/tasks", payload(t), &made); err != nil {
		return "", err
	}
	if made.ID == "" {
		return "", errors.New("googletasks: the service gave the task no id")
	}
	return made.ID, nil
}

// Update implements domain.Backend.
func (c Client) Update(listID string, t domain.Task) error {
	return notFound(c.web().Do(http.MethodPatch, taskPath(listID, t.RemoteID), payload(t), nil))
}

// Delete implements domain.Backend.
func (c Client) Delete(listID, remoteID string) error {
	return notFound(c.web().Do(http.MethodDelete, taskPath(listID, remoteID), nil, nil))
}
