package widget

import (
	"testing"
	"time"
)

func TestCheckboxAndSwitchToggle(t *testing.T) {
	for _, w := range []*Toggle{NewCheckbox("a", false), NewSwitch("b", false)} {
		changes := &counter{}
		w.OnChange = changes.handler()
		d := testDoc(w)
		tap(d, w)
		if !w.Checked || changes.n != 1 {
			t.Errorf("%q: Checked=%v changes=%d", w.Label, w.Checked, changes.n)
		}
		tap(d, w)
		if w.Checked {
			t.Errorf("%q: second tap must uncheck", w.Label)
		}
	}
}

func TestRadioGroupIsExclusive(t *testing.T) {
	a, b := NewRadio("A", "g", "a"), NewRadio("B", "g", "b")
	other := NewRadio("X", "other", "x")
	other.Checked = true
	col := NewColumn(0, a, b, other)
	d := NewDocument(1404, 1872)
	d.SetRoot(col)
	d.Layout()

	tap(d, a)
	tap(d, b)
	if a.Checked || !b.Checked {
		t.Errorf("a=%v b=%v, want only b", a.Checked, b.Checked)
	}
	if !other.Checked {
		t.Error("another group must be untouched")
	}
	changes := &counter{}
	b.OnChange = changes.handler()
	tap(d, b) // tapping the checked radio does nothing
	if !b.Checked || changes.n != 0 {
		t.Errorf("checked radio: Checked=%v changes=%d", b.Checked, changes.n)
	}
}

func TestSliderTapDragAndSnap(t *testing.T) {
	s := NewSlider(0, 100, 10, 0)
	s.Style.Width = Px(1000)
	inputs, changes := &counter{}, &counter{}
	s.OnInput, s.OnChange = inputs.handler(), changes.handler()
	d := testDoc(s)

	x0, x1, y := s.track()
	d.HandlePointer(PointerDown, x0+(x1-x0)*0.74, y)
	if s.Value != 70 { // 74% snaps to the 10 step
		t.Errorf("Value after down = %v, want 70", s.Value)
	}
	d.HandlePointer(PointerMove, x0+(x1-x0)*0.31, y)
	if s.Value != 30 || changes.n != 0 {
		t.Errorf("drag: Value=%v changes=%d (OnChange only on release)", s.Value, changes.n)
	}
	d.HandlePointer(PointerUp, x1+500, y) // released far outside: clamps to the end
	if s.Value != 100 || changes.n != 1 || inputs.n != 2 {
		t.Errorf("release: Value=%v changes=%d inputs=%d", s.Value, changes.n, inputs.n)
	}
}

func TestComboBoxPicksFromPopup(t *testing.T) {
	cb := NewComboBox("Cidade", Option{"A", "a"}, Option{"B", "b"}, Option{"C", "c"})
	changes := &counter{}
	cb.OnChange = changes.handler()
	d := testDoc(cb)

	tap(d, cb)
	if !d.HasPopup() || !d.PopupOpenFor(cb) {
		t.Fatal("popup should be open")
	}
	list := d.popups[0].(*menuList)
	tapAt(d, list.rowRect(1).Center())
	if cb.Value() != "b" || changes.n != 1 || d.HasPopup() {
		t.Errorf("Value=%q changes=%d popup=%v", cb.Value(), changes.n, d.HasPopup())
	}
	cb.SetValue("c")
	if cb.Index != 2 {
		t.Errorf("SetValue: Index=%d", cb.Index)
	}
}

func TestTouchOutsidePopupOnlyClosesIt(t *testing.T) {
	cb := NewComboBox("x", Option{"A", "a"})
	other := NewButton("other")
	clicks := &counter{}
	other.OnClick = clicks.handler()
	col := NewColumn(40, cb, other)
	col.Style.AlignItems = AlignStart
	d := NewDocument(1404, 1872)
	d.SetRoot(col)
	d.Layout()

	tap(d, cb)
	tap(d, other) // outside the popup: closes it and must NOT click the button
	if d.HasPopup() || clicks.n != 0 {
		t.Errorf("popup=%v clicks=%d", d.HasPopup(), clicks.n)
	}
	tap(d, other)
	if clicks.n != 1 {
		t.Errorf("after the popup is gone the button works, clicks=%d", clicks.n)
	}
}

