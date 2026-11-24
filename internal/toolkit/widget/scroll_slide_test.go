package widget

import (
	"fmt"
	"image"
	"testing"

	"go-pocketbook-base/internal/kernel/geom"
)

// slideFixture is a page: a heading, a Scroll of n rows and a footer button.
func slideFixture(n int) (*Document, *Scroll, *Label) {
	head := NewLabel("heading")
	var rows []Widget
	for i := 1; i <= n; i++ {
		l := NewLabel(fmt.Sprintf("Row %03d", i))
		l.Style.MinHeight = 90
		if i%2 == 0 {
			l.Style.Background = Soft
		}
		rows = append(rows, l)
	}
	s := NewScroll(rows...)
	s.Style.FlexGrow = 1
	d := NewDocument(1404, 1872)
	d.SetRoot(NewColumn(10, head, s, NewButton("Footer")))
	return d, s, head
}

// sameAsFullRender fails when the frame is not what a full render of the document shows.
func sameAsFullRender(t *testing.T, d *Document, f Frame, what string) {
	t.Helper()
	want := d.Render()
	bad := 0
	for i := range f.Image.Pix {
		if diff := int(f.Image.Pix[i]) - int(want.Pix[i]); diff > 8 || diff < -8 {
			bad++
		}
	}
	if bad > 0 {
		t.Fatalf("%s: %d pixels differ from a full render", what, bad)
	}
}

func TestScrollingSlidesThePixelsAndPaintsOnlyTheStrip(t *testing.T) {
	d, s, _ := slideFixture(80)
	d.Update()
	for i, dy := range []float64{160, 160, 90, -90, 1, -300, 4000} {
		s.ScrollBy(dy)
		f := d.Update()
		st := d.LastUpdate()
		if i < 6 { // the last one is a jump past the window: it may repaint everything
			if f.Full || !st.Slid {
				t.Fatalf("step %d: a scroll must slide, not repaint all (full=%v slid=%v)", i, f.Full, st.Slid)
			}
			view := s.viewport()
			if limit := int(0.5 * view.W * view.H); st.Painted > limit {
				t.Errorf("step %d: %d pixels repainted for a move of %v, more than half the window (%d)", i, st.Painted, dy, limit)
			}
		}
		sameAsFullRender(t, d, f, fmt.Sprintf("step %d (offset %v)", i, s.Offset))
	}
}

func TestScrollingTellsThePanelTheWholeWindowMoved(t *testing.T) {
	d, s, _ := slideFixture(80)
	d.Update()
	s.ScrollBy(160)
	f := d.Update()
	v := d.windowRect(s.viewport(), true)
	covered := false
	for _, r := range f.Rects {
		if v.In(r) {
			covered = true
		}
	}
	if !covered {
		t.Errorf("the panel must be told about the whole window %v: %v", v, f.Rects)
	}
	if st := d.LastUpdate(); st.Sent < v.Dx()*v.Dy() {
		t.Errorf("sent %d pixels, the window has %d", st.Sent, v.Dx()*v.Dy())
	}
}

func TestScrollingDoesNotTouchTheRowsThatDidNotChange(t *testing.T) {
	d, s, _ := slideFixture(80)
	d.Update()
	s.ScrollBy(2000) // far: the strip is the whole window, still not a full repaint of the page
	f := d.Update()
	sameAsFullRender(t, d, f, "a jump")
}

func TestScrollingAndEditingAtTheSameTimeRepaintsTheEdit(t *testing.T) {
	d, s, head := slideFixture(80)
	d.Update()
	row := s.Content.children[3].(*Label) // visible after the move below
	s.ScrollBy(100)
	row.Text = "an edited row with other words"
	head.Text = "a new heading"
	f := d.Update()
	sameAsFullRender(t, d, f, "scroll + edits")
}

func TestScrollingAfterTheContentShrankStaysRight(t *testing.T) {
	d, s, _ := slideFixture(80)
	d.Update()
	s.ScrollBy(3000)
	d.Update()
	s.Content.children = s.Content.children[:20] // the list got short: the offset must come back
	f := d.Update()
	sameAsFullRender(t, d, f, "a shorter list")
	s.ScrollBy(-500)
	sameAsFullRender(t, d, d.Update(), "back up")
}

func TestScrollUnderAPopupRepaintsInsteadOfSliding(t *testing.T) {
	d, s, head := slideFixture(80)
	d.Update()
	d.OpenPopup(head, NewLabel("floating"), geom.R(200, 300, 400, 200), geom.Size{W: 400, H: 200})
	d.Update()
	s.ScrollBy(160)
	f := d.Update()
	if d.LastUpdate().Slid {
		t.Error("with a popup open the pixels must not slide: the popup would move with them")
	}
	sameAsFullRender(t, d, f, "scroll under a popup")
}

func TestSwipeScrollingKeepsSlidingAndStaysEqualToAFullRender(t *testing.T) {
	d, s, _ := slideFixture(80)
	d.Update()
	v := s.viewport()
	x := int(v.X + v.W/2)
	d.HandlePointer(PointerDown, float64(x), v.Y+700)
	for _, y := range []float64{640, 560, 470, 400} {
		d.HandlePointer(PointerMove, float64(x), v.Y+y)
		f := d.Update()
		if f.Full {
			t.Fatalf("a drag step repainted the whole screen at y=%v", y)
		}
		sameAsFullRender(t, d, f, fmt.Sprintf("drag to %v", y))
	}
	d.HandlePointer(PointerUp, float64(x), v.Y+400)
	if s.Offset == 0 {
		t.Fatal("the swipe must have scrolled")
	}
}

func TestAnUnchangedScrollUpdatesNothing(t *testing.T) {
	d, _, _ := slideFixture(80)
	d.Update()
	if f := d.Update(); len(f.Rects) != 0 || f.Full {
		t.Errorf("nothing changed, yet rects=%v full=%v", f.Rects, f.Full)
	}
}

func TestRingIsTheBorderBetweenTwoRectangles(t *testing.T) {
	outer, inner := image.Rect(0, 0, 10, 10), image.Rect(2, 3, 8, 9)
	area := 0
	for _, r := range ring(outer, inner) {
		area += r.Dx() * r.Dy()
	}
	if want := 100 - 36; area != want {
		t.Errorf("ring area = %d, want %d", area, want)
	}
	if got := ring(inner, inner); len(got) != 0 {
		t.Errorf("equal rectangles have no ring: %v", got)
	}
}
