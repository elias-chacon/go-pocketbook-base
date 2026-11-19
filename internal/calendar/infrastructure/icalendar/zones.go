package icalendar

import (
	"strings"
	"time"
	_ "time/tzdata" // the reader may have no zoneinfo: carry the time zone database
)

// windowsZones maps the time zone names Outlook and Exchange write (Windows names)
// to IANA names. It covers the common ones; an unknown name falls back to UTC with
// a warning, so the event is never lost, only possibly shifted.
var windowsZones = map[string]string{
	"UTC":                             "UTC",
	"GMT Standard Time":               "Europe/London",
	"Greenwich Standard Time":         "Atlantic/Reykjavik",
	"W. Europe Standard Time":         "Europe/Berlin",
	"Romance Standard Time":           "Europe/Paris",
	"Central Europe Standard Time":    "Europe/Budapest",
	"Central European Standard Time":  "Europe/Warsaw",
	"E. Europe Standard Time":         "Europe/Chisinau",
	"Turkey Standard Time":            "Europe/Istanbul",
	"Russian Standard Time":           "Europe/Moscow",
	"Israel Standard Time":            "Asia/Jerusalem",
	"Egypt Standard Time":             "Africa/Cairo",
	"South Africa Standard Time":      "Africa/Johannesburg",
	"Arabian Standard Time":           "Asia/Dubai",
	"India Standard Time":             "Asia/Calcutta",
	"China Standard Time":             "Asia/Shanghai",
	"Singapore Standard Time":         "Asia/Singapore",
	"Tokyo Standard Time":             "Asia/Tokyo",
	"Korea Standard Time":             "Asia/Seoul",
	"AUS Eastern Standard Time":       "Australia/Sydney",
	"New Zealand Standard Time":       "Pacific/Auckland",
	"Atlantic Standard Time":          "America/Halifax",
	"Eastern Standard Time":           "America/New_York",
	"Central Standard Time":           "America/Chicago",
	"Mountain Standard Time":          "America/Denver",
	"Pacific Standard Time":           "America/Los_Angeles",
	"E. South America Standard Time":  "America/Sao_Paulo",
	"Central Brazilian Standard Time": "America/Cuiaba",
	"Argentina Standard Time":         "America/Argentina/Buenos_Aires",
	"SA Pacific Standard Time":        "America/Bogota",
	"Pacific SA Standard Time":        "America/Santiago",
}

// resolveZone finds the location of a TZID: an IANA name (also when written with a
// prefix such as "/mozilla.org/20050126_1/Europe/Paris") or a Windows name. The
// second result is false when the name is not known.
func resolveZone(tzid string) (*time.Location, bool) {
	tzid = strings.Trim(strings.TrimSpace(tzid), `"`)
	if tzid == "" {
		return time.UTC, true
	}
	if iana, ok := windowsZones[tzid]; ok {
		tzid = iana
	}
	if loc, err := time.LoadLocation(tzid); err == nil {
		return loc, true
	}
	// a path prefix: keep the last two (Area/City) or three (Area/Region/City) parts
	parts := strings.Split(strings.Trim(tzid, "/"), "/")
	for n := 3; n >= 2; n-- {
		if len(parts) > n {
			if loc, err := time.LoadLocation(strings.Join(parts[len(parts)-n:], "/")); err == nil {
				return loc, true
			}
		}
	}
	return time.UTC, false
}
