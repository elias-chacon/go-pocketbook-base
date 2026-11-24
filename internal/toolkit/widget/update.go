package widget

import (
	"fmt"
	"image"
	"math"
	"reflect"
	"strings"
	"time"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
)

// Partial repaint.
//
// Painting the whole screen costs ~1 s on the reader, but a tap usually changes
// one small thing. Update works out WHAT changed by comparing a "snapshot" of
// every element (its fields plus its bounds) with the one taken at the previous
// Update, repaints only that area into a small canvas and patches it into the
// last frame. Elements do not have to announce their changes: setting
// cb.Checked = true in a handler is enough, as before.
//
// The result is pixel-identical to a full Render (tests check it), so it is
// only an optimization: when too much changed (a page switch, a popup over half
// the screen) or when in doubt, the whole screen is repainted.

const (
	// paintSlack is how far (design px) an element may paint outside its Bounds
	// (focus ring, anti-aliased edges, a slider knob).
	paintSlack = 8.0
	// maxDirtyRects caps the areas sent to the panel; more are merged into one.
	maxDirtyRects = 4
	// fullAreaShare is the part of the screen above which a full repaint is
	// cheaper than patching.
	fullAreaShare = 0.4
)

// Frame is what Update returns.
type Frame struct {
	// Image is the whole screen, complete and up to date.
	Image *image.Gray
	// Rects are the device-pixel areas that changed since the previous Update
	// (the whole screen when Full). Empty when nothing changed.
	Rects []image.Rectangle
	// Full is true when the whole screen was repainted.
	Full bool
}

// snap is what Update remembers about an element between two frames.
type snap struct {
	sig string
	// bounds are the element's bounds as if every Scroll above it were at offset 0, so that
	// scrolling alone does not change them.
	bounds geom.Rect
	scroll *Scroll   // the innermost Scroll above the element (nil when none)
	shift  float64   // how far the Scrolls above had moved it up when this was taken
	view   geom.Rect // the window of that Scroll then, on the screen (empty without one)
}

// screen is where the element is on the screen, cut to the window of its Scroll, once the Scrolls
// above it have moved it up by shift.
func (n snap) screen(shift float64) geom.Rect {
	r := n.bounds
	r.Y -= shift
	if n.scroll == nil {
		return r
	}
	return cut(r, n.view)
}

// cut is the part of r that lies inside v (empty when there is none).
func cut(r, v geom.Rect) geom.Rect {
	x0, y0 := geom.Max(r.X, v.X), geom.Max(r.Y, v.Y)
	x1, y1 := geom.Min(r.Right(), v.Right()), geom.Min(r.Bottom(), v.Bottom())
	if x1 <= x0 || y1 <= y0 {
		return geom.Rect{}
	}
	return geom.R(x0, y0, x1-x0, y1-y0)
}

// signer lets an element add state to its snapshot that the reflection below
// cannot see (state behind pointers, like a menu bar's *Menu items).
type signer interface{ StateSignature() string }

// UpdateStats is how long the last Update took, step by step, and what it did with the screen.
type UpdateStats struct {
	Layout   time.Duration // Document.Layout
	Snapshot time.Duration // describing every element (the reflection)
	Diff     time.Duration // working out what changed
	Paint    time.Duration // sliding and repainting
	Full     bool          // the whole screen was repainted
	Slid     bool          // the pixels of a Scroll were moved instead of repainted
	Painted  int           // device pixels repainted
	Sent     int           // device pixels in Frame.Rects (what goes to the panel)
}

// LastUpdate is the cost of the last Update.
func (d *Document) LastUpdate() UpdateStats { return d.stats }

// ResetFrame makes the next Update repaint the whole screen, without asking for
// a repaint: use it when something outside the document drew over the screen
// (the system keyboard) or when the system asks for a redraw on its own.
func (d *Document) ResetFrame() { d.forceFull = true }

// InvalidateAll is ResetFrame plus a repaint request.
func (d *Document) InvalidateAll() {
	d.ResetFrame()
	d.Invalidate()
}

// updatePlan is what one Update does to the frame.
type updatePlan struct {
	slides []slide
	paint  []image.Rectangle // repainted after the slides
	send   []image.Rectangle // what the panel is told about: the repainted areas and the slid windows
}

