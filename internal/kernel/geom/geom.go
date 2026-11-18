// Package geom has the small geometry types shared by the drawing and widget
// packages. All values are float64 "design pixels" (see widget.Document).
package geom

// Point is a position.
type Point struct{ X, Y float64 }

// Size is a width and a height.
type Size struct{ W, H float64 }

// Rect is a rectangle given by its top-left corner and size.
type Rect struct{ X, Y, W, H float64 }

// Insets are per-side distances (CSS padding/margin order: top right bottom left).
type Insets struct{ Top, Right, Bottom, Left float64 }

// All returns equal insets on every side.
func All(v float64) Insets { return Insets{v, v, v, v} }

// XY returns vertical (top/bottom) and horizontal (left/right) insets.
func XY(vertical, horizontal float64) Insets {
	return Insets{vertical, horizontal, vertical, horizontal}
}

// Horizontal is Left + Right.
func (i Insets) Horizontal() float64 { return i.Left + i.Right }

// Vertical is Top + Bottom.
func (i Insets) Vertical() float64 { return i.Top + i.Bottom }

// Right is the x of the right edge.
func (r Rect) Right() float64 { return r.X + r.W }

// Bottom is the y of the bottom edge.
func (r Rect) Bottom() float64 { return r.Y + r.H }

// Center is the middle point.
func (r Rect) Center() Point { return Point{r.X + r.W/2, r.Y + r.H/2} }

// Size returns the rectangle's size.
func (r Rect) Size() Size { return Size{r.W, r.H} }

// Contains reports whether p is inside r (right and bottom edges excluded).
func (r Rect) Contains(p Point) bool {
	return p.X >= r.X && p.X < r.X+r.W && p.Y >= r.Y && p.Y < r.Y+r.H
}

// Inset shrinks r by i; the size never becomes negative.
func (r Rect) Inset(i Insets) Rect {
	return Rect{
		X: r.X + i.Left,
		Y: r.Y + i.Top,
		W: Max(0, r.W-i.Horizontal()),
		H: Max(0, r.H-i.Vertical()),
	}
}

// Max returns the larger of a and b.
func Max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// Min returns the smaller of a and b.
func Min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Clamp limits v to [lo, hi].
func Clamp(v, lo, hi float64) float64 { return Max(lo, Min(hi, v)) }

// R builds a Rect from x, y, width and height.
func R(x, y, w, h float64) Rect { return Rect{X: x, Y: y, W: w, H: h} }
