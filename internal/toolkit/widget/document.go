package widget

import (
	"image"
	"time"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
)

// RefWidth is the design width in pixels (a PB741 screen). Everything is
// described in these units and scaled to the real screen.
const RefWidth = 1404.0

// Document owns an element tree: it lays it out, paints it and routes touch.
type Document struct {
	W, H             float64 // design size
	Scale            float64 // device pixels per design pixel
	DeviceW, DeviceH int
	Root             Widget
	Keyboard         Keyboard
	// Invert shows the screen in negative (light text on a dark page), for dim light
	// or a reader who finds it easier. Change it with SetInvert, which repaints it all.
	Invert bool
	// OnInvalidate is called when the screen needs repainting (the device
	// hooks it to InkView's Repaint).
	OnInvalidate func()
	// LongPress is how long a touch must last to count as a long press (DefaultLongPress when 0).
	LongPress time.Duration
	// Now is the clock of the touches (time.Now when nil; tests move it).
	Now func() time.Time

	modal      *Modal // the open modal, below the popups
	popups     []Widget
	popupOwner Widget // element that opened the popup (its chevron shows "open")
	focus      Widget
	capture    Widget
	swallow    bool        // the current touch only dismissed a popup
	pressedAt  time.Time   // when the current touch went down
	canvas     *gfx.Canvas // reused between frames: a screen-size RGBA is 10 MB

	// partial repaint (see Update)
	frame     *image.Gray
	snap      map[Widget]snap
	forceFull bool
}

// NewDocument creates a document for a deviceW x deviceH screen.
func NewDocument(deviceW, deviceH int) *Document {
	scale := float64(deviceW) / RefWidth
	return &Document{
		W: RefWidth, H: float64(deviceH) / scale, Scale: scale,
		DeviceW: deviceW, DeviceH: deviceH,
	}
}

// SetInvert turns the negative on or off and repaints the whole screen.
func (d *Document) SetInvert(on bool) {
	if d.Invert == on {
		return
	}
	d.Invert, d.forceFull = on, true
	d.Invalidate()
}

// SetRoot makes w the root element.
func (d *Document) SetRoot(w Widget) {
	d.Root = w
	attach(w, d)
}

// Invalidate requests a repaint.
func (d *Document) Invalidate() {
	if d.OnInvalidate != nil {
		d.OnInvalidate()
	}
}

// Layout positions every element (and open popups keep their anchor).
func (d *Document) Layout() {
	if d.Root != nil {
		layoutWidget(d.Root, geom.Rect{W: d.W, H: d.H})
	}
	if d.modal != nil {
		d.layoutModal()
	}
}

// Render lays out and paints the document into a grayscale image.
func (d *Document) Render() *image.Gray {
	d.Layout()
	return d.paintFull()
}

func paintTree(c *gfx.Canvas, w Widget) {
	b := w.Elem()
	if !b.Visible() {
		return
	}
	paintBox(c, b)
	w.Paint(c)
	if cl, ok := w.(clipper); ok {
		for _, k := range b.children {
			paintClipped(c, k, cl.clipRect(), geom.Rect{})
		}
		c.ResetClip()
		return
	}
	for _, k := range b.children {
		paintTree(c, k)
	}
}

// paintBox draws the background and border described by the element's Style.
// A disabled element gets a washed-out border and fill.
func paintBox(c *gfx.Canvas, b *Base) {
	s := &b.Style
	bg, border := s.Background, s.BorderColor
	if b.Disabled {
		border = Mid
		if bg == Ink {
			bg = Mid
		}
	}
	c.FillRoundRect(b.Bounds, s.BorderRadius, bg)
	if s.BackgroundImage != nil {
		paintBackgroundImage(c, b)
	}
	c.StrokeRoundRect(b.Bounds, s.BorderRadius, s.BorderWidth, border)
}

// ---- queries (document.getElementById & friends) ----

// Walk visits every element depth-first; fn returns false to stop.
func (d *Document) Walk(fn func(Widget) bool) {
	if d.Root != nil && !walk(d.Root, fn) {
		return
	}
	if d.modal != nil { // so GetElementByID finds the modal's elements too
		walk(d.modal, fn)
	}
}

