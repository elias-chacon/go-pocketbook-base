package widget

import (
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
)

// Chip is a compact capsule: a toggleable tag, optionally with a close (×) button.
type Chip struct {
	Base
	Label    string
	Selected bool
	Closable bool
	OnChange Handler
	OnClose  Handler // when nil, closing removes the chip from its parent
}

const (
	chipH      = 72.0
	chipCloseW = 52.0
)

// NewChip creates a chip.
func NewChip(label string) *Chip {
	c := &Chip{Label: label}
	c.Init(c)
	s := &c.Style
	s.Background, s.BorderColor, s.BorderWidth, s.BorderRadius = Paper, Ink, BorderThin, chipH/2
	s.MinHeight, s.Padding = chipH, geom.XY(0, 28)
	return c
}

// Measure implements Widget.
func (c *Chip) Measure(float64) geom.Size {
	w := gfx.TextWidth(c.Label, FontBody-2, true)
	if c.Selected {
		w += 30 + 12
	}
	if c.Closable {
		w += chipCloseW - 12
	}
	return geom.Size{W: w, H: lineHeight(FontBody)}
}

func (c *Chip) closeRect() geom.Rect {
	r := c.ContentRect()
	return geom.R(r.Right()-chipCloseW+12, r.Y, chipCloseW, r.H)
}

// Paint implements Widget.
func (c *Chip) Paint(cv *gfx.Canvas) {
	r := c.ContentRect()
	color := c.fg(Ink)
	if c.Selected {
		cv.FillRoundRect(c.Bounds, c.Style.BorderRadius, c.fg(Ink))
		color = Paper
	}
	x, right := r.X, r.Right()
	if c.Selected {
		cv.Icon(gfx.IconCheck, geom.R(x, r.Y+(r.H-30)/2, 30, 30), iconStroke, color)
		x += 30 + 12
	}
	if c.Closable {
		cr := c.closeRect()
		cv.Icon(gfx.IconClose, geom.R(cr.X+8, cr.Y+(cr.H-28)/2, 28, 28), iconStroke-1, color)
		right = cr.X
	}
	cv.Text(c.Label, geom.R(x, r.Y, right-x, r.H), gfx.TextStyle{Size: FontBody - 2, Bold: true, Color: color, Ellipsis: true})
}

// HandlePointer implements PointerHandler.
func (c *Chip) HandlePointer(e *PointerEvent) {
	if e.Type != PointerClick {
		return
	}
	if c.Closable && c.closeRect().Contains(geom.Point{X: e.X, Y: e.Y}) {
		if c.OnClose != nil {
			Fire(c.OnClose, c)
		} else if c.parent != nil {
			c.parent.Elem().RemoveChild(c.self)
		}
		return
	}
	if g, ok := c.parent.(*ChipGroup); ok && g.Exclusive {
		if c.Selected {
			return // one chip stays selected in an exclusive group
		}
		g.deselectAll()
	}
	c.Selected = !c.Selected
	Fire(c.OnChange, c)
}

// ChipGroup lays chips out in wrapping rows. With Exclusive only one chip can
// be selected at a time (like radio buttons).
type ChipGroup struct {
	Panel
	Exclusive bool
}

// NewChipGroup creates a wrapping group of chips.
func NewChipGroup(exclusive bool, chips ...*Chip) *ChipGroup {
	g := &ChipGroup{Exclusive: exclusive}
	g.Init(g)
	g.Style.Display, g.Style.FlexWrap, g.Style.Gap, g.Style.AlignItems = DisplayFlex, true, 16, AlignStart
	for _, c := range chips {
		g.AppendChild(c)
	}
	return g
}

func (g *ChipGroup) deselectAll() {
	for _, k := range g.children {
		if c, ok := k.(*Chip); ok {
			c.Selected = false
		}
	}
}

// SelectedLabels returns the labels of the selected chips.
func (g *ChipGroup) SelectedLabels() []string {
	var out []string
	for _, k := range g.children {
		if c, ok := k.(*Chip); ok && c.Selected {
			out = append(out, c.Label)
		}
	}
	return out
}
