package widget

import (
	"fmt"
	"image"
	"math"
	"reflect"
	"strings"

	"go-pocketbook-base/internal/geom"
	"go-pocketbook-base/internal/gfx"
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
	sig    string
	bounds geom.Rect
}

// signer lets an element add state to its snapshot that the reflection below
// cannot see (state behind pointers, like a menu bar's *Menu items).
type signer interface{ StateSignature() string }

// ResetFrame makes the next Update repaint the whole screen, without asking for
// a repaint: use it when something outside the document drew over the screen
// (the system keyboard) or when the system asks for a redraw on its own.
func (d *Document) ResetFrame() { d.forceFull = true }

// InvalidateAll is ResetFrame plus a repaint request.
func (d *Document) InvalidateAll() {
	d.ResetFrame()
	d.Invalidate()
}

// Update lays the document out and repaints what changed since the last call:
// the first call, InvalidateAll and big changes repaint everything; otherwise
// only the changed areas are painted and listed in Frame.Rects.
func (d *Document) Update() Frame {
	d.Layout()
	cur := d.snapshot()
	full := d.forceFull || d.frame == nil
	var rects []image.Rectangle
	if !full {
		rects, full = d.changedRects(cur)
	}
	d.snap, d.forceFull = cur, false

	if full {
		d.frame = d.paintFull()
		return Frame{Image: d.frame, Rects: []image.Rectangle{d.frame.Bounds()}, Full: true}
	}
	for _, r := range rects {
		d.paintRegion(r)
	}
	return Frame{Image: d.frame, Rects: rects}
}

// snapshot records every element of the tree and of the open popups.
func (d *Document) snapshot() map[Widget]snap {
	out := make(map[Widget]snap, len(d.snap))
	add := func(w Widget) bool {
		out[w] = snap{sig: d.signature(w), bounds: w.Elem().Bounds}
		return true
	}
	if d.Root != nil {
		walk(d.Root, add)
	}
	for _, p := range d.popups {
		walk(p, add)
	}
	return out
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
	if v := reflect.ValueOf(w); v.Kind() == reflect.Ptr && !v.IsNil() && v.Elem().Kind() == reflect.Struct {
		fmt.Fprintf(&sb, "%v", v.Elem())
	} else {
		fmt.Fprintf(&sb, "%v", w)
	}
	// Painted from document state, not from the element's own fields.
	fmt.Fprintf(&sb, "|focus=%t|popup=%t", d.focus == w, d.PopupOpenFor(w))
	return sb.String()
}

// changedRects compares cur with the previous snapshot and returns the device
// areas to repaint, or full=true when repainting everything is the better deal.
func (d *Document) changedRects(cur map[Widget]snap) (rects []image.Rectangle, full bool) {
	add := func(r geom.Rect) {
		if r.W > 0 && r.H > 0 {
			rects = append(rects, d.deviceRect(r))
		}
	}
	for w, n := range cur {
		o, known := d.snap[w]
		switch {
		case !known:
			add(n.bounds)
		case o.sig != n.sig:
			add(o.bounds)
			add(n.bounds)
		}
	}
	for w, o := range d.snap {
		if _, still := cur[w]; !still {
			add(o.bounds)
		}
	}
	rects = mergeRects(rects)
	if len(rects) > maxDirtyRects {
		rects = []image.Rectangle{unionAll(rects)}
	}
	area := 0
	for _, r := range rects {
		area += r.Dx() * r.Dy()
	}
	screen := d.DeviceW * d.DeviceH
	return rects, float64(area) > fullAreaShare*float64(screen)
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
	c.Clear(Paper)
	if d.Root != nil {
		paintTree(c, d.Root)
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
	c.Clear(Paper)
	area := geom.R(float64(w.Min.X)/d.Scale, float64(w.Min.Y)/d.Scale, float64(w.Dx())/d.Scale, float64(w.Dy())/d.Scale)
	if d.Root != nil {
		paintTreeIn(c, d.Root, area)
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
	for _, k := range b.children {
		paintTreeIn(c, k, area)
	}
}
