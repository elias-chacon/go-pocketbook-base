package presentation

import (
	"image"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
)

// RenderIcon draws the Launcher's own launcher icon (size x size): a grid of four
// tiles. focused is the pressed, inverted variant.
func RenderIcon(size int, focused bool) image.Image {
	s := float64(size)
	bg, fg := gfx.Gray(255), gfx.Gray(0)
	if focused {
		bg, fg = fg, bg
	}
	c := gfx.NewCanvas(size, size, 1)
	c.Clear(bg)
	c.StrokeRoundRect(geom.R(s*0.06, s*0.06, s*0.88, s*0.88), s*0.14, s*0.04, fg)
	for _, p := range [][2]float64{{0.22, 0.22}, {0.54, 0.22}, {0.22, 0.54}, {0.54, 0.54}} {
		c.FillRoundRect(geom.R(s*p[0], s*p[1], s*0.24, s*0.24), s*0.04, fg)
	}
	return c.Image()
}