// Update lays the document out and repaints what changed since the last call:
// the first call, InvalidateAll and big changes repaint everything; otherwise
// only the changed areas are painted and listed in Frame.Rects. A Scroll that only moved
// slides the pixels it already has and paints the strip it uncovered (see slide).
func (d *Document) Update() Frame {
	t0 := time.Now()
	d.Layout()
	t1 := time.Now()
	cur, offs := d.snapshot()
	t2 := time.Now()
	full := d.forceFull || d.frame == nil
	var plan updatePlan
	if !full {
		plan, full = d.plan(cur, offs)
	}
	t3 := time.Now()
	d.snap, d.scrollOff, d.forceFull = cur, offs, false

	st := UpdateStats{Layout: t1.Sub(t0), Snapshot: t2.Sub(t1), Diff: t3.Sub(t2), Full: full}
	var f Frame
	if full {
		d.frame = d.paintFull()
		f = Frame{Image: d.frame, Rects: []image.Rectangle{d.frame.Bounds()}, Full: true}
		st.Painted = area(d.frame.Bounds())
	} else {
		for _, s := range plan.slides {
			d.slide(s)
		}
		for _, r := range plan.paint {
			d.paintRegion(r)
			st.Painted += area(r)
		}
		st.Slid = len(plan.slides) > 0
		f = Frame{Image: d.frame, Rects: plan.send}
	}
	for _, r := range f.Rects {
		st.Sent += area(r)
	}
	st.Paint = time.Since(t3)
	d.stats = st
	return f
}

// snapCtx is where an element is in the tree, as far as the Scrolls above it go.
type snapCtx struct {
	scroll *Scroll
	shift  float64
	view   geom.Rect
}

// snapshot records every element of the tree and of the open popups, and the offset of every
// Scroll. The elements of a Scroll are recorded as if it were at offset 0: moving it changes
// none of them.
func (d *Document) snapshot() (map[Widget]snap, map[*Scroll]float64) {
	out := make(map[Widget]snap, len(d.snap))
	offs := make(map[*Scroll]float64)
	if d.Root != nil {
		d.snapTree(out, offs, d.Root, snapCtx{})
	}
	if d.modal != nil {
		d.snapTree(out, offs, d.modal, snapCtx{})
	}
	for _, p := range d.popups {
		d.snapTree(out, offs, p, snapCtx{})
	}
	return out, offs
}

func (d *Document) snapTree(out map[Widget]snap, offs map[*Scroll]float64, w Widget, c snapCtx) {
	b := w.Elem()
	out[w] = snap{sig: d.signature(w), bounds: b.Bounds, scroll: c.scroll, shift: c.shift, view: c.view}
	sc, isScroll := w.(*Scroll)
	if !isScroll {
		for _, k := range b.children {
			d.snapTree(out, offs, k, c)
		}
		return
	}
	offs[sc] = sc.Offset
	view := sc.viewport()
	view.Y -= c.shift
	inner := snapCtx{scroll: sc, shift: c.shift + sc.Offset, view: view}
	saved := normalize(sc.Content, sc.Offset, nil)
	for _, k := range b.children {
		d.snapTree(out, offs, k, inner)
	}
	for _, s := range saved {
		s.b.Bounds = s.r
	}
}

// savedBounds is the exact bounds of an element before normalize moved it.
type savedBounds struct {
	b *Base
	r geom.Rect
}

// normalize puts the tree w back where the layout placed it, before the Scroll moved it up by dy,
// rounded to 1/1024 so that the sum does not leave a trace of rounding that would make an
// element look different from one frame to the next. It returns the bounds to restore.
func normalize(w Widget, dy float64, saved []savedBounds) []savedBounds {
	b := w.Elem()
	saved = append(saved, savedBounds{b, b.Bounds})
	b.Bounds.Y = math.Round((b.Bounds.Y+dy)*1024) / 1024
	for _, k := range b.children {
		saved = normalize(k, dy, saved)
	}
	return saved
}

