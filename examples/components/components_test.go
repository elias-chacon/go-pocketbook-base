package components

import (
	"testing"

	"go-pocketbook-base/examples/harness"
	"go-pocketbook-base/internal/toolkit/widget"
)

func TestEveryComponentExampleRendersAndRespondsToTouch(t *testing.T) {
	examples := map[string]func() widget.Widget{
		"FlexRow": FlexRow, "FlexGrow": FlexGrow, "FlexWrap": FlexWrap, "FlexColumn": FlexColumn,
		"GridLayout": GridLayout, "LabelExample": LabelExample, "ButtonExample": ButtonExample,
		"ToggleExample": ToggleExample, "ComboBoxExample": ComboBoxExample, "DropdownExample": DropdownExample,
		"MenuBarExample": MenuBarExample, "TextInputExample": TextInputExample, "NumberInputExample": NumberInputExample,
		"MaskedInputExample": MaskedInputExample, "SearchInputExample": SearchInputExample,
		"TextAreaExample": TextAreaExample, "TableExample": TableExample, "ProgressExample": ProgressExample,
		"GaugeExample": GaugeExample, "SliderExample": SliderExample, "SlideButtonExample": SlideButtonExample,
		"ChipsExample": ChipsExample, "CalendarExample": CalendarExample, "CardExample": CardExample, "ImageExample": ImageExample,
	}
	for name, build := range examples {
		doc, _ := harness.NewDoc(build())
		if harness.Inked(doc) == 0 {
			t.Errorf("%s draws nothing", name)
		}
		for _, w := range harness.Interactive(doc) {
			if w.Elem().Disabled {
				continue // a disabled element correctly ignores touch
			}
			if got := doc.TargetAt(w.Elem().Bounds.Center()); got != w {
				t.Errorf("%s: a touch at the center of %T reaches %T", name, w, got)
			}
			harness.Tap(doc, w) // runs the example's handlers
			doc.ClosePopups()
		}
	}
}

func TestModalExampleOpensAboveThePageAndTakesTheTouch(t *testing.T) {
	page := widget.NewButton("Page button")
	doc, _ := harness.NewDoc(page)
	m := ModalExample().(*widget.Modal)
	m.Open(doc)
	doc.Layout()
	if !doc.HasModal() {
		t.Fatal("the modal must be open")
	}
	for _, w := range harness.Interactive(doc) {
		if w == widget.Widget(page) {
			continue // behind the modal: checked below
		}
		if got := doc.TargetAt(w.Elem().Bounds.Center()); got != w {
			t.Errorf("a touch at the center of %T reaches %T", w, got)
		}
	}
	if doc.TargetAt(page.Bounds.Center()) != nil {
		t.Error("the page behind the modal must not get the touch")
	}
	m.Cancel()
	if doc.HasModal() || doc.TargetAt(page.Bounds.Center()) != page {
		t.Error("after Cancel the page works again")
	}
}