func walk(w Widget, fn func(Widget) bool) bool {
	if !fn(w) {
		return false
	}
	for _, k := range w.Elem().children {
		if !walk(k, fn) {
			return false
		}
	}
	return true
}

// GetElementByID returns the element with the given ID, or nil.
func (d *Document) GetElementByID(id string) Widget {
	var found Widget
	d.Walk(func(w Widget) bool {
		if w.Elem().ID == id {
			found = w
		}
		return found == nil
	})
	return found
}

// QuerySelectorAll returns the elements that have className.
func (d *Document) QuerySelectorAll(className string) []Widget {
	var out []Widget
	d.Walk(func(w Widget) bool {
		if w.Elem().ClassName == className {
			out = append(out, w)
		}
		return true
	})
	return out
}

// ---- focus & keyboard ----

// Focused returns the element being edited, or nil.
func (d *Document) Focused() Widget { return d.focus }

// RequestText opens the keyboard for owner; the owner is "focused" until done.
func (d *Document) RequestText(owner Widget, req KeyboardRequest) {
	if d.Keyboard == nil {
		return
	}
	done := req.Done
	req.Done = func(text string, ok bool) {
		d.focus = nil
		d.forceFull = true // the system keyboard drew over the screen
		done(text, ok)
		d.Invalidate()
	}
	d.focus = owner
	d.Keyboard.Request(req)
}

// ---- modal ----

// OpenModal shows m above the page (and below any popup). It takes every touch
// until closed, except when minimized. Opening another one replaces it.
func (d *Document) OpenModal(m *Modal) {
	d.ClosePopups()
	if d.modal != nil && d.modal != m {
		attach(d.modal, nil)
	}
	d.modal = m
	attach(m, d)
	m.Refresh()
	d.Invalidate()
}

// CloseModal closes m if it is the open modal (see Modal.Close).
func (d *Document) CloseModal(m *Modal) {
	if d.modal != m {
		return
	}
	d.modal, d.focus = nil, nil
	attach(m, nil)
	d.ClosePopups()
	d.Invalidate()
}

// HasModal reports whether a modal is open.
func (d *Document) HasModal() bool { return d.modal != nil }

// HandleBack is what the Back key does: close the open popup, else cancel the
// modal. It reports whether it did something (otherwise the app should leave).
func (d *Document) HandleBack() bool {
	switch {
	case d.HasPopup():
		d.ClosePopups()
	case d.modal != nil:
		d.modal.Cancel()
	default:
		return false
	}
	return true
}

// ---- popups (menus, drop-downs) ----

// OpenPopup shows w (opened by owner) above the rest of the UI, below (or above, when there is
// no room) anchor. A touch outside any popup closes them all.
func (d *Document) OpenPopup(owner, w Widget, anchor geom.Rect, size geom.Size) {
	d.popupOwner = owner
	r := geom.Rect{X: anchor.X, Y: anchor.Bottom() + 6, W: size.W, H: size.H}
	if r.Bottom() > d.H && anchor.Y-6-size.H >= 0 {
		r.Y = anchor.Y - 6 - size.H
	}
	r.X = geom.Clamp(r.X, 0, geom.Max(0, d.W-r.W))
	attach(w, d)
	layoutWidget(w, r)
	d.popups = append(d.popups, w)
	d.Invalidate()
}

// ClosePopups closes every open popup.
func (d *Document) ClosePopups() {
	if len(d.popups) == 0 {
		return
	}
	d.popups, d.popupOwner = nil, nil
	d.Invalidate()
}

// HasPopup reports whether a popup is open.
func (d *Document) HasPopup() bool { return len(d.popups) > 0 }

// PopupOpenFor reports whether owner currently has a popup open.
func (d *Document) PopupOpenFor(owner Widget) bool {
	return d != nil && d.popupOwner == owner && d.HasPopup()
}

// Modal returns the open modal, or nil.
func (d *Document) Modal() *Modal { return d.modal }

// paintBackgroundImage draws the element's BackgroundImage, cut to its rounded
// corners when it has any.
func paintBackgroundImage(c *gfx.Canvas, b *Base) {
	s := &b.Style
	if s.BorderRadius > 0 {
		c.Clip(b.Bounds, s.BorderRadius)
		defer c.ResetClip()
	}
	c.DrawImage(s.BackgroundImage, b.Bounds, s.BackgroundSize)
}
