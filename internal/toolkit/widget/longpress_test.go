package widget

import (
	"testing"
	"time"
)

// press touches the center of w and lets held pass before the finger is lifted.
func press(d *Document, clock *time.Time, w Widget, held time.Duration) {
	p := w.Elem().Bounds.Center()
	d.HandlePointer(PointerDown, p.X*d.Scale, p.Y*d.Scale)
	*clock = clock.Add(held)
	d.HandlePointer(PointerUp, p.X*d.Scale, p.Y*d.Scale)
}

func longPressDoc() (*Document, *Button, *time.Time) {
	b := NewButton("x")
	d := testDoc(b)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	d.Now = func() time.Time { return now }
	return d, b, &now
}

func TestATouchOfTwoSecondsIsALongPressNotAClick(t *testing.T) {
	d, b, clock := longPressDoc()
	clicks, longs := &counter{}, &counter{}
	b.OnClick, b.OnLongPress = clicks.handler(), longs.handler()
	press(d, clock, b, 2*time.Second)
	if longs.n != 1 || clicks.n != 0 {
		t.Errorf("long=%d click=%d", longs.n, clicks.n)
	}
	press(d, clock, b, 300*time.Millisecond)
	if longs.n != 1 || clicks.n != 1 {
		t.Errorf("a short touch is a click: long=%d click=%d", longs.n, clicks.n)
	}
}

func TestALongTouchOnAnElementWithoutALongPressIsStillAClick(t *testing.T) {
	d, b, clock := longPressDoc()
	clicks := &counter{}
	b.OnClick = clicks.handler()
	press(d, clock, b, 5*time.Second)
	if clicks.n != 1 {
		t.Errorf("click=%d", clicks.n)
	}
}

func TestTheLongPressBubblesToTheParentAndTheTimeCanBeChanged(t *testing.T) {
	inner := NewButton("inner")
	cell := NewColumn(0, inner)
	longs := &counter{}
	cell.OnLongPress = longs.handler()
	d := testDoc(cell)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	d.Now = func() time.Time { return now }
	d.LongPress = 500 * time.Millisecond
	press(d, &now, inner, 600*time.Millisecond)
	if longs.n != 1 {
		t.Errorf("long=%d", longs.n)
	}
}
