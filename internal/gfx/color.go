// Package gfx is the vector drawing layer (a thin wrapper over fogleman/gg):
// colors, fonts, text, shapes and icons, all in design units that a Canvas
// scales to the real screen.
package gfx

import (
	"fmt"
	"strconv"
	"strings"
)

// Color is an RGBA color. The zero value is fully transparent ("no color").
type Color struct{ R, G, B, A uint8 }

// RGB builds an opaque color.
func RGB(r, g, b uint8) Color { return Color{r, g, b, 255} }

// Gray builds an opaque gray (v: 0 black … 255 white).
func Gray(v uint8) Color { return Color{v, v, v, 255} }

// IsZero reports whether the color is transparent / unset.
func (c Color) IsZero() bool { return c.A == 0 }

// Hex parses "#rgb" or "#rrggbb" (CSS notation) so styles can later come from CSS.
func Hex(s string) (Color, error) {
	h := strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	if len(h) != 6 {
		return Color{}, fmt.Errorf("color %q: want #rgb or #rrggbb", s)
	}
	v, err := strconv.ParseUint(h, 16, 32)
	if err != nil {
		return Color{}, fmt.Errorf("color %q: %w", s, err)
	}
	return RGB(uint8(v>>16), uint8(v>>8), uint8(v)), nil
}
