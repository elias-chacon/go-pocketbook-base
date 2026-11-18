// examples/custom/customwidget.go
package custom

import (
	"strconv"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
	"go-pocketbook-base/internal/toolkit/widget"
)

// Badge is a custom element: a dark pill showing a number. It only draws.
type Badge struct {
	widget.Base
	Count int
}

// NewBadge creates a badge. Every custom element calls Init with itself.
func NewBadge(count int) *Badge {
	b := &Badge{Count: count}
	b.Init(b)
	b.Style.Background = widget.Ink
	b.Style.BorderRadius = 24
	b.Style.Padding = geom.XY(6, 20)
	return b
}

// Measure reports the natural size of the content (the number).
func (b *Badge) Measure(avail float64) geom.Size {
	return geom.Size{W: gfx.TextWidth(strconv.Itoa(b.Count), widget.FontSmall, true), H: widget.FontSmall * 1.3}
}

// Paint draws the number; the pill itself comes from Style.Background.
func (b *Badge) Paint(c *gfx.Canvas) {
	c.Text(strconv.Itoa(b.Count), b.ContentRect(), gfx.TextStyle{
		Size: widget.FontSmall, Bold: true, Color: widget.Paper, Align: gfx.AlignCenter,
	})
}

// TapCounter is a custom interactive element: each tap adds one.
type TapCounter struct {
	widget.Base
	Count    int
	OnChange widget.Handler
}

// NewTapCounter creates a counter that looks like a button.
func NewTapCounter() *TapCounter {
	t := &TapCounter{}
	t.Init(t)
	t.Style.BorderColor, t.Style.BorderWidth, t.Style.BorderRadius = widget.Ink, widget.BorderThin, widget.Radius
	t.Style.Padding = geom.XY(0, 32)
	t.Style.MinHeight = widget.ControlHeight
	return t
}

func (t *TapCounter) label() string { return "Taps: " + strconv.Itoa(t.Count) }

// Measure implements widget.Widget.
func (t *TapCounter) Measure(avail float64) geom.Size {
	return geom.Size{W: gfx.TextWidth(t.label(), widget.FontBody, true), H: widget.FontBody * 1.3}
}

// Paint implements widget.Widget.
func (t *TapCounter) Paint(c *gfx.Canvas) {
	c.Text(t.label(), t.ContentRect(), gfx.TextStyle{
		Size: widget.FontBody, Bold: true, Color: widget.Ink, Align: gfx.AlignCenter,
	})
}

// HandlePointer makes the element a touch target (widget.PointerHandler).
func (t *TapCounter) HandlePointer(e *widget.PointerEvent) {
	if e.Type != widget.PointerClick {
		return
	}
	t.Count++
	widget.Fire(t.OnChange, t) // run the user's OnChange callback
}
