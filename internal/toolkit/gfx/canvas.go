package gfx

import (
	"image"
	"image/draw"
	"math"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/raster"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"go-pocketbook-base/internal/kernel/geom"
)

// TextAlign positions text horizontally inside its box.
type TextAlign int

// Text alignments.
const (
	AlignStart TextAlign = iota
	AlignCenter
	AlignEnd
)

// TextStyle describes one line of text.
type TextStyle struct {
	Size     float64
	Bold     bool
	Color    Color
	Align    TextAlign
	Ellipsis bool // shorten with "…" when it does not fit the box
}

// capHalf is half the cap height of the font as a fraction of its size: text is
// vertically centered on the middle of its capital letters, not of the line box.
const capHalf = 0.36

// Canvas draws in design units; scale converts them to device pixels, so the
// same screen description renders at any resolution.
type Canvas struct {
	dc     *gg.Context
	rgba   *image.RGBA
	scale  float64
	ox, oy float64 // device position of the canvas's top-left corner (see NewCanvasAt)

	// clip state (see Clip): the mask buffer and rasterizer are allocated on
	// first use and reused, so clipping costs only the clipped area.
	mask *image.Alpha
	ras  *raster.Rasterizer
	clip image.Rectangle // device area touched by the active clip; empty when none
}

// NewCanvas creates a canvas of w x h device pixels.
func NewCanvas(w, h int, scale float64) *Canvas {
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	return &Canvas{dc: gg.NewContextForRGBA(rgba), rgba: rgba, scale: scale}
}

// NewCanvasAt creates a canvas of w x h device pixels that is a window onto the
// screen starting at device pixel (ox, oy): drawing a widget at its screen
// position lands in the right place inside the window, pixel for pixel as it
// would in a full-screen canvas. Used to repaint only the part that changed.
func NewCanvasAt(w, h int, scale float64, ox, oy int) *Canvas {
	c := NewCanvas(w, h, scale)
	c.ox, c.oy = float64(ox), float64(oy)
	c.dc.Translate(-c.ox, -c.oy) // shapes and text; not applied by hand in Clip and clippedString
	return c
}

// Size is the canvas size in device pixels.
func (c *Canvas) Size() (w, h int) { return c.dc.Width(), c.dc.Height() }

// Image returns the canvas as an 8-bit grayscale image (what InkView blits).
//
// The conversion is a plain loop over the bytes: draw.Draw between RGBA and Gray
// goes pixel by pixel through color.Color and took 2.7 s of a 3.8 s frame on the
// reader (2.6 M pixels). The canvas is opaque (Clear paints the paper), so the
// RGB bytes are not premultiplied by anything.
func (c *Canvas) Image() *image.Gray {
	src := c.rgba
	w, h := src.Rect.Dx(), src.Rect.Dy()
	g := image.NewGray(src.Rect)
	for y := 0; y < h; y++ {
		in := src.Pix[y*src.Stride : y*src.Stride+w*4]
		out := g.Pix[y*g.Stride : y*g.Stride+w]
		for x := range out {
			p := in[x*4 : x*4+4 : x*4+4]
			// Rec. 601 luma in integers: (299 R + 587 G + 114 B) / 1000.
			out[x] = uint8((299*uint32(p[0]) + 587*uint32(p[1]) + 114*uint32(p[2]) + 500) / 1000)
		}
	}
	return g
}

func (c *Canvas) color(col Color) {
	c.dc.SetRGBA255(int(col.R), int(col.G), int(col.B), int(col.A))
}

func (c *Canvas) s(v float64) float64 { return v * c.scale }

// Clear fills the whole canvas.
func (c *Canvas) Clear(col Color) {
	c.ResetClip()
	c.color(col)
	c.dc.Clear()
}

// FillRoundRect fills r with rounded corners of the given radius.
func (c *Canvas) FillRoundRect(r geom.Rect, radius float64, col Color) {
	if col.IsZero() {
		return
	}
	c.color(col)
	c.dc.DrawRoundedRectangle(c.s(r.X), c.s(r.Y), c.s(r.W), c.s(r.H), c.s(radius))
	c.dc.Fill()
}

