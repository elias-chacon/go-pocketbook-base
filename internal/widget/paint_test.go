package widget

import (
	"testing"
	"time"

	"go-pocketbook-base/internal/gfx"
)

// holdKeyboard never answers, so the field that asked for it stays focused.
type holdKeyboard struct{}

func (holdKeyboard) Request(KeyboardRequest) {}

// inkedPixels counts dark pixels of the rendered document.
func inkedPixels(d *Document) int {
	n := 0
	for _, p := range d.Render().Pix {
		if p < 128 {
			n++
		}
	}
	return n
}

// everyWidgetInEveryState builds one of each component in its main states.
func everyWidgetInEveryState() []Widget {
	off := NewLabel("disabled label")
	off.Disabled = true

	primary, ghost, icon := NewButton("p"), NewButton("g"), NewIconButton(gfx.IconPlus)
	primary.SetVariant(VariantPrimary)
	ghost.SetVariant(VariantGhost)
	disabledPrimary := NewButton("x")
	disabledPrimary.SetVariant(VariantPrimary)
	disabledPrimary.Disabled = true

	toggles := []Widget{}
	for _, on := range []bool{true, false} {
		for _, t := range []*Toggle{NewCheckbox("c", on), NewRadio("r", "g", "v"), NewSwitch("s", on)} {
			t.Checked = on
			toggles = append(toggles, t)
		}
	}
	disabledToggle := NewSwitch("d", true)
	disabledToggle.Disabled = true

	readOnly, disabledInput := NewInput("ro"), NewInput("off")
	readOnly.ReadOnly, disabledInput.Disabled = true, true
	password := NewInput("")
	password.Type, password.Value = TypePassword, "secret"
	search := NewSearchInput("s")
	search.Value = "q"
	number := NewNumberInput(5, 1)
	number.NoSteppers = true

	empty, filled := NewTable(2, TableColumn{Title: "A", Sortable: true}), NewTable(2, TableColumn{Title: "A", Sortable: true}, TableColumn{Title: "B"})
	filled.Rows = [][]string{{"1", "x"}, {"2", "y"}, {"3", "z"}}
	filled.SortCol, filled.SortDesc, filled.Selected, filled.Page = 0, true, 1, 1

	slider := NewSlider(0, 10, 1, 5)
	disabledSlider := NewSlider(0, 10, 1, 5)
	disabledSlider.Disabled = true

	selectedChip, closable, offChip := NewChip("a"), NewChip("b"), NewChip("c")
	selectedChip.Selected, closable.Closable, offChip.Disabled = true, true, true

	cal := NewCalendar(time.Time{})
	cal.Locale = LocalePTBR
	calSel := NewCalendar(time.Now())
	calSel.Locale.FirstWeekday = time.Monday

	gauge := NewGauge(0, 100, 0, "%")
	gauge.Label = "g"
	return append(toggles,
		off, primary, ghost, icon, disabledPrimary, disabledToggle,
		readOnly, disabledInput, password, search, number, NewMaskedInput("00-00", "12"),
		NewTextArea("ta", 2), empty, filled,
		NewProgress(0), NewProgress(100), gauge, NewGauge(0, 100, 100, "%"),
		slider, disabledSlider, selectedChip, closable, offChip, NewChipGroup(true, NewChip("x")),
		cal, calSel, NewCard("title", "sub", NewLabel("body")),
		NewComboBox("pick"), NewDropdown("menu"), NewMenuBar(&Menu{Label: "m", Selected: true}, &Menu{Label: "n"}))
}

func TestEveryWidgetPaintsInEveryState(t *testing.T) {
	d := NewDocument(1404, 6000)
	d.Keyboard = holdKeyboard{}
	col := NewColumn(10, everyWidgetInEveryState()...)
	col.Style.AlignItems = AlignStart
	d.SetRoot(col)
	d.OnInvalidate = func() {} // hooked like the device does

	if inkedPixels(d) == 0 {
		t.Fatal("nothing was drawn")
	}
}

func TestOpenPopupsAndFocusRingsPaint(t *testing.T) {
	combo := NewComboBox("pick", Option{"A", "a"}, Option{"B", "b"})
	combo.Index = 0
	drop := NewDropdown("menu", MenuItem{Label: "one", Icon: gfx.IconPlus}, MenuItem{Separator: true}, MenuItem{Label: "two", Disabled: true})
	bar := NewMenuBar(&Menu{Label: "More", Items: []MenuItem{{Label: "Exit"}}})
	text, area := NewInput("name"), NewTextArea("notes", 2)
	col := NewColumn(30, combo, drop, bar, text, area)
	col.Style.AlignItems = AlignStart
	d := NewDocument(1404, 1872)
	d.Keyboard = holdKeyboard{}
	d.SetRoot(col)
	d.Layout()
	base := inkedPixels(d)

	for _, w := range []Widget{combo, drop} {
		tap(d, w)
		if !d.PopupOpenFor(w) || inkedPixels(d) <= base {
			t.Errorf("%T: an open popup must be drawn", w)
		}
		d.ClosePopups()
	}
	tapAt(d, bar.MenuRect(0).Center())
	if !d.HasPopup() {
		t.Error("menu bar popup")
	}
	d.ClosePopups()

	for _, w := range []Widget{text, area} { // a keyboard that never answers leaves the field focused
		tap(d, w)
		if d.Focused() != w {
			t.Errorf("%T must be focused while the keyboard is open", w)
		}
		_ = inkedPixels(d) // paints the focus ring
	}
}

func TestInvalidateAndDocumentHooks(t *testing.T) {
	btn := NewButton("x")
	calls := 0
	d := testDoc(btn)
	d.OnInvalidate = func() { calls++ }
	btn.OnClick = func(*Event) {}
	tap(d, btn)
	if calls == 0 {
		t.Error("a click must request a repaint")
	}
	if btn.Self() != Widget(btn) {
		t.Error("Self must return the outer widget")
	}
	bare := NewDocument(1404, 100) // no root: nothing to paint or hit
	if bare.TargetAt(geomPt(1, 1)) != nil || bare.Render() == nil {
		t.Error("an empty document must still work")
	}
	bare.HandlePointer(PointerDown, 5, 5)
	bare.HandlePointer(PointerMove, 5, 5)
	bare.HandlePointer(PointerUp, 5, 5)
}

func TestIconButtonIsASquareTouchTarget(t *testing.T) {
	b := NewIconButton(gfx.IconSearch)
	d := testDoc(b)
	if b.Bounds.W < ControlHeight || b.Bounds.H < ControlHeight {
		t.Errorf("icon button = %vx%v", b.Bounds.W, b.Bounds.H)
	}
	clicks := &counter{}
	b.OnClick = clicks.handler()
	tap(d, b)
	if clicks.n != 1 {
		t.Errorf("clicks = %d", clicks.n)
	}
}
