package widget

import (
	"fmt"
	"math"

	"go-pocketbook-base/internal/geom"
	"go-pocketbook-base/internal/gfx"
)

// Progress is a horizontal progress bar (<progress>) with the percentage
// written inside; the text inverts where the fill passes under it.
type Progress struct {
	Base
	Value, Min, Max float64
	Text            string // overrides the percentage when set
}

// NewProgress creates a 0..100 progress bar.
func NewProgress(value float64) *Progress {
	p := &Progress{Value: value, Max: 100}
	p.init(p)
	p.Style.BorderColor, p.Style.BorderWidth, p.Style.BorderRadius = Ink, BorderThin, 22
	p.Style.Background = Paper
	return p
}

// Measure implements Widget.
func (p *Progress) Measure(avail float64) geom.Size { return geom.Size{W: geom.Min(avail, 240), H: 44} }

// Paint implements Widget.
func (p *Progress) Paint(c *gfx.Canvas) {
	r := p.Bounds.Inset(geom.All(BorderThin + 3))
	fill := geom.R(r.X, r.Y, r.W*fraction(p.Value, p.Min, p.Max), r.H)
	rad := r.H / 2
	if fill.W > 0 {
		c.FillRoundRect(geom.R(fill.X, fill.Y, geom.Max(fill.W, 2*rad), fill.H), rad, Ink)
	}
	text := p.Text
	if text == "" {
		text = fmt.Sprintf("%.0f%%", fraction(p.Value, p.Min, p.Max)*100)
	}
	ts := gfx.TextStyle{Size: FontSmall, Bold: true, Color: Ink, Align: gfx.AlignCenter}
	c.Text(text, r, ts)
	c.Clip(fill, rad) // same text, inverted, only over the filled part
	ts.Color = Paper
	c.Text(text, r, ts)
	c.ResetClip()
}

// Gauge is a semicircular meter (speedometer style) with the value in the middle.
type Gauge struct {
	Base
	Value, Min, Max float64
	Unit, Label     string
	Thickness       float64
}

// NewGauge creates a gauge over [min, max].
func NewGauge(min, max, value float64, unit string) *Gauge {
	g := &Gauge{Min: min, Max: max, Value: value, Unit: unit, Thickness: 26}
	g.init(g)
	g.Style.Width = Px(380)
	return g
}

// Measure implements Widget: the semicircle plus room for the label.
func (g *Gauge) Measure(avail float64) geom.Size {
	w := geom.Min(avail, 380)
	return geom.Size{W: w, H: w*0.56 + 34}
}

// Paint implements Widget.
func (g *Gauge) Paint(c *gfx.Canvas) {
	r := g.ContentRect()
	radius := geom.Min(r.W/2, (r.H-34)) - g.Thickness/2
	ctr := geom.Point{X: r.X + r.W/2, Y: r.Y + radius + g.Thickness/2}
	frac := fraction(g.Value, g.Min, g.Max)

	c.Arc(ctr, radius, math.Pi, 2*math.Pi, g.Thickness, Mid)
	if frac > 0 {
		c.Arc(ctr, radius, math.Pi, math.Pi+math.Pi*frac, g.Thickness, g.fg(Ink))
	}
	value := fmt.Sprintf("%.0f%s", g.Value, g.Unit)
	size := fitFontSize(value, FontTitle+8, radius*1.5)
	c.Text(value, geom.R(ctr.X-radius, ctr.Y-radius*0.62, radius*2, radius*0.5), gfx.TextStyle{Size: size, Bold: true, Color: Ink, Align: gfx.AlignCenter})
	small := gfx.TextStyle{Size: FontSmall, Color: Muted}
	c.Text(fmt.Sprintf("%.0f", g.Min), geom.R(ctr.X-radius-g.Thickness/2, ctr.Y+8, 90, 30), small)
	small.Align = gfx.AlignEnd
	c.Text(fmt.Sprintf("%.0f", g.Max), geom.R(ctr.X+radius+g.Thickness/2-90, ctr.Y+8, 90, 30), small)
	if g.Label != "" {
		small.Align = gfx.AlignCenter
		c.Text(g.Label, geom.R(r.X, ctr.Y+4, r.W, 34), small)
	}
}

// fitFontSize shrinks a bold font size until text fits maxW (never below 60%).
func fitFontSize(text string, size, maxW float64) float64 {
	for s := size; s > size*0.6; s -= 2 {
		if gfx.TextWidth(text, s, true) <= maxW {
			return s
		}
	}
	return size * 0.6
}
