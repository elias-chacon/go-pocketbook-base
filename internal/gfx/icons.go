package gfx

import "go-pocketbook-base/internal/geom"

// IconKind selects a vector icon.
type IconKind int

// Available icons.
const (
	IconNone IconKind = iota
	IconCheck
	IconChevronDown
	IconChevronUp
	IconChevronLeft
	IconChevronRight
	IconPlus
	IconMinus
	IconClose
	IconMenu
	IconSearch
	IconCalendar
)

// pt is a point in the unit square of an icon (0..1).
type pt struct{ u, v float64 }

// strokes are polylines in unit-square coordinates, one per icon.
var iconStrokes = map[IconKind][][]pt{
	IconCheck:        {{{0.2, 0.55}, {0.42, 0.75}, {0.8, 0.28}}},
	IconChevronDown:  {{{0.2, 0.35}, {0.5, 0.65}, {0.8, 0.35}}},
	IconChevronUp:    {{{0.2, 0.65}, {0.5, 0.35}, {0.8, 0.65}}},
	IconChevronLeft:  {{{0.62, 0.2}, {0.35, 0.5}, {0.62, 0.8}}},
	IconChevronRight: {{{0.38, 0.2}, {0.65, 0.5}, {0.38, 0.8}}},
	IconPlus:         {{{0.5, 0.2}, {0.5, 0.8}}, {{0.2, 0.5}, {0.8, 0.5}}},
	IconMinus:        {{{0.2, 0.5}, {0.8, 0.5}}},
	IconClose:        {{{0.25, 0.25}, {0.75, 0.75}}, {{0.75, 0.25}, {0.25, 0.75}}},
	IconMenu:         {{{0.2, 0.28}, {0.8, 0.28}}, {{0.2, 0.5}, {0.8, 0.5}}, {{0.2, 0.72}, {0.8, 0.72}}},
	IconSearch:       {{{0.62, 0.62}, {0.86, 0.86}}},
	IconCalendar:     {{{0.15, 0.4}, {0.85, 0.4}}, {{0.32, 0.12}, {0.32, 0.3}}, {{0.68, 0.12}, {0.68, 0.3}}},
}

// Icon draws kind centered in r, as a square of the shorter side, with the
// given stroke width (in design units).
func (c *Canvas) Icon(kind IconKind, r geom.Rect, width float64, col Color) {
	side := geom.Min(r.W, r.H)
	x0, y0 := r.X+(r.W-side)/2, r.Y+(r.H-side)/2
	at := func(p pt) geom.Point { return geom.Point{X: x0 + p.u*side, Y: y0 + p.v*side} }

	for _, stroke := range iconStrokes[kind] {
		pts := make([]geom.Point, len(stroke))
		for i, p := range stroke {
			pts[i] = at(p)
		}
		c.Polyline(pts, width, col)
	}
	// Icons that also contain a closed shape.
	switch kind {
	case IconSearch:
		c.StrokeCircle(at(pt{0.43, 0.43}), 0.27*side+width/2, width, col)
	case IconCalendar:
		c.StrokeRoundRect(geom.Rect{X: x0 + 0.12*side, Y: y0 + 0.2*side, W: 0.76 * side, H: 0.68 * side}, 0.08*side, width, col)
	}
}
