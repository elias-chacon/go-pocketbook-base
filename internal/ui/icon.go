// Package ui draws the launcher icon of the application.
package ui

import (
	"image"

	"go-pocketbook-base/internal/geom"
	"go-pocketbook-base/internal/gfx"
)

// RenderIcon draws the launcher icon (size x size). When focused is true the
// colors are inverted, as shown while the icon is being tapped.
func RenderIcon(size int, focused bool) *image.Gray {
	s := float64(size)
	bg, fg := gfx.Gray(255), gfx.Gray(0)
	if focused {
		bg, fg = fg, bg
	}
	c := gfx.NewCanvas(size, size, 1)
	c.Clear(bg)
	c.StrokeRoundRect(geom.R(s*0.06, s*0.06, s*0.88, s*0.88), s*0.14, s*0.04, fg)
	c.Text("Go", geom.R(0, s*0.1, s, s*0.5), gfx.TextStyle{Size: s * 0.42, Bold: true, Color: fg, Align: gfx.AlignCenter})
	c.Line(geom.Point{X: s * 0.26, Y: s * 0.72}, geom.Point{X: s * 0.74, Y: s * 0.72}, s*0.035, fg)
	c.Text("hello", geom.R(0, s*0.74, s, s*0.18), gfx.TextStyle{Size: s * 0.12, Color: fg, Align: gfx.AlignCenter})
	return c.Image()
}
