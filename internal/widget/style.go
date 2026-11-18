// Package widget is a small DOM-like component library: every element is an
// object with JavaScript-style properties (ID, ClassName, Style, Hidden,
// Disabled, Value, OnClick, OnChange…), laid out with a CSS-like flex/grid
// engine and drawn with vector graphics. The naming follows the web so that
// screens can later be described in HTML and styled with CSS.
package widget

import (
	"image"

	"go-pocketbook-base/internal/geom"
	"go-pocketbook-base/internal/gfx"
)

// Display is the CSS "display" property.
type Display int

// Display values. Block (the zero value) stacks children vertically.
const (
	DisplayBlock Display = iota
	DisplayFlex
	DisplayGrid
	DisplayNone
)

// Direction is the CSS "flex-direction" property.
type Direction int

// Flex directions.
const (
	Row Direction = iota
	Column
)

// Justify is the CSS "justify-content" property (main-axis distribution).
type Justify int

// Justify values.
const (
	JustifyStart Justify = iota
	JustifyCenter
	JustifyEnd
	JustifySpaceBetween
	JustifySpaceAround
)

// Align is the CSS "align-items" property (cross-axis alignment).
type Align int

// Align values. Stretch (the zero value) fills the cross axis.
const (
	AlignStretch Align = iota
	AlignStart
	AlignCenter
	AlignEnd
)

// Length is a CSS-like length: auto (zero value), pixels or a percentage.
type Length struct {
	V    float64
	Unit LengthUnit
}

// LengthUnit is the unit of a Length.
type LengthUnit int

// Length units.
const (
	UnitAuto LengthUnit = iota
	UnitPx
	UnitPct
)

// Px is a length in design pixels.
func Px(v float64) Length { return Length{v, UnitPx} }

// Pct is a percentage of the available width.
func Pct(v float64) Length { return Length{v, UnitPct} }

// Track is one grid column: fixed pixels or a fraction (fr) of the free space.
type Track struct{ Px, Fr float64 }

// PxTrack is a fixed-width column.
func PxTrack(v float64) Track { return Track{Px: v} }

// FrTrack is a flexible column ("1fr").
func FrTrack(v float64) Track { return Track{Fr: v} }

// Style holds the visual and layout properties of an element. Lengths use the
// CSS border-box model: Width/Height include padding and border.
type Style struct {
	Display        Display
	FlexDirection  Direction
	FlexWrap       bool
	JustifyContent Justify
	AlignItems     Align
	Gap            float64
	FlexGrow       float64
	GridColumns    []Track // grid-template-columns
	GridColumnSpan int     // grid-column: span N (0 or 1 = one column)

	Width, Height     Length
	MinWidth          float64
	MinHeight         float64
	Padding, Margin   geom.Insets
	Background, Color gfx.Color
	// BackgroundImage is drawn over Background (CSS background-image) and
	// BackgroundSize says how it fits (Cover by default).
	BackgroundImage image.Image
	BackgroundSize  gfx.Fit
	BorderColor     gfx.Color
	BorderWidth     float64
	BorderRadius    float64
	FontSize        float64
	Bold            bool
	TextAlign       gfx.TextAlign
}

// boxInsets is the border plus padding: the space between the border box and the content.
func (s *Style) boxInsets() geom.Insets {
	return geom.Insets{
		Top:    s.Padding.Top + s.BorderWidth,
		Right:  s.Padding.Right + s.BorderWidth,
		Bottom: s.Padding.Bottom + s.BorderWidth,
		Left:   s.Padding.Left + s.BorderWidth,
	}
}

// text returns the text style for the element's own font properties, falling
// back to the theme.
func (s *Style) text(defaultSize float64) gfx.TextStyle {
	size := s.FontSize
	if size == 0 {
		size = defaultSize
	}
	col := s.Color
	if col.IsZero() {
		col = Ink
	}
	return gfx.TextStyle{Size: size, Bold: s.Bold, Color: col, Align: s.TextAlign, Ellipsis: true}
}
