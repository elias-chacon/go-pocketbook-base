package widget

import (
	"time"

	"go-pocketbook-base/internal/kernel/geom"
)

// DefaultLongPress is how long a touch lasts to be a long press.
const DefaultLongPress = 2 * time.Second

// hit returns the deepest visible element containing p, or nil.
func hit(w Widget, p geom.Point) Widget {
	b := w.Elem()
	if !b.Visible() || !b.Bounds.Contains(p) {
		return nil
	}
	for i := len(b.children) - 1; i >= 0; i-- {
		if h := hit(b.children[i], p); h != nil {
			return h
		}
	}
	return w
}

func isInteractive(w Widget) bool {
	if _, ok := w.(PointerHandler); ok {
		return true
	}
	return w.Elem().OnClick != nil
}

// interactiveFrom walks up from w to the first element that reacts to touch.
// A disabled element blocks the touch (as in a browser).
func interactiveFrom(w Widget) Widget {
	for ; w != nil; w = w.Elem().parent {
		if w.Elem().Disabled {
			return nil
		}
		if isInteractive(w) {
			return w
		}
	}
	return nil
}

// TargetAt returns the element that would receive a touch at p (design pixels).
func (d *Document) TargetAt(p geom.Point) Widget {
	for i := len(d.popups) - 1; i >= 0; i-- {
		if h := hit(d.popups[i], p); h != nil {
			return interactiveFrom(h)
		}
	}
	if d.modal != nil {
		if h := hit(d.modal, p); h != nil {
			return interactiveFrom(h)
		}
		if d.modal.blocks() {
			return nil // a modal takes the touches outside it
		}
	}
	if d.Root == nil {
		return nil
	}
	return interactiveFrom(hit(d.Root, p))
}

// HandlePointer routes a touch given in device pixels: Down picks the target,
// Move/Up go to it (so a slider can be dragged), and a Click is synthesized when
// the finger is released over the element it started on.
func (d *Document) HandlePointer(t PointerType, deviceX, deviceY float64) {
	p := geom.Point{X: deviceX / d.Scale, Y: deviceY / d.Scale}
	switch t {
	case PointerDown:
		d.pointerDown(p)
	case PointerMove:
		d.handOverToScroll(p)
		d.deliver(d.capture, &PointerEvent{Type: PointerMove, X: p.X, Y: p.Y})
	case PointerUp:
		d.pointerUp(p)
	}
}

func (d *Document) pointerDown(p geom.Point) {
	d.swallow, d.scrolling, d.downAt = false, false, p
	if d.HasPopup() && !d.insidePopup(p) {
		d.ClosePopups()
		d.swallow = true // this touch only dismissed the popup
		d.capture = nil
		return
	}
	d.capture = d.TargetAt(p)
	d.pressedAt = d.now()
	d.deliver(d.capture, &PointerEvent{Type: PointerDown, X: p.X, Y: p.Y})
}

func (d *Document) pointerUp(p geom.Point) {
	target := d.capture
	d.capture = nil
	if target == nil || d.swallow {
		return
	}
	d.deliver(target, &PointerEvent{Type: PointerUp, X: p.X, Y: p.Y})
	if d.scrolling {
		return // the touch was a swipe: it clicks nothing
	}
	if !target.Elem().Bounds.Contains(p) {
		return
	}
	if d.now().Sub(d.pressedAt) >= d.longPressTime() && d.longPress(target, p) {
		return
	}
	d.click(target, p)
}

// swipeSlop is how far (design pixels) a finger must travel, mostly vertically, for a touch that
// began on a button, field or card inside a Scroll to become a swipe of that Scroll.
const swipeSlop = 24

// handOverToScroll turns a touch that began on an interactive element inside a Scroll into a swipe
// of the Scroll once the finger moved vertically past swipeSlop: otherwise only the bare parts of
// the window would scroll it. The element never receives the rest of the touch, and nothing is clicked.
func (d *Document) handOverToScroll(p geom.Point) {
	if d.scrolling || d.capture == nil {
		return
	}
	dy, dx := p.Y-d.downAt.Y, p.X-d.downAt.X
	if dy < 0 {
		dy = -dy
	}
	if dx < 0 {
		dx = -dx
	}
	if dy < swipeSlop || dy < dx {
		return
	}
	for w := d.capture; w != nil; w = w.Elem().parent {
		sc, ok := w.(*Scroll)
		if !ok {
			continue
		}
		if sc.MaxOffset() > 0 && w != d.capture {
			d.capture, d.scrolling = sc, true
			sc.HandlePointer(&PointerEvent{Type: PointerDown, X: d.downAt.X, Y: d.downAt.Y})
		}
		return
	}
}

func (d *Document) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
}

func (d *Document) longPressTime() time.Duration {
	if d.LongPress > 0 {
		return d.LongPress
	}
	return DefaultLongPress
}

// longPress bubbles OnLongPress up from target and reports whether an element had one.
func (d *Document) longPress(target Widget, p geom.Point) bool {
	ev := &Event{Target: target, X: p.X, Y: p.Y}
	handled := false
	for w := target; w != nil && !ev.stopped; w = w.Elem().parent {
		if h := w.Elem().OnLongPress; h != nil {
			handled = true
			h(ev)
			break
		}
	}
	if handled {
		d.Invalidate()
	}
	return handled
}

func (d *Document) insidePopup(p geom.Point) bool {
	for _, pop := range d.popups {
		if hit(pop, p) != nil {
			return true
		}
	}
	return false
}

func (d *Document) deliver(target Widget, e *PointerEvent) {
	if target == nil {
		return
	}
	if h, ok := target.(PointerHandler); ok {
		h.HandlePointer(e)
	}
}

// click delivers the Click to the widget, then bubbles OnClick up the tree.
func (d *Document) click(target Widget, p geom.Point) {
	d.deliver(target, &PointerEvent{Type: PointerClick, X: p.X, Y: p.Y})
	ev := &Event{Target: target, X: p.X, Y: p.Y}
	for w := target; w != nil && !ev.stopped; w = w.Elem().parent {
		if h := w.Elem().OnClick; h != nil {
			h(ev)
		}
	}
	d.Invalidate()
}