// signature describes everything that can change how w looks: its fields (the
// text, the checked flag, the page of a table...) and its bounds, printed with
// %v. Pointers print as addresses, so children and parents are not followed:
// a change in a child shows up in the child's own signature.
func (d *Document) signature(w Widget) string {
	var sb strings.Builder
	if s, ok := w.(signer); ok {
		sb.WriteString(s.StateSignature())
	}
	v := reflect.ValueOf(w)
	if sc, ok := w.(*Scroll); ok {
		// Where the window is (and how a drag is going) is handled by the slide; the height of
		// the content decides how long the thumb is.
		c := *sc
		c.Offset, c.mode, c.dragFrom, c.dragOffset = 0, scrollIdle, 0, 0
		v = reflect.ValueOf(&c)
		fmt.Fprintf(&sb, "content=%v|", sc.Content.Bounds.H)
	}
	if v.Kind() == reflect.Ptr && !v.IsNil() && v.Elem().Kind() == reflect.Struct {
		fmt.Fprintf(&sb, "%v", v.Elem())
	} else {
		fmt.Fprintf(&sb, "%v", w)
	}
	// Painted from document state, not from the element's own fields.
	fmt.Fprintf(&sb, "|focus=%t|popup=%t", d.focus == w, d.PopupOpenFor(w))
	return sb.String()
}

// plan works out what to do to the frame, or full=true when repainting everything is the better
// deal.
func (d *Document) plan(cur map[Widget]snap, offs map[*Scroll]float64) (updatePlan, bool) {
	slides, repaint := d.slides(cur, offs)
	sliding := make(map[*Scroll]bool, len(slides))
	for _, s := range slides {
		sliding[s.scroll] = true
	}
	paint := d.changedRects(cur, offs, sliding)
	paint = append(paint, repaint...)
	for _, s := range slides {
		paint = append(paint, s.paint...)
	}
	paint = capRects(mergeRects(paint))
	n := 0
	for _, r := range paint {
		n += area(r)
	}
	if float64(n) > fullAreaShare*float64(d.DeviceW*d.DeviceH) {
		return updatePlan{}, true
	}
	send := append([]image.Rectangle(nil), paint...)
	for _, s := range slides {
		send = append(send, s.view)
	}
	return updatePlan{slides: slides, paint: paint, send: capRects(mergeRects(send))}, false
}

// changedRects compares cur with the previous snapshot and returns the device
// areas to repaint. Where a Scroll slid its pixels, what an element covered before is where it is
// now.
func (d *Document) changedRects(cur map[Widget]snap, offs map[*Scroll]float64, sliding map[*Scroll]bool) []image.Rectangle {
	var rects []image.Rectangle
	add := func(r geom.Rect) {
		if r.W > 0 && r.H > 0 {
			rects = append(rects, d.deviceRect(r))
		}
	}
	before := func(o snap) geom.Rect {
		if o.scroll != nil && sliding[o.scroll] {
			return o.screen(offs[o.scroll])
		}
		return o.screen(o.shift)
	}
	for w, n := range cur {
		o, known := d.snap[w]
		switch {
		case !known:
			add(n.screen(n.shift))
		case o.sig != n.sig:
			add(before(o))
			add(n.screen(n.shift))
		}
	}
	for w, o := range d.snap {
		if _, still := cur[w]; !still {
			add(before(o))
		}
	}
	return rects
}

// slide is a Scroll whose content moved: its window is moved with it instead of repainted.
type slide struct {
	scroll *Scroll
	view   image.Rectangle   // the window whose pixels move
	dy     int               // device pixels the content went up (negative: down)
	paint  []image.Rectangle // what the move uncovers or leaves out: the strip, the bar, the edges
}

