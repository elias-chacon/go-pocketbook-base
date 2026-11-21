package graphcal

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCalendarsAndTheEventsOfAWindowInUTCAcrossPages(t *testing.T) {
	var prefer, query string
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case r.URL.Path == "/me/calendars":
			_, _ = w.Write([]byte(`{"value":[{"id":"CAL 1","name":"Calendar"}]}`))
		case r.URL.Query().Get("p") == "2":
			_, _ = w.Write([]byte(`{"value":[{"id":"e3","subject":"","start":{"dateTime":"2026-05-22T00:00:00.0000000"},"end":{"dateTime":"2026-05-23T00:00:00.0000000"},"isAllDay":true}]}`))
		default:
			prefer, query = r.Header.Get("Prefer"), r.URL.RawQuery
			_, _ = w.Write([]byte(`{"value":[
			 {"id":"e1","subject":"Dentist","bodyPreview":"bring the card","start":{"dateTime":"2026-05-20T12:00:00.0000000"},"end":{"dateTime":"2026-05-20T13:00:00.0000000"},"location":{"displayName":"Downtown"}},
			 {"id":"e2","subject":"Gone","isCancelled":true,"start":{"dateTime":"2026-05-21T12:00:00.0000000"},"end":{"dateTime":"2026-05-21T13:00:00.0000000"}}],
			 "@odata.nextLink":"` + srv.URL + `/me/calendars/CAL%201/calendarView?p=2"}`))
		}
	}))
	defer srv.Close()
	c := Client{HTTP: srv.Client(), Token: func() (string, error) { return "tok", nil }, Base: srv.URL}
	cals, err := c.Calendars()
	if err != nil || len(cals) != 1 || cals[0].ID != "CAL 1" || cals[0].Name != "Calendar" {
		t.Fatalf("%+v %v", cals, err)
	}
	from := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	evs, err := c.Events("CAL 1", from, from.AddDate(0, 1, 0))
	if err != nil || len(evs) != 2 {
		t.Fatalf("%+v %v", evs, err)
	}
	if prefer != `outlook.timezone="UTC"` || query == "" {
		t.Errorf("prefer %q query %q", prefer, query)
	}
	e := evs[0]
	if e.UID != "e1" || e.Title != "Dentist" || e.Location != "Downtown" || e.Description != "bring the card" ||
		!e.Start.Equal(time.Date(2026, 5, 20, 12, 0, 0, 0, time.UTC)) || e.AllDay {
		t.Errorf("event = %+v", e)
	}
	if evs[1].Title != "(no title)" || !evs[1].AllDay {
		t.Errorf("an all-day event with no subject = %+v", evs[1])
	}
}

func TestAnErrorAnswerIsReturned(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) }))
	defer srv.Close()
	c := Client{HTTP: srv.Client(), Base: srv.URL}
	if _, err := c.Calendars(); err == nil {
		t.Error("a 403 must be an error")
	}
}
