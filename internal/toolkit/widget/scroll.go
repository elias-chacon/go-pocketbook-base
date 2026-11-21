package widget

import (
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
)

// Scroll is a panel with a fixed height whose content may be taller: it shows a window onto the
// content, with a scroll bar on the right. The content moves by dragging a finger over it (on a
// part that is not a button or a field), by dragging the thumb of the bar, or by tapping the bar
// above or below the thumb (one page at a time). Elements that the window cuts are drawn cut
// and the ones outside it are neither drawn nor touched.
//
// The height must be definite: give it Style.Height, or Style.FlexGrow inside a column that has
// one. The width of the content is the width of the panel without the bar. The e-ink screen
// repaints what moved on every step of a drag, so for a long list prefer the Pager; the Scroll is
// for a page of cards that does not fit the screen.
type Scroll struct {
	Base
	// Content is the column the children go in (add more with Content.AppendChild).
	Content *Panel
	// Offset is how many design pixels of the content are above the window.
	Offset float64
	// OnScroll is called after the offset changed by a touch or by ScrollTo.
	OnScroll Handler

	mode       scrollMode
	dragFrom   float64
	dragOffset float64
}

type scrollMode int

const (
	scrollIdle scrollMode = iota
	scrollSwipe
	scrollThumb
	scrollTrack
)

const (
	scrollBarW   = 26.0 // the bar and the gap between it and the content
	scrollThumbW = 14.0
	scrollMinTh  = 70.0 // the thumb never gets shorter than this, to be easy to grab
)

// NewScroll creates a scrolling panel with the given children stacked in a column.
func NewScroll(children ...Widget) *Scroll {
	s := &Scroll{}
	s.Init(s)
	s.Style.Padding = geom.Insets{Right: scrollBarW + 10}
	s.Content = NewColumn(24, children...)
	s.AppendChild(s.Content)
	return s
}

// viewport is the window onto the content: the panel without its bar.
func (s *Scroll) viewport() geom.Rect { return s.Bounds.Inset(s.Style.boxInsets()) }

// MaxOffset is the largest Offset: the content that does not fit the window (0 when it fits).
func (s *Scroll) MaxOffset() float64 {
	return geom.Max(0, s.Content.Bounds.H-s.viewport().H)
}

// PageSize is how far a tap on the bar moves the content: a window, less a few lines to keep
// some context.
func (s *Scroll) PageSize() float64 { return geom.Max(1, s.viewport().H-96) }

// ScrollTo moves the content so that y design pixels of it are above the window.
func (s *Scroll) ScrollTo(y float64) {
	y = geom.Clamp(y, 0, s.MaxOffset())
	if y == s.Offset {
		return
	}
	s.Offset = y
	s.Invalidate()
	Fire(s.OnScroll, s)
}

// ScrollBy moves the content by dy design pixels (positive: down).
func (s *Scroll) ScrollBy(dy float64) { s.ScrollTo(s.Offset + dy) }

// ScrollIntoView scrolls the least that shows w (a descendant) whole, or its top when it is
// taller than the window. Call it after a layout: it compares the bounds as they are now.
func (s *Scroll) ScrollIntoView(w Widget) {
	b, v := w.Elem().Bounds, s.viewport()
	switch {
	case b.Y < v.Y || b.H > v.H:
		s.ScrollBy(b.Y - v.Y)
	case b.Bottom() > v.Bottom():
		s.ScrollBy(b.Bottom() - v.Bottom())
	}
}

// afterLayout is called by the layout once the children are placed at offset 0: it keeps the offset
// in range (the content may have become shorter) and moves the content up by it.
func (s *Scroll) afterLayout() {
	s.Offset = geom.Clamp(s.Offset, 0, s.MaxOffset())
	if s.Offset != 0 {
		translate(s.Content, 0, -s.Offset)
	}
}

// translate moves w and everything inside it.
func translate(w Widget, dx, dy float64) {
	b := w.Elem()
	b.Bounds.X += dx
	b.Bounds.Y += dy
	for _, c := range b.children {
		translate(c, dx, dy)
	}
}

// Measure implements Widget (a Scroll has children, so this is not used for its size).
func (s *Scroll) Measure(avail float64) geom.Size { return geom.Size{W: avail} }

