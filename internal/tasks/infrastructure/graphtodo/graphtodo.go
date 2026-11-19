// Package graphtodo is the domain.Backend for Microsoft To Do (Outlook.com, Microsoft 365)
// over Microsoft Graph. The steps (checklist items) and the repetition stay on the reader:
// they have no field in the task that Sync could trust on both sides.
package graphtodo

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go-pocketbook-base/internal/kernel/webjson"
	"go-pocketbook-base/internal/tasks/domain"
)

// DefaultBase is Microsoft Graph's address.
const DefaultBase = "https://graph.microsoft.com/v1.0"

// listsPath is the address of the lists of To Do.
const listsPath = "/me/todo/lists/"

// pageSize is how many tasks one request asks for.
const pageSize = 100

// Client implements domain.Backend over Graph.
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

// notFound turns a 404 into the domain's error.
func notFound(err error) error {
	var se *webjson.StatusError
	if errors.As(err, &se) && se.Code == http.StatusNotFound {
		return domain.ErrNotFound
	}
	return err
}

type listsPage struct {
	Value []struct {
		ID   string `json:"id"`
		Name string `json:"displayName"`
	} `json:"value"`
	Next string `json:"@odata.nextLink"`
}

// Lists implements domain.Backend.
func (c Client) Lists() ([]domain.RemoteList, error) {
	var out []domain.RemoteList
	next := "/me/todo/lists"
	for next != "" {
		var page listsPage
		if err := c.web().Do(http.MethodGet, next, nil, &page); err != nil {
			return nil, err
		}
		for _, l := range page.Value {
			out = append(out, domain.RemoteList{ID: l.ID, Name: l.Name})
		}
		next = page.Next
	}
	return out, nil
}

// dateTime is Graph's {dateTime, timeZone}.
type dateTime struct {
	DateTime string `json:"dateTime"`
	TimeZone string `json:"timeZone"`
}

const graphLayout = "2006-01-02T15:04:05.0000000"

func (d *dateTime) time() time.Time {
	if d == nil || d.DateTime == "" {
		return time.Time{}
	}
	for _, layout := range []string{graphLayout, "2006-01-02T15:04:05", time.RFC3339Nano} {
		if t, err := time.Parse(layout, d.DateTime); err == nil {
			return t
		}
	}
	return time.Time{}
}

func stamp(t time.Time) *dateTime {
	if t.IsZero() {
		return nil
	}
	return &dateTime{DateTime: t.UTC().Format(graphLayout), TimeZone: "UTC"}
}

type remoteTask struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Importance string `json:"importance"`
	Status     string `json:"status"`
	Body       struct {
		Content string `json:"content"`
	} `json:"body"`
	Due       *dateTime `json:"dueDateTime"`
	Completed *dateTime `json:"completedDateTime"`
	Reminder  *dateTime `json:"reminderDateTime"`
	Created   string    `json:"createdDateTime"`
}

type tasksPage struct {
	Value []remoteTask `json:"value"`
	Next  string       `json:"@odata.nextLink"`
}

// Tasks implements domain.Backend.
func (c Client) Tasks(listID string) ([]domain.Task, error) {
	var out []domain.Task
	next := listsPath + url.PathEscape(listID) + "/tasks?$top=" + strconv.Itoa(pageSize)
	for next != "" {
		var page tasksPage
		if err := c.web().Do(http.MethodGet, next, nil, &page); err != nil {
			return nil, notFound(err)
		}
		for _, r := range page.Value {
			out = append(out, taskOf(r))
		}
		next = page.Next
	}
	return out, nil
}

func taskOf(r remoteTask) domain.Task {
	t := domain.Task{
		RemoteID: r.ID, Title: r.Title, Notes: r.Body.Content,
		Done: r.Status == "completed", Important: r.Importance == "high",
	}
	if d := r.Due.time(); !d.IsZero() {
		t.Due = domain.Day(d)
	}
	if t.Done {
		t.DoneAt = r.Completed.time()
	}
	t.Reminder = r.Reminder.time()
	if created, err := time.Parse(time.RFC3339Nano, r.Created); err == nil {
		t.CreatedAt = created
	}
	return t
}

// payload is the JSON sent to create or change a task; the date fields are always
// sent, null when empty, so clearing a date on the reader clears it at the service.
func payload(t domain.Task) map[string]interface{} {
	status, importance := "notStarted", "normal"
	if t.Done {
		status = "completed"
	}
	if t.Important {
		importance = "high"
	}
	p := map[string]interface{}{
		"title": t.Title, "importance": importance, "status": status,
		"body":             map[string]string{"content": t.Notes, "contentType": "text"},
		"dueDateTime":      stamp(t.Due),
		"isReminderOn":     !t.Reminder.IsZero(),
		"reminderDateTime": stamp(t.Reminder),
	}
	return p
}

// Create implements domain.Backend.
func (c Client) Create(listID string, t domain.Task) (string, error) {
	var made remoteTask
	err := c.web().Do(http.MethodPost, "/me/todo/lists/"+url.PathEscape(listID)+"/tasks", payload(t), &made)
	if err != nil {
		return "", err
	}
	if made.ID == "" {
		return "", errors.New("graphtodo: the service gave the task no id")
	}
	return made.ID, nil
}

func taskPath(listID, remoteID string) string {
	return listsPath + url.PathEscape(listID) + "/tasks/" + url.PathEscape(remoteID)
}

// Update implements domain.Backend.
func (c Client) Update(listID string, t domain.Task) error {
	return notFound(c.web().Do(http.MethodPatch, taskPath(listID, t.RemoteID), payload(t), nil))
}

// Delete implements domain.Backend.
func (c Client) Delete(listID, remoteID string) error {
	return notFound(c.web().Do(http.MethodDelete, taskPath(listID, remoteID), nil, nil))
}