// slides finds the Scrolls that moved since the last frame. One that can slide the pixels it has
// does; the others are listed in repaint, to be painted whole.
func (d *Document) slides(cur map[Widget]snap, offs map[*Scroll]float64) (slides []slide, repaint []image.Rectangle) {
	free := d.modal == nil && len(d.popups) == 0 // nothing floats over the window
	for w, n := range cur {
		if _, ok := w.(*Scroll); ok && n.scroll != nil {
			free = false // a Scroll inside a Scroll
		}
	}
	for s, off := range offs {
		prev, known := d.scrollOff[s]
		if !known || prev == off {
			continue
		}
		n := cur[s]
		box := n.bounds
		box.Y -= n.shift
		v := box.Inset(s.Style.boxInsets())
		dy := int(math.Round((off - prev) * d.Scale))
		view := d.windowRect(v, true)
		if !free || dy == 0 || dy >= view.Dy() || -dy >= view.Dy() {
			repaint = append(repaint, d.deviceRect(box))
			continue
		}
		slides = append(slides, d.newSlide(s, v, view, dy))
	}
	return slides, repaint
}

// newSlide lists what moving the window of s by dy leaves to paint.
func (d *Document) newSlide(s *Scroll, v geom.Rect, view image.Rectangle, dy int) slide {
	sl := slide{scroll: s, view: view, dy: dy}
	if dy > 0 {
		sl.paint = append(sl.paint, image.Rect(view.Min.X, view.Max.Y-dy, view.Max.X, view.Max.Y))
	} else {
		sl.paint = append(sl.paint, image.Rect(view.Min.X, view.Min.Y, view.Max.X, view.Min.Y-dy))
	}
	sl.paint = append(sl.paint, d.deviceRect(barRect(v)))
	sl.paint = append(sl.paint, ring(d.windowRect(v, false), view)...)
	return sl
}

// windowRect is the device rectangle of v: the biggest one inside it when inner, else the smallest
// one that holds it.
func (d *Document) windowRect(v geom.Rect, inner bool) image.Rectangle {
	lo, hi := math.Floor, math.Ceil
	if inner {
		lo, hi = math.Ceil, math.Floor
	}
	r := image.Rect(int(lo(v.X*d.Scale)), int(lo(v.Y*d.Scale)), int(hi(v.Right()*d.Scale)), int(hi(v.Bottom()*d.Scale)))
	return r.Intersect(image.Rect(0, 0, d.DeviceW, d.DeviceH))
}

// ring is outer without inner, as up to four rectangles (none when the two are the same).
func ring(outer, inner image.Rectangle) []image.Rectangle {
	var out []image.Rectangle
	for _, r := range []image.Rectangle{
		image.Rect(outer.Min.X, outer.Min.Y, outer.Max.X, inner.Min.Y),
		image.Rect(outer.Min.X, inner.Max.Y, outer.Max.X, outer.Max.Y),
		image.Rect(outer.Min.X, inner.Min.Y, inner.Min.X, inner.Max.Y),
		image.Rect(inner.Max.X, inner.Min.Y, outer.Max.X, inner.Max.Y),
	} {
		if !r.Empty() {
			out = append(out, r)
		}
	}
	return out
}

// slide moves the pixels of the window of s by the amount its content moved, in the last frame.
// What is left of the window uncovered is in s.paint.
func (d *Document) slide(s slide) {
	f, v := d.frame, s.view
	w := v.Dx()
	if s.dy > 0 { // the content went up: copy from below, top first
		for y := v.Min.Y; y < v.Max.Y-s.dy; y++ {
			copy(f.Pix[y*f.Stride+v.Min.X:][:w], f.Pix[(y+s.dy)*f.Stride+v.Min.X:][:w])
		}
		return
	}
	n := -s.dy // the content went down: copy from above, bottom first
	for y := v.Max.Y - 1; y >= v.Min.Y+n; y-- {
		copy(f.Pix[y*f.Stride+v.Min.X:][:w], f.Pix[(y-n)*f.Stride+v.Min.X:][:w])
	}
}

// capRects joins the areas into one when there are too many of them.
func capRects(rs []image.Rectangle) []image.Rectangle {
	if len(rs) > maxDirtyRects {
		return []image.Rectangle{unionAll(rs)}
	}
	return rs
}

// deviceRect converts a design rectangle to device pixels, grown by paintSlack
// and clamped to the screen.
func (d *Document) deviceRect(r geom.Rect) image.Rectangle {
	x0 := int(math.Floor((r.X - paintSlack) * d.Scale))
	y0 := int(math.Floor((r.Y - paintSlack) * d.Scale))
	x1 := int(math.Ceil((r.Right() + paintSlack) * d.Scale))
	y1 := int(math.Ceil((r.Bottom() + paintSlack) * d.Scale))
	return image.Rect(x0, y0, x1, y1).Intersect(image.Rect(0, 0, d.DeviceW, d.DeviceH))
}

