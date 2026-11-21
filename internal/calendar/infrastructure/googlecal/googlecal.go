// Package googlecal is the domain.Backend for Google Calendar over its REST API. It asks for the
// events of a window with singleEvents, so Google expands the repeating ones and each occurrence
// is a single event of the reader.
package googlecal

import (
	"net/url"
	"strings"
	"time"

	"go-pocketbook-base/internal/calendar/domain"
	"go-pocketbook-base/internal/kernel/webjson"
)

// DefaultBase is the address of the Google Calendar API.
const DefaultBase = "https://www.googleapis.com/calendar/v3"

const (
	pageSize = "250"
	noTitle  = "(no title)"
)

// Client implements domain.Backend over Google Calendar.
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

// Calendars implements domain.Backend.
func (c Client) Calendars() ([]domain.Calendar, error) {
	var out []domain.Calendar
	token := ""
	for {
		var page struct {
			Items []struct {
				ID      string `json:"id"`
				Summary string `json:"summary"`
			} `json:"items"`
			Next string `json:"nextPageToken"`
		}
		path := "/users/me/calendarList?maxResults=" + pageSize
		if token != "" {
			path += "&pageToken=" + url.QueryEscape(token)
		}
		if err := c.web().Do("GET", path, nil, &page); err != nil {
			return nil, err
		}
		for _, it := range page.Items {
			out = append(out, domain.Calendar{ID: it.ID, Name: it.Summary})
		}
		if token = page.Next; token == "" {
			return out, nil
		}
	}
}

// when is Google's start/end: a date for an all-day event, a date-time otherwise.
type when struct {
	Date     string `json:"date"`
	DateTime string `json:"dateTime"`
}

func (w when) time() (t time.Time, allDay bool) {
	if w.Date != "" {
		d, _ := time.Parse("2006-01-02", w.Date)
		return d, true
	}
	d, _ := time.Parse(time.RFC3339, w.DateTime)
	return d.UTC(), false
}

// Events implements domain.Backend.
func (c Client) Events(calendarID string, from, to time.Time) ([]domain.Event, error) {
	var out []domain.Event
	token := ""
	for {
		var page struct {
			Items []struct {
				ID          string `json:"id"`
				Status      string `json:"status"`
				Summary     string `json:"summary"`
				Description string `json:"description"`
				Location    string `json:"location"`
				Start       when   `json:"start"`
				End         when   `json:"end"`
			} `json:"items"`
			Next string `json:"nextPageToken"`
		}
		q := url.Values{"timeMin": {from.UTC().Format(time.RFC3339)}, "timeMax": {to.UTC().Format(time.RFC3339)},
			"singleEvents": {"true"}, "orderBy": {"startTime"}, "maxResults": {pageSize}}
		if token != "" {
			q.Set("pageToken", token)
		}
		if err := c.web().Do("GET", "/calendars/"+url.PathEscape(calendarID)+"/events?"+q.Encode(), nil, &page); err != nil {
			return nil, err
		}
		for _, it := range page.Items {
			if it.Status == "cancelled" {
				continue
			}
			start, allDay := it.Start.time()
			end, _ := it.End.time()
			title := strings.TrimSpace(it.Summary)
			if title == "" {
				title = noTitle
			}
			out = append(out, domain.Event{UID: it.ID, Title: title, Description: it.Description, Location: it.Location, Start: start, End: end, AllDay: allDay})
		}
		if token = page.Next; token == "" {
			return out, nil
		}
	}
}