// StrokeRoundRect draws a border of the given width fully inside r.
func (c *Canvas) StrokeRoundRect(r geom.Rect, radius, width float64, col Color) {
	if col.IsZero() || width <= 0 {
		return
	}
	half := width / 2
	c.color(col)
	c.dc.SetLineWidth(c.s(width))
	c.dc.DrawRoundedRectangle(c.s(r.X+half), c.s(r.Y+half), c.s(r.W-width), c.s(r.H-width), c.s(geom.Max(0, radius-half)))
	c.dc.Stroke()
}

// Line draws a straight segment with round caps.
func (c *Canvas) Line(a, b geom.Point, width float64, col Color) {
	c.Polyline([]geom.Point{a, b}, width, col)
}

// Polyline draws connected segments with round caps and joins.
func (c *Canvas) Polyline(pts []geom.Point, width float64, col Color) {
	if len(pts) < 2 || col.IsZero() {
		return
	}
	c.color(col)
	c.dc.SetLineWidth(c.s(width))
	c.dc.SetLineCapRound()
	c.dc.SetLineJoinRound()
	c.dc.NewSubPath()
	c.dc.MoveTo(c.s(pts[0].X), c.s(pts[0].Y))
	for _, p := range pts[1:] {
		c.dc.LineTo(c.s(p.X), c.s(p.Y))
	}
	c.dc.Stroke()
}

// FillCircle fills a disc.
func (c *Canvas) FillCircle(center geom.Point, radius float64, col Color) {
	if col.IsZero() {
		return
	}
	c.color(col)
	c.dc.DrawCircle(c.s(center.X), c.s(center.Y), c.s(radius))
	c.dc.Fill()
}

// StrokeCircle draws a ring whose outer edge is at radius.
func (c *Canvas) StrokeCircle(center geom.Point, radius, width float64, col Color) {
	if col.IsZero() || width <= 0 {
		return
	}
	c.color(col)
	c.dc.SetLineWidth(c.s(width))
	c.dc.DrawCircle(c.s(center.X), c.s(center.Y), c.s(radius-width/2))
	c.dc.Stroke()
}

// Arc strokes an arc from angle a1 to a2 (radians, 0 = right, clockwise on screen).
func (c *Canvas) Arc(center geom.Point, radius, a1, a2, width float64, col Color) {
	if col.IsZero() || width <= 0 {
		return
	}
	c.color(col)
	c.dc.SetLineWidth(c.s(width))
	c.dc.SetLineCapRound()
	c.dc.NewSubPath()
	c.dc.DrawArc(c.s(center.X), c.s(center.Y), c.s(radius), a1, a2)
	c.dc.Stroke()
}

// Clip restricts drawing to r (rounded) until ResetClip. Clips do not nest: a
// new Clip replaces the previous one.
//
// gg's own Clip allocates a screen-size mask each time and makes every
// DrawString allocate and composite a screen-size image, which on the reader
// (1 GB of RAM) made the Data page take seconds. Here the mask is reused and
// text under a clip only touches the pixels around the text.
func (c *Canvas) Clip(r geom.Rect, radius float64) {
	c.ResetClip()
	w, h := c.Size()
	if c.mask == nil {
		c.mask = image.NewAlpha(image.Rect(0, 0, w, h))
		c.ras = raster.NewRasterizer(w, h)
	}
	x, y, rw, rh := c.s(r.X)-c.ox, c.s(r.Y)-c.oy, c.s(r.W), c.s(r.H)
	c.ras.Clear()
	c.ras.UseNonZeroWinding = true
	c.ras.AddPath(roundRectPath(x, y, rw, rh, c.s(radius)))
	c.ras.Rasterize(raster.NewAlphaOverPainter(c.mask))
	c.clip = image.Rect(int(math.Floor(x))-1, int(math.Floor(y))-1, int(math.Ceil(x+rw))+1, int(math.Ceil(y+rh))+1).Intersect(c.mask.Bounds())
	_ = c.dc.SetMask(c.mask) // same size by construction
}

