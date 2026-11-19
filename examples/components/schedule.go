package components

import (
	"fmt"
	"time"

	"go-pocketbook-base/internal/toolkit/widget"
)

// MonthViewExample shows a month like the month view of Outlook: every day lists
// what is on it, and a second tap on the selected day opens it.
func MonthViewExample() widget.Widget {
	month := widget.NewMonthView(time.Now())
	month.Locale = widget.LocalePTBR
	month.Entries = func(day time.Time) []string {
		if day.Weekday() == time.Friday {
			return []string{"09:00 Reunião", "14:00 Dentista"}
		}
		return nil
	}
	month.OnChange = func(e *widget.Event) { fmt.Println("selected:", month.Selected.Format("2006-01-02")) }
	month.OnOpen = func(e *widget.Event) { fmt.Println("open:", month.Selected.Format("2006-01-02")) }
	return month
}

// TimeGridExample shows a work week: the hours down the side and the events as blocks.
func TimeGridExample() widget.Widget {
	monday := time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)
	days := make([]time.Time, 5)
	for i := range days {
		days[i] = monday.AddDate(0, 0, i)
	}
	grid := widget.NewTimeGrid(days...)
	grid.Locale = widget.LocalePTBR
	grid.Today = days[2]
	grid.Events = []widget.GridEvent{
		{ID: "a", Title: "Reunião", Detail: "09:00–10:30", Start: monday.Add(9 * time.Hour), End: monday.Add(10*time.Hour + 30*time.Minute)},
		{ID: "b", Title: "Dentista", Detail: "10:00–11:00", Start: monday.Add(10 * time.Hour), End: monday.Add(11 * time.Hour)},
		{ID: "c", Title: "Feriado", Start: days[3], End: days[3].AddDate(0, 0, 1), AllDay: true},
	}
	grid.OnEvent = func(id string) { fmt.Println("event:", id) }
	grid.OnSlot = func(at time.Time) { fmt.Println("new event at", at.Format("2006-01-02 15:04")) }
	return grid
}
