package widget

import (
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
)

// Variant is the visual weight of a Button.
type Variant string

// Button variants.
const (
	VariantPrimary   Variant = "primary"   // filled
	VariantSecondary Variant = "secondary" // outlined (default)
	VariantGhost     Variant = "ghost"     // text only
)

// Button is a tappable <button> with a label and an optional icon.
type Button struct {
	Base
	Text string // JS: textContent
	Icon gfx.IconKind
}

// NewButton creates an outlined button.
func NewButton(text string) *Button {
	b := &Button{Text: text}
	b.Init(b)
	b.Style.BorderRadius = Radius
	b.Style.MinHeight = ControlHeight
	b.Style.Padding = geom.XY(0, 32)
	b.Style.Bold = true
	b.Style.TextAlign = gfx.AlignCenter // Style.TextAlign places the label: AlignStart for the rows of a list
	b.SetVariant(VariantSecondary)
	return b
}

// NewIconButton creates a square button showing only an icon.
func NewIconButton(icon gfx.IconKind) *Button {
	b := NewButton("")
	b.Icon = icon
	b.Style.Padding = geom.All(0)
	b.Style.MinWidth = ControlHeight
	return b
}

// SetVariant applies the look of a variant (like switching a CSS class).
func (b *Button) SetVariant(v Variant) {
	b.ClassName = string(v)
	s := &b.Style
	switch v {
	case VariantPrimary:
		s.Background, s.Color, s.BorderColor, s.BorderWidth = Ink, Paper, Ink, BorderThin
	case VariantGhost:
		s.Background, s.Color, s.BorderColor, s.BorderWidth = gfx.Color{}, Ink, gfx.Color{}, 0
	default:
		s.Background, s.Color, s.BorderColor, s.BorderWidth = Paper, Ink, Ink, BorderThin
	}
}

const (
	iconSize = 34.0
	iconGap  = 14.0
)

// Measure implements Widget.
func (b *Button) Measure(float64) geom.Size {
	ts := b.Style.text(FontBody)
	w := 0.0
	if b.Text != "" {
		w = gfx.TextWidth(b.Text, ts.Size, ts.Bold)
	}
	if b.Icon != gfx.IconNone {
		w += iconSize
		if b.Text != "" {
			w += iconGap
		}
	}
	return geom.Size{W: w, H: lineHeight(ts.Size)}
}

// Paint implements Widget: icon and label in the content box, centered unless Style.TextAlign says
// AlignStart (left) or AlignEnd (right).
func (b *Button) Paint(c *gfx.Canvas) {
	r := b.ContentRect()
	ts := b.Style.text(FontBody)
	ts.Color = b.fg(ts.Color)
	if b.Disabled && b.Style.Background == Ink {
		ts.Color = Paper
	}
	total := b.Measure(r.W).W
	x := r.X + (r.W-total)/2
	switch b.Style.TextAlign {
	case gfx.AlignStart:
		x = r.X
	case gfx.AlignEnd:
		x = r.X + r.W - total
	}
	if b.Icon != gfx.IconNone {
		c.Icon(b.Icon, geom.R(x, r.Y+(r.H-iconSize)/2, iconSize, iconSize), iconStroke, ts.Color)
		x += iconSize + iconGap
	}
	if b.Text != "" {
		ts.Align = gfx.AlignStart
		c.Text(b.Text, geom.R(x, r.Y, r.X+r.W-x, r.H), ts)
	}
}

// HandlePointer makes the button a touch target; use OnClick for the action.
func (b *Button) HandlePointer(*PointerEvent) {
	// Intentionally empty: implementing PointerHandler is what makes the button
	// a touch target; the Document runs OnClick.
}
