package widget

import "testing"

// blocks is a column of n black boxes 200 high and a button in the last one.
func scrollFixture(n int) (*Document, *Scroll, *Button) {
	var items []Widget
	for i := 0; i < n; i++ {
		b := NewPanel()
		b.Style.Height, b.Style.Background = Px(200), Ink
		items = append(items, b)
	}
	btn := NewButton("Last")
	items = append(items, btn)
	s := NewScroll(items...)
	s.Style.Height = Px(600)
	footer := NewLabel("footer")
	footer.ID = "footer"
	root := NewColumn(0, s, footer)
	root.Style.AlignItems = AlignStretch
	d := NewDocument(1404, 1872)
	d.SetRoot(root)
	d.Layout()
	return d, s, btn
}

func TestScrollShowsAWindowOntoTallerContentAndMovesIt(t *testing.T) {
	d, s, btn := scrollFixture(6)
	if s.Bounds.H != 600 || s.MaxOffset() <= 0 {
		t.Fatalf("window %v, max offset %v", s.Bounds.H, s.MaxOffset())
	}
	first := s.Content.Elem().children[0].Elem().Bounds.Y
	s.ScrollTo(300)
	d.Layout()
	if got := s.Content.Elem().children[0].Elem().Bounds.Y; got != first-300 {
		t.Errorf("content moved to %v, want %v", got, first-300)
	}
	s.ScrollTo(1e9)
	d.Layout()
	if s.Offset != s.MaxOffset() || btn.Bounds.Bottom() > s.Bounds.Bottom()+0.5 {
		t.Errorf("clamped to the end: offset %v of %v, button bottom %v", s.Offset, s.MaxOffset(), btn.Bounds.Bottom())
	}
	s.ScrollTo(-5)
	if s.Offset != 0 {
		t.Error("never above the top")
	}
}

func TestScrollContentThatFitsHasNoBarAndNoOffset(t *testing.T) {
	_, s, _ := scrollFixture(1)
	s.Style.Height = Px(1000)
	s.Document().Layout()
	if s.MaxOffset() != 0 {
		t.Errorf("max offset %v", s.MaxOffset())
	}
	s.ScrollBy(100)
	if s.Offset != 0 {
		t.Error("nothing to scroll")
	}
}

func TestScrollSwipeOnAnInertPartMovesTheContentAndAButtonStillClicks(t *testing.T) {
	d, s, btn := scrollFixture(6)
	from := geomPt(100, 300)
	d.HandlePointer(PointerDown, from.X, from.Y)
	d.HandlePointer(PointerMove, from.X, from.Y-250)
	d.HandlePointer(PointerUp, from.X, from.Y-250)
	d.Layout()
	if s.Offset != 250 {
		t.Fatalf("a swipe of 250 up moves the content 250: %v", s.Offset)
	}
	clicks := counter{}
	btn.OnClick = clicks.handler()
	s.ScrollTo(s.MaxOffset())
	d.Layout()
	tap(d, btn)
	if clicks.n != 1 {
		t.Error("a button inside the window still takes its tap")
	}
}

func TestScrollSwipeThatStartsOnAButtonScrollsAndDoesNotClickIt(t *testing.T) {
	d, s, btn := scrollFixture(6)
	clicks := counter{}
	btn.OnClick = clicks.handler()
	s.ScrollTo(s.MaxOffset())
	d.Layout()
	at := btn.Bounds.Center()
	d.HandlePointer(PointerDown, at.X, at.Y)
	d.HandlePointer(PointerMove, at.X, at.Y+10) // a tremor is not a swipe
	d.HandlePointer(PointerMove, at.X, at.Y+200)
	d.HandlePointer(PointerUp, at.X, at.Y+200)
	d.Layout()
	if want := s.MaxOffset() - 200; s.Offset != want {
		t.Errorf("a swipe down of 200 from a button: offset %v, want %v", s.Offset, want)
	}
	if clicks.n != 0 {
		t.Error("a swipe must not click the button it began on")
	}
	s.ScrollTo(s.MaxOffset()) // the button is back in the window
	d.Layout()
	tap(d, btn)
	if clicks.n != 1 {
		t.Error("a plain tap still clicks")
	}
}

func TestScrollElementsOutsideTheWindowAreNotTouchedNorDrawn(t *testing.T) {
	d, s, btn := scrollFixture(6)
	if got := d.TargetAt(btn.Bounds.Center()); got == Widget(btn) {
		t.Error("a button below the window must not get the touch")
	}
	if InView(btn) {
		t.Error("InView says it is out of sight")
	}
	img := d.Render()
	// Between the bottom of the window and the footer the screen must stay white: the black boxes
	// that overflow the window are cut.
	y := int(s.Bounds.Bottom()) + 4
	for x := 20; x < 1300; x += 97 {
		if img.GrayAt(x, y).Y != 255 {
			t.Fatalf("content leaks out of the window at %d,%d", x, y)
		}
	}
	if img.GrayAt(100, 100).Y == 255 {
		t.Error("the box inside the window is drawn")
	}
}

func TestScrollBarThumbDragsAndTrackPages(t *testing.T) {
	d, s, _ := scrollFixture(6)
	track := s.bar()
	th := s.thumb()
	d.HandlePointer(PointerDown, th.Center().X, th.Center().Y)
	d.HandlePointer(PointerMove, th.Center().X, th.Center().Y+(track.H-th.H))
	d.HandlePointer(PointerUp, th.Center().X, th.Center().Y+(track.H-th.H))
	d.Layout()
	if s.MaxOffset()-s.Offset > 0.5 {
		t.Errorf("dragging the thumb to the end shows the end: %v of %v", s.Offset, s.MaxOffset())
	}
	s.ScrollTo(0)
	d.Layout()
	tapAt(d, geomPt(track.X+track.W/2, track.Bottom()-5))
	if s.Offset != s.PageSize() {
		t.Errorf("a tap under the thumb pages down: %v want %v", s.Offset, s.PageSize())
	}
	tapAt(d, geomPt(track.X+track.W/2, track.Y+3))
	if s.Offset != 0 {
		t.Errorf("a tap above the thumb pages up: %v", s.Offset)
	}
}

func TestScrollIntoViewBringsAnElementInAndOnScrollFires(t *testing.T) {
	d, s, btn := scrollFixture(6)
	fired := counter{}
	s.OnScroll = fired.handler()
	s.ScrollIntoView(btn)
	d.Layout()
	if btn.Bounds.Bottom() > s.Bounds.Bottom()+0.5 || btn.Bounds.Y < s.Bounds.Y || fired.n == 0 {
		t.Errorf("button %+v window %+v fired %d", btn.Bounds, s.Bounds, fired.n)
	}
}

func TestScrollKeepsItsOffsetInRangeWhenTheContentShrinks(t *testing.T) {
	d, s, _ := scrollFixture(6)
	s.ScrollTo(s.MaxOffset())
	d.Layout()
	for len(s.Content.Elem().children) > 2 {
		s.Content.RemoveChild(s.Content.Elem().children[0])
	}
	d.Layout()
	if s.Offset > s.MaxOffset() {
		t.Errorf("offset %v beyond %v", s.Offset, s.MaxOffset())
	}
}