// bar is the track of the scroll bar, and thumb the part of it that shows where the window is.
func (s *Scroll) bar() geom.Rect {
	v := s.viewport()
	return geom.R(v.Right()+10, v.Y, scrollThumbW, v.H)
}

func (s *Scroll) thumb() geom.Rect {
	track, total := s.bar(), s.Content.Bounds.H
	if total <= 0 {
		return track
	}
	h := geom.Clamp(track.H*s.viewport().H/total, scrollMinTh, track.H)
	y := track.Y
	if m := s.MaxOffset(); m > 0 {
		y += (track.H - h) * s.Offset / m
	}
	return geom.R(track.X, y, track.W, h)
}

// Paint implements Widget: the bar, when there is something to scroll.
func (s *Scroll) Paint(c *gfx.Canvas) {
	if s.MaxOffset() <= 0 {
		return
	}
	c.FillRoundRect(s.bar(), scrollThumbW/2, Soft)
	c.FillRoundRect(s.thumb(), scrollThumbW/2, Ink)
}

// clipRect is the window the children are cut to (see paintTree).
func (s *Scroll) clipRect() geom.Rect { return s.viewport() }

// HandlePointer implements PointerHandler.
func (s *Scroll) HandlePointer(e *PointerEvent) {
	switch e.Type {
	case PointerDown:
		s.mode, s.dragFrom, s.dragOffset = scrollSwipe, e.Y, s.Offset
		if e.X >= s.bar().X-10 && s.MaxOffset() > 0 {
			s.pressBar(e)
		}
	case PointerMove:
		s.drag(e.Y)
	case PointerUp:
		s.mode = scrollIdle
	}
}

// pressBar starts a drag of the thumb, or pages when the touch is on the track.
func (s *Scroll) pressBar(e *PointerEvent) {
	th := s.thumb()
	if e.Y >= th.Y && e.Y <= th.Bottom() {
		s.mode = scrollThumb
		return
	}
	s.mode = scrollTrack
	if e.Y < th.Y {
		s.ScrollBy(-s.PageSize())
	} else {
		s.ScrollBy(s.PageSize())
	}
}

func (s *Scroll) drag(y float64) {
	switch s.mode {
	case scrollSwipe:
		s.ScrollTo(s.dragOffset - (y - s.dragFrom))
	case scrollThumb:
		if free := s.bar().H - s.thumb().H; free > 0 {
			s.ScrollTo(s.dragOffset + (y-s.dragFrom)*s.MaxOffset()/free)
		}
	}
}

// clipper is implemented by an element that shows its children through a window: what is outside
// the window is not drawn.
type clipper interface{ clipRect() geom.Rect }

// afterLayouter is implemented by an element that adjusts its children once the layout placed them.
type afterLayouter interface{ afterLayout() }

// InView reports whether the center of w is inside the window of every Scroll above it: an element
// scrolled out of sight is not reachable by touch (the tests of the touch-accuracy rule skip it).
func InView(w Widget) bool {
	center := w.Elem().Bounds.Center()
	for p := w.Elem().parent; p != nil; p = p.Elem().parent {
		if cl, ok := p.(clipper); ok && !cl.clipRect().Contains(center) {
			return false
		}
	}
	return true
}

// intersects reports whether two rectangles share some area.
func intersects(a, b geom.Rect) bool {
	return a.X < b.Right() && a.Right() > b.X && a.Y < b.Bottom() && a.Bottom() > b.Y
}

// within reports whether a is entirely inside b.
func within(a, b geom.Rect) bool {
	return a.X >= b.X && a.Right() <= b.Right() && a.Y >= b.Y && a.Bottom() <= b.Bottom()
}

// paintClipped paints w and its subtree through the window clip: what is outside is skipped, what
// is cut is drawn under a clip (applied again before every element, because an element that clips
// for itself drops it). Only what also touches area is painted (a zero area: everything).
func paintClipped(c *gfx.Canvas, w Widget, clip, area geom.Rect) {
	b := w.Elem()
	if !b.Visible() || !intersects(b.Bounds, clip) || (area.W > 0 && !intersects(b.Bounds, area)) {
		return
	}
	if within(b.Bounds, clip) {
		c.ResetClip()
	} else {
		c.Clip(clip, 0)
	}
	paintBox(c, b)
	w.Paint(c)
	for _, k := range b.children {
		paintClipped(c, k, clip, area)
	}
}
