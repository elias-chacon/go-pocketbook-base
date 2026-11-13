package gfx

import (
	"image"
	"image/draw"

	"github.com/fogleman/gg"
	"golang.org/x/image/font"

	"go-pocketbook-base/internal/geom"
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
	dc    *gg.Context
	scale float64
}

// NewCanvas creates a canvas of w x h device pixels.
func NewCanvas(w, h int, scale float64) *Canvas {
	return &Canvas{dc: gg.NewContext(w, h), scale: scale}
}

// Image returns the canvas as an 8-bit grayscale image (what InkView blits).
func (c *Canvas) Image() *image.Gray {
	src := c.dc.Image()
	g := image.NewGray(src.Bounds())
	draw.Draw(g, g.Bounds(), src, src.Bounds().Min, draw.Src)
	return g
}

func (c *Canvas) color(col Color) {
	c.dc.SetRGBA255(int(col.R), int(col.G), int(col.B), int(col.A))
}

func (c *Canvas) s(v float64) float64 { return v * c.scale }

// Clear fills the whole canvas.
func (c *Canvas) Clear(col Color) {
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

// Clip restricts drawing to r (rounded) until ResetClip.
func (c *Canvas) Clip(r geom.Rect, radius float64) {
	c.dc.DrawRoundedRectangle(c.s(r.X), c.s(r.Y), c.s(r.W), c.s(r.H), c.s(radius))
	c.dc.Clip()
}

// ResetClip removes the clip set by Clip.
func (c *Canvas) ResetClip() { c.dc.ResetClip() }

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
	c.dc.SetFontFace(face)
	c.color(ts.Color)
	c.dc.DrawString(s, x, c.s(r.Y+r.H/2)+capHalf*size)
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
