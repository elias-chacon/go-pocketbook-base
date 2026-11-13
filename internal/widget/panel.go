package widget

import "go-pocketbook-base/internal/geom"

// Panel is the generic container (a <div>): lay it out with Style.Display.
type Panel struct{ Base }

// NewPanel creates a block container with the given children.
func NewPanel(children ...Widget) *Panel {
	p := &Panel{}
	p.Init(p)
	for _, c := range children {
		p.AppendChild(c)
	}
	return p
}

// NewRow creates a flex row (align-items: center) with the given gap.
func NewRow(gap float64, children ...Widget) *Panel {
	p := NewPanel(children...)
	p.Style.Display, p.Style.FlexDirection, p.Style.AlignItems, p.Style.Gap = DisplayFlex, Row, AlignCenter, gap
	return p
}

// NewColumn creates a flex column with the given gap.
func NewColumn(gap float64, children ...Widget) *Panel {
	p := NewPanel(children...)
	p.Style.Display, p.Style.FlexDirection, p.Style.Gap = DisplayFlex, Column, gap
	return p
}

// NewGrid creates a grid with the given columns and gap.
func NewGrid(gap float64, columns []Track, children ...Widget) *Panel {
	p := NewPanel(children...)
	p.Style.Display, p.Style.GridColumns, p.Style.Gap = DisplayGrid, columns, gap
	return p
}

// NewCard creates a bordered, rounded surface with a title and optional
// subtitle followed by the given content.
func NewCard(title, subtitle string, content ...Widget) *Panel {
	c := NewColumn(Space)
	s := &c.Style
	s.BorderWidth, s.BorderColor, s.BorderRadius = BorderThin, Ink, Radius+4
	s.Padding, s.Background = geom.All(28), Paper
	c.ClassName = "card"
	if title != "" {
		t := NewLabel(title)
		t.Style.FontSize, t.Style.Bold = FontBody+6, true
		c.AppendChild(t)
	}
	if subtitle != "" {
		st := NewLabel(subtitle)
		st.Style.Color, st.Style.FontSize = Muted, FontSmall
		c.AppendChild(st)
	}
	for _, w := range content {
		c.AppendChild(w)
	}
	return c
}
