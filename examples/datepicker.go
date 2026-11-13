// examples/datepicker.go
package examples

import (
	"time"

	"go-pocketbook-base/internal/widget"
)

// DatePicker is a calendar with the chosen date written below it and a
// "Today" shortcut.
type DatePicker struct {
	Root     widget.Widget
	Calendar *widget.Calendar

	chosen *widget.Label
}

// NewDatePicker builds the picker. onPick receives every chosen date.
func NewDatePicker(initial time.Time, onPick func(time.Time)) *DatePicker {
	p := &DatePicker{Calendar: widget.NewCalendar(initial), chosen: widget.NewLabel("")}
	p.Calendar.Locale = widget.LocalePTBR

	pick := func(t time.Time) {
		p.Calendar.Selected = t
		p.Calendar.Year, p.Calendar.Month = t.Year(), t.Month()
		p.show()
		onPick(t)
	}
	p.Calendar.OnChange = func(e *widget.Event) { pick(p.Calendar.Selected) }

	today := widget.NewButton("Today")
	today.OnClick = func(e *widget.Event) { pick(time.Now()) }

	p.show()
	p.Root = widget.NewColumn(20, p.Calendar, p.chosen, today)
	return p
}

func (p *DatePicker) show() {
	if p.Calendar.Selected.IsZero() {
		p.chosen.Text = "No date chosen"
		return
	}
	p.chosen.Text = "Date: " + p.Calendar.Selected.Format("02/01/2006")
}
