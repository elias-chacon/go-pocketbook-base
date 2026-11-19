package application

import (
	"bytes"
	"image"
	"image/color"
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

// IconFromImage makes an icon renderer from a picture (the icon an app ships as
// icon.png). The picture is scaled to fit the requested size, laid on white and
// turned into the 8-bit gray the home screen wants; focused is its negative.
func IconFromImage(img image.Image) domain.IconRenderer {
	return func(size int, focused bool) image.Image {
		rgba := gfx.Resize(img, size, size, gfx.FitContain)
		out := image.NewGray(image.Rect(0, 0, size, size))
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				c := rgba.RGBAAt(x, y)
				white := 255 - int(c.A) // the transparent part shows the white under it
				v := (299*(int(c.R)+white) + 587*(int(c.G)+white) + 114*(int(c.B)+white)) / 1000
				if v > 255 {
					v = 255
				}
				if focused {
					v = 255 - v
				}
				out.SetGray(x, y, color.Gray{Y: uint8(v)})
			}
		}
		return out
	}
}

// maxIconEdge bounds a decoded icon: an icon is a few dozen kilobytes, not a photo.
const maxIconEdge = 512

// IconFromPNG is IconFromImage for the bytes of an icon.png (or any image format).
func IconFromPNG(data []byte) (domain.IconRenderer, error) {
	img, err := gfx.Decode(bytes.NewReader(data), maxIconEdge, maxIconEdge)
	if err != nil {
		return nil, err
	}
	return IconFromImage(img), nil
}