func TestDropdownAndDisabledItem(t *testing.T) {
	done, blocked := &counter{}, &counter{}
	dd := NewDropdown("Ações",
		MenuItem{Label: "Go", OnSelect: func() { done.n++ }},
		MenuItem{Separator: true},
		MenuItem{Label: "No", Disabled: true, OnSelect: func() { blocked.n++ }})
	d := testDoc(dd)

	tap(d, dd)
	list := d.popups[0].(*menuList)
	tapAt(d, list.rowRect(2).Center()) // disabled: nothing, popup stays
	tapAt(d, list.rowRect(1).Center()) // separator: nothing
	if blocked.n != 0 || !d.HasPopup() {
		t.Fatalf("disabled/separator rows must not act (blocked=%d popup=%v)", blocked.n, d.HasPopup())
	}
	tapAt(d, list.rowRect(0).Center())
	if done.n != 1 || dd.Chosen.Label != "Go" || d.HasPopup() {
		t.Errorf("done=%d chosen=%q popup=%v", done.n, dd.Chosen.Label, d.HasPopup())
	}
}

func TestMenuBar(t *testing.T) {
	direct, sub := &counter{}, &counter{}
	bar := NewMenuBar(
		&Menu{Label: "Page", OnSelect: func() { direct.n++ }},
		&Menu{Label: "More", Items: []MenuItem{{Label: "Exit", OnSelect: func() { sub.n++ }}}})
	d := testDoc(bar)
	rects := bar.menuRects()

	tapAt(d, rects[0].Center())
	if direct.n != 1 || d.HasPopup() {
		t.Fatalf("direct menu: n=%d popup=%v", direct.n, d.HasPopup())
	}
	tapAt(d, rects[1].Center())
	if !d.HasPopup() {
		t.Fatal("menu with items must open a popup")
	}
	tapAt(d, d.popups[0].(*menuList).rowRect(0).Center())
	if sub.n != 1 {
		t.Errorf("sub item not run: %d", sub.n)
	}
}

func TestChips(t *testing.T) {
	a, b := NewChip("A"), NewChip("B")
	g := NewChipGroup(true, a, b)
	d := NewDocument(1404, 1872)
	d.SetRoot(g)
	d.Layout()

	tap(d, a)
	tap(d, b)
	if a.Selected || !b.Selected {
		t.Errorf("exclusive: a=%v b=%v", a.Selected, b.Selected)
	}
	tap(d, b) // an exclusive group keeps one selected
	if !b.Selected {
		t.Error("the last chip cannot be deselected in an exclusive group")
	}

	multi := NewChipGroup(false, NewChip("X"), NewChip("Y"))
	d2 := NewDocument(1404, 1872)
	d2.SetRoot(multi)
	d2.Layout()
	tap(d2, multi.children[0])
	tap(d2, multi.children[1])
	if got := multi.SelectedLabels(); len(got) != 2 {
		t.Errorf("multi selected = %v", got)
	}
}

func TestClosableChipRemovesItself(t *testing.T) {
	c := NewChip("tag")
	c.Closable = true
	keep := NewChip("keep")
	g := NewChipGroup(false, c, keep)
	d := NewDocument(1404, 1872)
	d.SetRoot(g)
	d.Layout()

	tapAt(d, c.closeRect().Center())
	if len(g.children) != 1 || g.children[0] != Widget(keep) {
		t.Errorf("children after close = %d", len(g.children))
	}
}