// ResetClip removes the clip set by Clip.
func (c *Canvas) ResetClip() {
	c.dc.ResetClip()
	if c.clip.Empty() {
		return
	}
	for y := c.clip.Min.Y; y < c.clip.Max.Y; y++ {
		row := c.mask.Pix[c.mask.PixOffset(c.clip.Min.X, y):c.mask.PixOffset(c.clip.Max.X, y)]
		for i := range row {
			row[i] = 0
		}
	}
	c.clip = image.Rectangle{}
}

// roundRectPath is a rounded rectangle (device pixels) as a rasterizer path.
func roundRectPath(x, y, w, h, radius float64) raster.Path {
	r := math.Max(0, math.Min(radius, math.Min(w, h)/2))
	k := r * (1 - 0.5522847498) // distance from the corner to a bezier control point
	pt := func(px, py float64) fixed.Point26_6 {
		return fixed.Point26_6{X: fixed.Int26_6(px * 64), Y: fixed.Int26_6(py * 64)}
	}
	var p raster.Path
	p.Start(pt(x+r, y))
	p.Add1(pt(x+w-r, y))
	p.Add3(pt(x+w-k, y), pt(x+w, y+k), pt(x+w, y+r))
	p.Add1(pt(x+w, y+h-r))
	p.Add3(pt(x+w, y+h-k), pt(x+w-k, y+h), pt(x+w-r, y+h))
	p.Add1(pt(x+r, y+h))
	p.Add3(pt(x+k, y+h), pt(x, y+h-k), pt(x, y+h-r))
	p.Add1(pt(x, y+r))
	p.Add3(pt(x, y+k), pt(x+k, y), pt(x+r, y))
	return p
}

// Text draws one line of text inside r, vertically centered.
func (c *Canvas) Text(s string, r geom.Rect, ts TextStyle) {
	if s == "" || ts.Color.IsZero() {
		return
	}
	size := c.s(ts.Size)
	face := Face(ts.Bold, size)
	maxW := c.s(r.W)
	if ts.Ellipsis {
		s = ellipsize(s, maxW, face)
	}
	w := float64(font.MeasureString(face, s)) / 64
	x := c.s(r.X)
	switch ts.Align {
	case AlignCenter:
		x += (maxW - w) / 2
	case AlignEnd:
		x += maxW - w
	}
	baseline := c.s(r.Y+r.H/2) + capHalf*size
	if !c.clip.Empty() {
		// The scratch image has no translation, so apply the window offset here.
		c.clippedString(face, s, x-c.ox, baseline-c.oy, w, ts.Color)
		return
	}
	c.dc.SetFontFace(face)
	c.color(ts.Color)
	c.dc.DrawString(s, x, baseline)
}

// clippedString draws s under the active clip by rendering it into a small
// scratch image and compositing only that area through the clip mask.
func (c *Canvas) clippedString(face font.Face, s string, x, baseline, w float64, col Color) {
	m := face.Metrics()
	box := image.Rect(int(x)-2, int(baseline)-int(m.Ascent>>6)-2, int(x+w)+3, int(baseline)+int(m.Descent>>6)+2).Intersect(c.clip)
	if box.Empty() {
		return
	}
	tmp := gg.NewContext(box.Dx(), box.Dy())
	tmp.SetFontFace(face)
	tmp.SetRGBA255(int(col.R), int(col.G), int(col.B), int(col.A))
	tmp.DrawString(s, x-float64(box.Min.X), baseline-float64(box.Min.Y))
	draw.DrawMask(c.rgba, box, tmp.Image(), image.Point{}, c.mask, box.Min, draw.Over)
}

// ellipsize shortens s until it fits maxW device pixels.
func ellipsize(s string, maxW float64, face font.Face) string {
	if float64(font.MeasureString(face, s))/64 <= maxW {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		cand := string(runes) + "…"
		if float64(font.MeasureString(face, cand))/64 <= maxW {
			return cand
		}
	}
	return ""
}
