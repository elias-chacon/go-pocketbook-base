package application

import (
	"image"
	"strings"
	"unicode"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/platform/domain"
	"go-pocketbook-base/internal/toolkit/gfx"
)

// DefaultIcon draws a launcher icon for an app that has none of its own: a
// rounded tile with the initial of the title and the title under it. focused is
// the inverted variant shown while the icon is tapped.
func DefaultIcon(title string) domain.IconRenderer {
	return func(size int, focused bool) image.Image {
		s := float64(size)
		bg, fg := gfx.Gray(255), gfx.Gray(0)
		if focused {
			bg, fg = fg, bg
		}
		c := gfx.NewCanvas(size, size, 1)
		c.Clear(bg)
		c.StrokeRoundRect(geom.R(s*0.06, s*0.06, s*0.88, s*0.88), s*0.14, s*0.04, fg)
		c.Text(initial(title), geom.R(0, s*0.1, s, s*0.55), gfx.TextStyle{Size: s * 0.46, Bold: true, Color: fg, Align: gfx.AlignCenter})
		c.Text(title, geom.R(s*0.1, s*0.68, s*0.8, s*0.2), gfx.TextStyle{Size: s * 0.12, Color: fg, Align: gfx.AlignCenter, Ellipsis: true})
		return c.Image()
	}
}

// initial is the first letter of title, upper-cased ("?" for an empty title).
func initial(title string) string {
	for _, r := range strings.TrimSpace(title) {
		return string(unicode.ToUpper(r))
	}
	return "?"
}
