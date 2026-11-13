package examples

import (
	"fmt"
	"time"

	"go-pocketbook-base/internal/gfx"
	"go-pocketbook-base/internal/widget"
)

// This file has one small function per component. The documentation embeds
// each function (tools/docsync), so every snippet there compiles and runs.

// FlexRow spreads three items along a row with the free space between them.
func FlexRow() widget.Widget {
	row := widget.NewRow(12, widget.NewButton("A"), widget.NewButton("B"), widget.NewButton("C"))
	row.Style.JustifyContent = widget.JustifySpaceBetween
	return row
}

// FlexGrow gives the leftover width to one child.
func FlexGrow() widget.Widget {
	fixed := widget.NewButton("Fixed")
	fill := widget.NewButton("Grows")
	fill.Style.FlexGrow = 1
	return widget.NewRow(12, fixed, fill)
}

// FlexWrap breaks the children into several lines when they do not fit.
func FlexWrap() widget.Widget {
	panel := widget.NewRow(12)
	panel.Style.FlexWrap = true
	panel.Style.AlignItems = widget.AlignStart
	for _, name := range []string{"one", "two", "three", "four", "five", "six"} {
		panel.AppendChild(widget.NewButton(name))
	}
	return panel
}

// FlexColumn stacks the children; they stretch to the width of the column.
func FlexColumn() widget.Widget {
	return widget.NewColumn(12, widget.NewButton("First"), widget.NewButton("Second"))
}

// GridLayout uses three columns (1fr, 1fr and 120 px); one child spans two.
func GridLayout() widget.Widget {
	wide := widget.NewButton("spans two columns")
	wide.Style.GridColumnSpan = 2
	columns := []widget.Track{widget.FrTrack(1), widget.FrTrack(1), widget.PxTrack(120)}
	return widget.NewGrid(12, columns,
		widget.NewButton("1fr"), widget.NewButton("1fr"), widget.NewButton("120 px"), wide)
}

// LabelExample shows a bold title and a muted paragraph that wraps by itself.
func LabelExample() widget.Widget {
	title := widget.NewLabel("A bold title")
	title.Style.FontSize, title.Style.Bold = widget.FontTitle, true
	note := widget.NewLabel("Long text wraps by itself inside the width it has available.")
	note.Style.Color = widget.Muted
	return widget.NewColumn(8, title, note)
}

// ButtonExample shows the variants, an icon, an icon-only and a disabled button.
func ButtonExample() widget.Widget {
	save := widget.NewButton("Save")
	save.SetVariant(widget.VariantPrimary)
	save.OnClick = func(e *widget.Event) { fmt.Println("saved") }

	search := widget.NewButton("Search")
	search.Icon = gfx.IconSearch

	add := widget.NewIconButton(gfx.IconPlus)

	off := widget.NewButton("Unavailable")
	off.Disabled = true // greyed out, ignores touch

	return widget.NewRow(12, save, search, add, off)
}

// ToggleExample shows a check-box, two radio buttons of one group and a switch.
func ToggleExample() widget.Widget {
	news := widget.NewCheckbox("Send me the news", true)
	news.OnChange = func(e *widget.Event) { fmt.Println("news:", news.Checked) }

	free := widget.NewRadio("Free", "plan", "free") // same name = same group
	pro := widget.NewRadio("Pro", "plan", "pro")
	pro.Checked = true
	for _, r := range []*widget.Toggle{free, pro} {
		r.OnChange = func(e *widget.Event) {
			t := e.Target.(*widget.Toggle)
			fmt.Println("plan:", t.Value)
		}
	}

	wifi := widget.NewSwitch("Wi-Fi", true)
	return widget.NewColumn(4, news, free, pro, wifi)
}

// ComboBoxExample keeps one option from a list.
func ComboBoxExample() widget.Widget {
	city := widget.NewComboBox("Choose a city",
		widget.Option{Label: "São Paulo", Value: "sp"},
		widget.Option{Label: "Rio de Janeiro", Value: "rj"},
		widget.Option{Label: "Curitiba", Value: "cwb"})
	city.OnChange = func(e *widget.Event) { fmt.Println("city:", city.Value()) }
	return city
}

// DropdownExample opens a menu of actions.
func DropdownExample() widget.Widget {
	return widget.NewDropdown("Actions",
		widget.MenuItem{Label: "Duplicate", Icon: gfx.IconPlus, OnSelect: func() { fmt.Println("duplicate") }},
		widget.MenuItem{Label: "Rename", OnSelect: func() { fmt.Println("rename") }},
		widget.MenuItem{Separator: true},
		widget.MenuItem{Label: "Delete", Icon: gfx.IconClose, OnSelect: func() { fmt.Println("delete") }},
		widget.MenuItem{Label: "Archive", Disabled: true})
}

// MenuBarExample has a direct item and a menu with a sub-menu.
func MenuBarExample() widget.Widget {
	return widget.NewMenuBar(
		&widget.Menu{Label: "Home", Selected: true, OnSelect: func() { fmt.Println("home") }},
		&widget.Menu{Label: "More", Items: []widget.MenuItem{
			{Label: "About", OnSelect: func() { fmt.Println("about") }},
			{Separator: true},
			{Label: "Quit", Icon: gfx.IconClose, OnSelect: func() { fmt.Println("quit") }},
		}})
}

