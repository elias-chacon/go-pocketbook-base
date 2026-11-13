package widget

import (
	"math"
	"testing"

	"go-pocketbook-base/internal/geom"
)

// box is a fixed-size leaf used to observe the layout.
func box(w, h float64) *Panel {
	p := NewPanel()
	p.Style.Width, p.Style.Height = Px(w), Px(h)
	return p
}

func near(a, b float64) bool { return math.Abs(a-b) < 0.01 }

func assertRect(t *testing.T, name string, got, want geom.Rect) {
	t.Helper()
	if !near(got.X, want.X) || !near(got.Y, want.Y) || !near(got.W, want.W) || !near(got.H, want.H) {
		t.Errorf("%s = %+v, want %+v", name, got, want)
	}
}

func layout(root Widget, w, h float64) { layoutWidget(root, geom.Rect{W: w, H: h}) }

func TestBlockStacksAndStretches(t *testing.T) {
	a, b := box(0, 50), box(0, 70)
	a.Style.Width, b.Style.Width = Length{}, Length{}
	root := NewPanel(a, b)
	root.Style.Gap = 10
	layout(root, 400, 600)
	assertRect(t, "a", a.Bounds, geom.R(0, 0, 400, 50))
	assertRect(t, "b", b.Bounds, geom.R(0, 60, 400, 70))
}

func TestFlexRowGrowGapAndPadding(t *testing.T) {
	fixed, grow := box(100, 40), box(0, 40)
	grow.Style.Width = Length{}
	grow.Style.FlexGrow = 1
	root := NewRow(20, fixed, grow)
	root.Style.Padding = geom.All(10)
	root.Style.AlignItems = AlignStart
	layout(root, 500, 100)
	assertRect(t, "fixed", fixed.Bounds, geom.R(10, 10, 100, 40))
	assertRect(t, "grow", grow.Bounds, geom.R(130, 10, 360, 40)) // 500-20 padding-100-20 gap
}

func TestFlexJustifyAndAlign(t *testing.T) {
	cases := []struct {
		name    string
		justify Justify
		wantX   []float64
	}{
		{"center", JustifyCenter, []float64{100, 200}},
		{"end", JustifyEnd, []float64{200, 300}},
		{"space-between", JustifySpaceBetween, []float64{0, 300}},
		{"space-around", JustifySpaceAround, []float64{50, 250}},
	}
	for _, c := range cases {
		a, b := box(100, 40), box(100, 40)
		root := NewRow(0, a, b)
		root.Style.JustifyContent, root.Style.AlignItems = c.justify, AlignCenter
		layout(root, 400, 100)
		if !near(a.Bounds.X, c.wantX[0]) || !near(b.Bounds.X, c.wantX[1]) {
			t.Errorf("%s: x = %v, %v; want %v", c.name, a.Bounds.X, b.Bounds.X, c.wantX)
		}
		if !near(a.Bounds.Y, 30) { // centered in 100
			t.Errorf("%s: y = %v, want 30", c.name, a.Bounds.Y)
		}
	}
}

func TestFlexColumnStretchesWidthAndGrowsHeight(t *testing.T) {
	top, rest := box(0, 60), box(0, 0)
	top.Style.Width, rest.Style.Width, rest.Style.Height = Length{}, Length{}, Length{}
	rest.Style.FlexGrow = 1
	root := NewColumn(10, top, rest)
	layout(root, 300, 500)
	assertRect(t, "top", top.Bounds, geom.R(0, 0, 300, 60))
	assertRect(t, "rest", rest.Bounds, geom.R(0, 70, 300, 430))
}

func TestFlexWrap(t *testing.T) {
	items := []Widget{box(150, 40), box(150, 40), box(150, 40)}
	root := NewRow(10, items...)
	root.Style.FlexWrap, root.Style.AlignItems = true, AlignStart
	layout(root, 340, 300)
	assertRect(t, "1", items[0].Elem().Bounds, geom.R(0, 0, 150, 40))
	assertRect(t, "2", items[1].Elem().Bounds, geom.R(160, 0, 150, 40))
	assertRect(t, "3 wraps", items[2].Elem().Bounds, geom.R(0, 50, 150, 40))
}

func TestGridFractionsSpanAndRowHeights(t *testing.T) {
	a, b, wide := box(0, 40), box(0, 70), box(0, 30)
	for _, p := range []*Panel{a, b, wide} {
		p.Style.Width = Length{}
	}
	wide.Style.GridColumnSpan = 2
	root := NewGrid(10, []Track{PxTrack(100), FrTrack(1)}, a, b, wide)
	layout(root, 410, 400)
	assertRect(t, "a", a.Bounds, geom.R(0, 0, 100, 40))   // explicit height is kept
	assertRect(t, "b", b.Bounds, geom.R(110, 0, 300, 70)) // 410-100-10
	assertRect(t, "wide", wide.Bounds, geom.R(0, 80, 410, 30))
}

func TestHiddenElementsDoNotTakeSpace(t *testing.T) {
	a, hidden, c := box(0, 30), box(0, 99), box(0, 30)
	for _, p := range []*Panel{a, hidden, c} {
		p.Style.Width = Length{}
	}
	hidden.Hidden = true
	root := NewPanel(a, hidden, c)
	layout(root, 200, 200)
	if !near(c.Bounds.Y, 30) {
		t.Errorf("c.Y = %v, want 30 (hidden element must be skipped)", c.Bounds.Y)
	}
}

func TestLabelWrapsAndAutoHeightPanelWrapsContent(t *testing.T) {
	l := NewLabel("alpha beta gamma delta epsilon zeta eta theta")
	card := NewCard("", "", l)
	one := measure(card, 2000)
	narrow := measure(card, 300)
	if narrow.H <= one.H {
		t.Errorf("narrow card height %v must exceed single-line height %v", narrow.H, one.H)
	}
	if narrow.W > 300+0.01 {
		t.Errorf("narrow card width %v exceeds the limit", narrow.W)
	}
}
