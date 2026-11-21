package widget

import (
	"fmt"
	"testing"
)

// manyOptions is a list of 60 options: "Option 01 ... Option 60", one of them with accents.
func manyOptions() []Option {
	var out []Option
	for i := 1; i <= 60; i++ {
		out = append(out, Option{Label: fmt.Sprintf("Option %02d", i), Value: fmt.Sprint(i)})
	}
	out[41].Label = "Português (Brasil)"
	return out
}

func longBox() (*Document, *ComboBox, *FakeKeyboard) {
	cb := NewComboBox("Pick one", manyOptions()...)
	cb.ID = "cb"
	d := testDoc(cb)
	kb := &FakeKeyboard{}
	d.Keyboard = kb
	return d, cb, kb
}

// openPage is the popup of a long list and the rows it shows now.
func openPage(t *testing.T, d *Document) (*comboPopup, *menuList) {
	t.Helper()
	if len(d.popups) != 1 {
		t.Fatalf("popups = %d", len(d.popups))
	}
	p, ok := d.popups[0].(*comboPopup)
	if !ok {
		t.Fatalf("the popup is %T, not the paged list", d.popups[0])
	}
	list, _ := p.body.children[0].(*menuList)
	return p, list
}

func TestAShortComboBoxStaysTheSimpleMenu(t *testing.T) {
	cb := NewComboBox("x", Option{"A", "a"}, Option{"B", "b"})
	d := testDoc(cb)
	tap(d, cb)
	if _, paged := d.popups[0].(*comboPopup); paged || !d.HasPopup() {
		t.Error("a list that fits is the plain menu")
	}
}

func TestALongComboBoxOpensAPagedPopupWithSearchNotAModal(t *testing.T) {
	d, cb, _ := longBox()
	tap(d, cb)
	if d.HasModal() {
		t.Fatal("it is a popup under the field, not a window")
	}
	p, list := openPage(t, d)
	if len(list.items) != p.rows || p.rows < 3 || p.pages() < 2 {
		t.Errorf("rows %d, items %d, pages %d", p.rows, len(list.items), p.pages())
	}
	if b := p.Bounds; b.Bottom() > d.H || b.Y < cb.Bounds.Bottom() {
		t.Errorf("the popup must fit under the field: %+v", b)
	}
	if p.search == nil || p.indicator.Text != fmt.Sprintf("1 / %d", p.pages()) {
		t.Errorf("search field and indicator: %v %q", p.search, p.indicator.Text)
	}
}

func TestTheLongListOpensOnThePageOfTheChosenOptionAndHighlightsIt(t *testing.T) {
	d, cb, _ := longBox()
	cb.SetValue("55")
	tap(d, cb)
	p, list := openPage(t, d)
	if p.page == 0 || list.selected < 0 || list.items[list.selected].Value != "55" {
		t.Errorf("page %d, selected %d", p.page, list.selected)
	}
}

func TestThePageButtonsTurnPagesAndStopAtTheEnds(t *testing.T) {
	d, cb, _ := longBox()
	tap(d, cb)
	p, _ := openPage(t, d)
	if !p.prev.Disabled || p.next.Disabled {
		t.Fatal("the first page has no previous")
	}
	tap(d, p.next)
	if p.page != 1 || p.prev.Disabled {
		t.Fatalf("page %d", p.page)
	}
	for i := 0; i < p.pages()+2; i++ {
		tap(d, p.next)
	}
	if p.page != p.pages()-1 || !p.next.Disabled {
		t.Errorf("the last page: %d of %d", p.page, p.pages())
	}
	if !d.HasPopup() {
		t.Error("turning pages must not close the popup")
	}
}

func TestSearchFiltersWithoutCaseOrAccentsAndTheSizeStaysTheSame(t *testing.T) {
	d, cb, kb := longBox()
	tap(d, cb)
	p, _ := openPage(t, d)
	before := p.Bounds
	kb.Reply = "PORTUGUES"
	tap(d, p.search)
	p, list := openPage(t, d)
	if len(list.items) != 1 || list.items[0].Label != "Português (Brasil)" {
		t.Fatalf("filtered rows: %+v", list.items)
	}
	if p.Bounds != before {
		t.Errorf("filtering must not move or resize the popup: %+v -> %+v", before, p.Bounds)
	}
	kb.Reply = "zzz"
	tap(d, p.search)
	if p, _ = openPage(t, d); len(p.matches) != 0 || p.indicator.Text != "1 / 1" {
		t.Errorf("nothing matches: %v %q", p.matches, p.indicator.Text)
	}
}

func TestChoosingARowSelectsItClosesThePopupAndTellsTheHandler(t *testing.T) {
	d, cb, kb := longBox()
	c := &counter{}
	cb.OnChange = c.handler()
	tap(d, cb)
	p, _ := openPage(t, d)
	kb.Reply = "option 07"
	tap(d, p.search)
	_, list := openPage(t, d)
	tapAt(d, list.rowRect(0).Center())
	if d.HasPopup() || c.n != 1 || cb.Value() != "7" {
		t.Errorf("popup=%v changes=%d value=%q", d.HasPopup(), c.n, cb.Value())
	}
}

func TestATouchOutsideThePagedPopupClosesItWithoutChoosing(t *testing.T) {
	d, cb, _ := longBox()
	c := &counter{}
	cb.OnChange = c.handler()
	tap(d, cb)
	tapAt(d, geomPt(5, d.H-5))
	if d.HasPopup() || c.n != 0 || cb.Index != -1 {
		t.Errorf("popup=%v changes=%d index=%d", d.HasPopup(), c.n, cb.Index)
	}
}

func TestTheLongListWorksWhenTheFieldIsLowOnTheScreen(t *testing.T) {
	cb := NewComboBox("x", manyOptions()...)
	d := NewDocument(1404, 1872)
	spacer := NewPanel()
	spacer.Style.Height = Px(1400)
	d.SetRoot(NewColumn(0, spacer, cb))
	d.Layout()
	tap(d, cb)
	p, _ := openPage(t, d)
	if p.Bounds.Y < 0 || p.Bounds.Bottom() > d.H {
		t.Errorf("the popup must stay on the screen: %+v", p.Bounds)
	}
}

func TestEverythingOfThePagedPopupStaysInsideItsPanel(t *testing.T) {
	d, cb, _ := longBox()
	tap(d, cb)
	p, _ := openPage(t, d)
	for _, w := range []Widget{p.search, p.body, p.prev, p.next, p.indicator} {
		b := w.Elem().Bounds
		if b.X < p.Bounds.X || b.Y < p.Bounds.Y || b.Right() > p.Bounds.Right() || b.Bottom() > p.Bounds.Bottom() {
			t.Errorf("%q %+v leaves the panel %+v", w.Elem().ID, b, p.Bounds)
		}
	}
	if b := p.next.Bounds; b.Bottom() > p.Bounds.Bottom()-BorderThin-6 {
		t.Errorf("the buttons must clear the padding and the border: %+v in %+v", b, p.Bounds)
	}
}
