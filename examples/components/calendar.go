package components

import (
	"fmt"
	"time"

	"go-pocketbook-base/internal/toolkit/widget"
)

// CalendarExample picks a day.
func CalendarExample() widget.Widget {
	calendar := widget.NewCalendar(time.Now())
	calendar.Locale = widget.LocalePTBR
	calendar.OnChange = func(e *widget.Event) { fmt.Println("date:", calendar.Selected.Format("2006-01-02")) }
	// Marked puts a dot under the days that have something (here, every Friday).
	calendar.Marked = func(day time.Time) bool { return day.Weekday() == time.Friday }
	return calendar
}
