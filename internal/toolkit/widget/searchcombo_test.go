package widget

import (
	"fmt"
	"testing"
)

// manyOptions is a list of 60 options: "Option 01 ... Option 60" and one with accents.
func manyOptions() []Option {
	var out []Option
	for i := 1; i <= 60; i++ {
		out = append(out, Option{Label: fmt.Sprintf("Option %02d", i), Value: fmt.Sprint(i)})
	}
	out[41].Label = "Português (Brasil)"
	return out
}

func searchBox() (*Document, *SearchComboBox, *FakeKeyboard) {
	sb := NewSearchComboBox("Pick one", manyOptions()...)
	sb.ID = "sb"
	d := testDoc(sb)
	kb := &FakeKeyboard{}
	d.Keyboard = kb
	return d, sb, kb
}

// optionButtons lists the options on the page of the open window.
func optionButtons(d *Document) []*Button {
	var out []*Button
	walk(d.modal, func(w Widget) bool {
		if b, ok := w.(*Button); ok && len(b.ID) > 7 && b.ID[:7] == "sb-opt-" {
			out = append(out, b)
		}
		return true
	})
	return out
}

func TestSearchComboBoxOpensOnThePageOfTheChosenOption(t *testing.T) {
	d, sb, _ := searchBox()
	sb.SetValue("55")
	tap(d, sb)
	if !d.HasModal() {
		t.Fatal("a tap opens the window")
	}
	d.Layout()
	var ids []string
	for _, b := range optionButtons(d) {
		ids = append(ids, b.Text)
	}
	found := false
	for _, id := range ids {
		found = found || id == "Option 55"
	}
	if !found {
		t.Errorf("the window opens where the chosen option is: %v", ids)
	}
	if sb.pager.Page() == 0 {
		t.Error("the page is not the first one")
	}
}

func TestSearchComboBoxFiltersWithoutCaseOrAccents(t *testing.T) {
	d, sb, kb := searchBox()
	tap(d, sb)
	kb.Reply = "PORTUGUES"

	d.Layout()
	tap(d, find(d, "sb-search"))
	d.Layout()
	opts := optionButtons(d)
	if len(opts) != 1 || opts[0].Text != "Português (Brasil)" {
		t.Fatalf("filtered: %d options", len(opts))
	}
	kb.Reply = "zzz"
	tap(d, find(d, "sb-search"))
	d.Layout()
	if len(optionButtons(d)) != 0 {
		t.Error("nothing matches")
	}
}

func TestSearchComboBoxChoosingClosesAndTellsTheHandler(t *testing.T) {
	d, sb, kb := searchBox()
	c := &counter{}
	sb.OnChange = c.handler()
	tap(d, sb)
	kb.Reply = "option 07"
	d.Layout()
	tap(d, find(d, "sb-search"))
	d.Layout()
	opts := optionButtons(d)
	if len(opts) != 1 {
		t.Fatalf("one match, got %d", len(opts))
	}
	tap(d, opts[0])
	if d.HasModal() || c.n != 1 || sb.Value() != "7" {
		t.Errorf("modal=%v changes=%d value=%q", d.HasModal(), c.n, sb.Value())
	}
}

func TestSearchComboBoxCanBeClosedWithoutChoosing(t *testing.T) {
	d, sb, _ := searchBox()
	sb.SetValue("3")
	c := &counter{}
	sb.OnChange = c.handler()
	tap(d, sb)
	d.HandleBack()
	if d.HasModal() || c.n != 0 || sb.Value() != "3" {
		t.Error("closing keeps the choice and says nothing")
	}
}

func TestSearchComboBoxTouchTargets(t *testing.T) {
	d, sb, _ := searchBox()
	tap(d, sb)
	d.Layout()
	for _, b := range optionButtons(d) {
		if b.Bounds.H < 56 {
			t.Errorf("%s is %v high", b.ID, b.Bounds.H)
		}
		if b.Bounds.Y+b.Bounds.H > d.H {
			t.Errorf("%s is off the screen", b.ID)
		}
		d.Layout()
		if got := d.TargetAt(b.Bounds.Center()); got != Widget(b) {
			t.Errorf("a touch at the center of %s reaches %v", b.ID, got)
		}
	}
}

func find(d *Document, id string) Widget {
	var found Widget
	d.Walk(func(w Widget) bool {
		if w.Elem().ID == id {
			found = w
		}
		return true
	})
	return found
}
