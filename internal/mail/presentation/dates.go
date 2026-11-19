package presentation

import "time"

// shortDate is the date of a list row: the time for today, "yesterday", the day and
// month within the year, and the full date before that.
func shortDate(t, now time.Time, yesterday string) string {
	if t.IsZero() {
		return ""
	}
	t, now = t.In(now.Location()), now.In(now.Location())
	day := func(x time.Time) time.Time { return time.Date(x.Year(), x.Month(), x.Day(), 0, 0, 0, 0, x.Location()) }
	switch days := int(day(now).Sub(day(t)).Hours() / 24); {
	case days == 0:
		return t.Format("15:04")
	case days == 1:
		return yesterday
	case t.Year() == now.Year():
		return t.Format("02/01")
	}
	return t.Format("02/01/2006")
}