func TestTableSortPageSelect(t *testing.T) {
	tb := NewTable(2, TableColumn{Title: "N", Sortable: true}, TableColumn{Title: "T"})
	tb.Rows = [][]string{{"10", "ten"}, {"9", "nine"}, {"100", "hundred"}}
	picked := &counter{}
	tb.OnSelect = picked.handler()
	d := testDoc(tb)

	tapAt(d, tb.colRects(tb.headerRect())[0].Center()) // sort by N ascending (numeric, not text)
	if got := tb.order(); got[0] != 1 || got[1] != 0 || got[2] != 2 {
		t.Errorf("asc order = %v, want [1 0 2]", got)
	}
	tapAt(d, tb.colRects(tb.headerRect())[0].Center()) // again: descending
	if got := tb.order(); got[0] != 2 {
		t.Errorf("desc order = %v", got)
	}
	tapAt(d, tb.colRects(tb.headerRect())[1].Center()) // not sortable: unchanged
	if tb.SortCol != 0 || !tb.SortDesc {
		t.Error("a non-sortable header must not change the sort")
	}

	tapAt(d, tb.rowRect(0).Center()) // first visible row = original index 2
	if tb.Selected != 2 || picked.n != 1 {
		t.Errorf("Selected=%d picked=%d", tb.Selected, picked.n)
	}

	_, next := tb.pagerRects()
	tapAt(d, next.Center())
	if tb.Page != 1 || tb.PageCount() != 2 {
		t.Errorf("Page=%d count=%d", tb.Page, tb.PageCount())
	}
	tapAt(d, next.Center()) // already on the last page
	if tb.Page != 1 {
		t.Errorf("Page ran past the end: %d", tb.Page)
	}
	if from, to := tb.pageRange(); from != 3 || to != 3 {
		t.Errorf("pageRange = %d-%d, want 3-3", from, to)
	}
	tapAt(d, tb.rowRect(1).Center()) // empty slot on the last page
	if tb.Selected != 2 {
		t.Errorf("an empty row slot must not select anything: %d", tb.Selected)
	}
}

func TestCalendarGridAndSelection(t *testing.T) {
	// 1 November 2026 is a Sunday.
	grid := monthGrid(2026, time.November, time.Sunday)
	if !sameDay(grid[0], time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("first cell = %v", grid[0])
	}
	if grid := monthGrid(2026, time.November, time.Monday); !sameDay(grid[0], time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("Monday-first grid starts at %v", grid[0])
	}

	cal := NewCalendar(time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC))
	changes := &counter{}
	cal.OnChange = changes.handler()
	d := testDoc(cal)

	tapAt(d, cal.cellRect(10).Center()) // Nov 11
	if cal.Selected.Day() != 11 || changes.n != 1 {
		t.Errorf("Selected=%v changes=%d", cal.Selected, changes.n)
	}
	tapAt(d, cal.cellRect(40).Center()) // trailing day of December: switches month
	if cal.Month != time.December || cal.Selected.Month() != time.December {
		t.Errorf("other-month tap: %v %v", cal.Month, cal.Selected)
	}
	_, next := cal.navRects()
	for i := 0; i < 2; i++ {
		tapAt(d, next.Center())
	}
	if cal.Year != 2027 || cal.Month != time.February {
		t.Errorf("next x2 from December = %d-%v", cal.Year, cal.Month)
	}
}

// A slide button with Step 50 on 0..100 has three resting positions:
// left, center and right (O====== / ===O=== / ======O).
func TestSlideButtonSnapsToLeftCenterRight(t *testing.T) {
	s := NewSlider(0, 100, 50, 50)
	s.Style.Width = Px(600)
	d := testDoc(s)
	if s.Value != 50 {
		t.Fatalf("starts centered, Value=%v", s.Value)
	}
	x0, x1, y := s.track()
	for _, c := range []struct{ at, want float64 }{
		{0.05, 0}, {0.2, 0}, {0.4, 50}, {0.6, 50}, {0.8, 100}, {1, 100},
	} {
		tapAt(d, geomPt(x0+(x1-x0)*c.at, y))
		if s.Value != c.want {
			t.Errorf("tap at %.0f%% -> %v, want %v", c.at*100, s.Value, c.want)
		}
	}
	// the knob is drawn at the matching end of the bar
	s.SetValue(0)
	if s.fraction() != 0 {
		t.Errorf("left fraction = %v", s.fraction())
	}
	s.SetValue(50)
	if s.fraction() != 0.5 {
		t.Errorf("center fraction = %v", s.fraction())
	}
}