// mergeRects joins rectangles that overlap or that are close enough for one
// bigger area to cost about the same as two (the panel update has a fixed cost).
func mergeRects(rs []image.Rectangle) []image.Rectangle {
	for merged := true; merged; {
		merged = false
	scan:
		for i := 0; i < len(rs); i++ {
			for j := i + 1; j < len(rs); j++ {
				u := rs[i].Union(rs[j])
				if rs[i].Overlaps(rs[j]) || area(u) <= 3*(area(rs[i])+area(rs[j]))/2 {
					rs[i] = u
					rs = append(rs[:j], rs[j+1:]...)
					merged = true
					break scan
				}
			}
		}
	}
	return rs
}

func area(r image.Rectangle) int { return r.Dx() * r.Dy() }

func unionAll(rs []image.Rectangle) image.Rectangle {
	u := rs[0]
	for _, r := range rs[1:] {
		u = u.Union(r)
	}
	return u
}

// paintFull repaints the whole screen and returns it as a new image.
func (d *Document) paintFull() *image.Gray {
	if d.canvas == nil {
		d.canvas = gfx.NewCanvas(d.DeviceW, d.DeviceH, d.Scale)
	}
	c := d.canvas
	c.Invert = d.Invert
	c.Clear(Paper)
	if d.Root != nil {
		paintTree(c, d.Root)
	}
	if d.modal != nil {
		paintTree(c, d.modal)
	}
	for _, p := range d.popups {
		paintTree(c, p)
	}
	return c.Image()
}

// windowPad is the extra border (device px) painted around a region and thrown
// away: the rasterizer does not anti-alias a shape correctly where it crosses
// the edge of the canvas, so the edge of the window must not be the edge of
// what we keep.
const windowPad = 24

// paintRegion repaints the device area r into a canvas a little bigger than r
// and copies r into the last frame.
func (d *Document) paintRegion(r image.Rectangle) {
	w := r.Inset(-windowPad).Intersect(d.frame.Bounds()) // the screen's own edges are real edges
	c := gfx.NewCanvasAt(w.Dx(), w.Dy(), d.Scale, w.Min.X, w.Min.Y)
	c.Invert = d.Invert
	c.Clear(Paper)
	area := geom.R(float64(w.Min.X)/d.Scale, float64(w.Min.Y)/d.Scale, float64(w.Dx())/d.Scale, float64(w.Dy())/d.Scale)
	if d.Root != nil {
		paintTreeIn(c, d.Root, area)
	}
	if d.modal != nil {
		paintTreeIn(c, d.modal, area)
	}
	for _, p := range d.popups {
		paintTreeIn(c, p, area)
	}
	g := c.Image()
	for y := r.Min.Y; y < r.Max.Y; y++ {
		src := g.Pix[(y-w.Min.Y)*g.Stride+(r.Min.X-w.Min.X):]
		dst := d.frame.Pix[y*d.frame.Stride+r.Min.X:]
		copy(dst[:r.Dx()], src[:r.Dx()])
	}
}

// paintTreeIn is paintTree for a window onto the screen: elements that cannot
// touch area are skipped (their children are still visited, they may overflow).
func paintTreeIn(c *gfx.Canvas, w Widget, area geom.Rect) {
	b := w.Elem()
	if !b.Visible() {
		return
	}
	if b.Bounds.X-paintSlack < area.Right() && b.Bounds.Right()+paintSlack > area.X &&
		b.Bounds.Y-paintSlack < area.Bottom() && b.Bounds.Bottom()+paintSlack > area.Y {
		paintBox(c, b)
		w.Paint(c)
	}
	if cl, ok := w.(clipper); ok {
		for _, k := range b.children {
			paintClipped(c, k, cl.clipRect(), area)
		}
		c.ResetClip()
		return
	}
	for _, k := range b.children {
		paintTreeIn(c, k, area)
	}
}
