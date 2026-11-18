// examples/recipes/settings.go
package recipes

import (
	"fmt"

	"go-pocketbook-base/internal/toolkit/widget"
)

// Settings is a preferences screen that mixes a switch, a slider, a
// combo-box and chips.
type Settings struct {
	Root       widget.Widget
	WiFi       *widget.Toggle
	Brightness *widget.Slider
	Language   *widget.ComboBox
	Topics     *widget.ChipGroup
}

// NewSettings builds the screen with sensible defaults.
func NewSettings() *Settings {
	s := &Settings{
		WiFi:       widget.NewSwitch("Wi-Fi", true),
		Brightness: widget.NewSlider(0, 100, 5, 50),
		Language: widget.NewComboBox("Language",
			widget.Option{Label: "Português", Value: "pt"},
			widget.Option{Label: "English", Value: "en"},
			widget.Option{Label: "Español", Value: "es"}),
		Topics: widget.NewChipGroup(false),
	}
	s.Language.SetValue("pt")

	// The slider grows to fill the row; its value is shown at the right.
	level := widget.NewLabel("50%")
	s.Brightness.Style.FlexGrow = 1
	s.Brightness.OnInput = func(e *widget.Event) {
		level.Text = fmt.Sprintf("%.0f%%", s.Brightness.Value)
	}

	for _, topic := range []string{"Fiction", "Science", "History"} {
		s.Topics.AppendChild(widget.NewChip(topic))
	}

	s.Root = widget.NewColumn(24,
		widget.NewCard("Connectivity", "", s.WiFi),
		widget.NewCard("Display", "Brightness", widget.NewRow(12, s.Brightness, level)),
		widget.NewCard("Language", "", s.Language),
		widget.NewCard("Topics", "Pick as many as you like", s.Topics))
	return s
}