// TextInputExample has a text field and a password field.
func TextInputExample() widget.Widget {
	name := widget.NewInput("Full name")
	name.MaxLength = 40
	name.OnChange = func(e *widget.Event) { fmt.Println("name:", name.Value) }

	password := widget.NewInput("Password")
	password.Type = widget.TypePassword // shown as bullets
	return widget.NewColumn(12, name, password)
}

// NumberInputExample has − / + buttons, a range, decimals and a prefix.
func NumberInputExample() widget.Widget {
	quantity := widget.NewNumberInput(3, 0)
	quantity.Min, quantity.Max = 0, 99
	quantity.OnChange = func(e *widget.Event) { fmt.Println("quantity:", quantity.Number()) }

	price := widget.NewNumberInput(19.9, 2)
	price.Prefix, price.DecimalSeparator, price.Step = "R$ ", ",", 0.5
	return widget.NewColumn(12, quantity, price)
}

// MaskedInputExample formats the typed digits with a mask.
func MaskedInputExample() widget.Widget {
	phone := widget.NewMaskedInput("(00) 00000-0000", "11987654321")
	phone.OnChange = func(e *widget.Event) {
		fmt.Println("shown:", phone.Display(), "digits:", phone.Value)
	}
	return phone
}

// SearchInputExample has the magnifier and clear buttons.
func SearchInputExample() widget.Widget {
	search := widget.NewSearchInput("Search books…")
	search.OnSubmit = func(e *widget.Event) { fmt.Println("searching:", search.Value) }
	return search
}

// TextAreaExample is a multi-line field.
func TextAreaExample() widget.Widget {
	notes := widget.NewTextArea("Notes…", 4)
	notes.OnChange = func(e *widget.Event) { fmt.Println(len(notes.Value), "characters") }
	return notes
}

// TableExample is a sortable, selectable, paged table.
func TableExample() widget.Widget {
	table := widget.NewTable(4,
		widget.TableColumn{Title: "Book", Flex: 3, Sortable: true},
		widget.TableColumn{Title: "Pages", Flex: 1, Align: gfx.AlignEnd, Sortable: true})
	table.Rows = [][]string{
		{"Dom Casmurro", "256"}, {"Vidas Secas", "176"}, {"Iracema", "160"},
		{"Macunaíma", "224"}, {"O Cortiço", "320"}, {"A Hora da Estrela", "88"},
	}
	table.OnSelect = func(e *widget.Event) {
		fmt.Println("selected:", table.Rows[table.Selected][0]) // Selected indexes Rows
	}
	return table
}

// ProgressExample is a progress bar; change Value and the bar follows.
func ProgressExample() widget.Widget {
	bar := widget.NewProgress(35)
	bar.Value = 60 // 0..100
	return bar
}

// GaugeExample is a semicircular meter.
func GaugeExample() widget.Widget {
	speed := widget.NewGauge(0, 240, 135, " km/h")
	speed.Label = "Speed"
	return speed
}

// SliderExample is a bar with a knob you can drag; Step makes it snap.
func SliderExample() widget.Widget {
	volume := widget.NewSlider(0, 100, 5, 40)
	volume.OnInput = func(e *widget.Event) { fmt.Println("dragging:", volume.Value) }
	volume.OnChange = func(e *widget.Event) { fmt.Println("released:", volume.Value) }
	return volume
}

// SlideButtonExample is a slider with three resting places: left, center, right.
func SlideButtonExample() widget.Widget {
	position := widget.NewSlider(0, 100, 50, 50) // step 50: 0, 50 or 100
	position.Style.Width = widget.Px(420)        // configurable size...
	position.ThumbSize = 72                      // ...and knob
	position.OnChange = func(e *widget.Event) {
		names := map[float64]string{0: "left", 50: "center", 100: "right"}
		fmt.Println("position:", names[position.Value])
	}
	return position
}

// ChipsExample has selectable chips and removable chips.
func ChipsExample() widget.Widget {
	topics := widget.NewChipGroup(false) // true = only one selected at a time
	for _, name := range []string{"Fiction", "Science", "History"} {
		chip := widget.NewChip(name)
		chip.OnChange = func(e *widget.Event) { fmt.Println("topics:", topics.SelectedLabels()) }
		topics.AppendChild(chip)
	}

	tags := widget.NewChipGroup(false)
	for _, name := range []string{"#reading", "#e-ink"} {
		chip := widget.NewChip(name)
		chip.Closable = true // the × removes the chip
		tags.AppendChild(chip)
	}
	return widget.NewColumn(16, topics, tags)
}

// CalendarExample picks a day.
func CalendarExample() widget.Widget {
	calendar := widget.NewCalendar(time.Now())
	calendar.Locale = widget.LocalePTBR
	calendar.OnChange = func(e *widget.Event) { fmt.Println("date:", calendar.Selected.Format("2006-01-02")) }
	return calendar
}

// CardExample groups a title, a subtitle, text and actions.
func CardExample() widget.Widget {
	ok := widget.NewButton("Learn more")
	ok.SetVariant(widget.VariantPrimary)
	later := widget.NewButton("Later")
	return widget.NewCard("Card title", "Optional subtitle",
		widget.NewLabel("A card groups related content."),
		widget.NewRow(14, ok, later))
}
