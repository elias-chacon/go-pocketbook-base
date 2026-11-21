package googlecal

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCalendarsAndTheExpandedEventsAcrossPages(t *testing.T) {
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch {
		case r.URL.Path == "/users/me/calendarList":
			_, _ = w.Write([]byte(`{"items":[{"id":"ana@gmail.com","summary":"Ana"},{"id":"x#holiday","summary":"Holidays"}]}`))
		case r.URL.Query().Get("pageToken") == "t2":
			_, _ = w.Write([]byte(`{"items":[{"id":"g3","summary":"Holiday","start":{"date":"2026-05-25"},"end":{"date":"2026-05-26"}}]}`))
		default:
			query = r.URL.RawQuery
			_, _ = w.Write([]byte(`{"items":[
			 {"id":"g1","summary":"Lunch","location":"Cafe","description":"x","start":{"dateTime":"2026-05-20T12:00:00-03:00"},"end":{"dateTime":"2026-05-20T13:00:00-03:00"}},
			 {"id":"g2","status":"cancelled","summary":"Gone","start":{"dateTime":"2026-05-21T12:00:00Z"},"end":{"dateTime":"2026-05-21T13:00:00Z"}}],
			 "nextPageToken":"t2"}`))
		}
	}))
	defer srv.Close()
	c := Client{HTTP: srv.Client(), Token: func() (string, error) { return "tok", nil }, Base: srv.URL}
	cals, err := c.Calendars()
	if err != nil || len(cals) != 2 || cals[1].ID != "x#holiday" {
		t.Fatalf("%+v %v", cals, err)
	}
	from := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	evs, err := c.Events("ana@gmail.com", from, from.AddDate(0, 1, 0))
	if err != nil || len(evs) != 2 {
		t.Fatalf("%+v %v", evs, err)
	}
	if query == "" || evs[0].UID != "g1" || evs[0].Location != "Cafe" || !evs[0].Start.Equal(time.Date(2026, 5, 20, 15, 0, 0, 0, time.UTC)) || evs[0].AllDay {
		t.Errorf("query %q event %+v", query, evs[0])
	}
	if !evs[1].AllDay || !evs[1].Start.Equal(time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)) || !evs[1].End.Equal(time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("an all-day event = %+v", evs[1])
	}
}
