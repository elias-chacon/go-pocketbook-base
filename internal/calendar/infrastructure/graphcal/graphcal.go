// Package graphcal is the domain.Backend for the calendars of Outlook.com and Microsoft 365 over
// Microsoft Graph. It reads the calendarView of a window: Graph expands the repeating events, and
// the times come in UTC (the Prefer header), so each occurrence is a single event of the reader.
package graphcal

import (
	"net/url"
	"strings"
	"time"

	"go-pocketbook-base/internal/calendar/domain"
	"go-pocketbook-base/internal/kernel/webjson"
)

// DefaultBase is Microsoft Graph's address.
const DefaultBase = "https://graph.microsoft.com/v1.0"

const (
	pageSize     = "100"
	noTitle      = "(no title)"
	graphLayout  = "2006-01-02T15:04:05.0000000"
	graphLayout0 = "2006-01-02T15:04:05"
)

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
	return webjson.Client{HTTP: c.HTTP, Token: c.Token, Base: base, Header: map[string]string{"Prefer": `outlook.timezone="UTC"`}}
}

// Calendars implements domain.Backend.
func (c Client) Calendars() ([]domain.Calendar, error) {
	var out []domain.Calendar
	next := "/me/calendars?$top=" + pageSize
	for next != "" {
		var page struct {
			Value []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"value"`
			Next string `json:"@odata.nextLink"`
		}
		if err := c.web().Do("GET", next, nil, &page); err != nil {
			return nil, err
		}
		for _, v := range page.Value {
			out = append(out, domain.Calendar{ID: v.ID, Name: v.Name})
		}
		next = page.Next
	}
	return out, nil
}

type dateTime struct {
	DateTime string `json:"dateTime"`
}

func (d dateTime) time() time.Time {
	for _, layout := range []string{graphLayout, graphLayout0, time.RFC3339Nano} {
		if t, err := time.Parse(layout, d.DateTime); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

// Events implements domain.Backend.
func (c Client) Events(calendarID string, from, to time.Time) ([]domain.Event, error) {
	q := url.Values{"startDateTime": {from.UTC().Format(time.RFC3339)}, "endDateTime": {to.UTC().Format(time.RFC3339)}, "$top": {pageSize}}
	next := "/me/calendars/" + url.PathEscape(calendarID) + "/calendarView?" + q.Encode()
	var out []domain.Event
	for next != "" {
		var page struct {
			Value []struct {
				ID          string   `json:"id"`
				Subject     string   `json:"subject"`
				Preview     string   `json:"bodyPreview"`
				Start       dateTime `json:"start"`
				End         dateTime `json:"end"`
				AllDay      bool     `json:"isAllDay"`
				IsCancelled bool     `json:"isCancelled"`
				Location    struct {
					Name string `json:"displayName"`
				} `json:"location"`
			} `json:"value"`
			Next string `json:"@odata.nextLink"`
		}
		if err := c.web().Do("GET", next, nil, &page); err != nil {
			return nil, err
		}
		for _, v := range page.Value {
			if v.IsCancelled {
				continue
			}
			title := strings.TrimSpace(v.Subject)
			if title == "" {
				title = noTitle
			}
			out = append(out, domain.Event{
				UID: v.ID, Title: title, Description: v.Preview, Location: v.Location.Name,
				Start: v.Start.time(), End: v.End.time(), AllDay: v.AllDay,
			})
		}
		next = page.Next
	}
	return out, nil
}
