// Package ui draws the application screen using vector graphics (gg).
// It is pure Go and has no dependency on the device, so it can be rendered
// and tested on a desktop (see cmd/preview).
package ui

import (
	"image"
	"image/draw"
	"math"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

var (
	regular = mustFont(goregular.TTF)
	bold    = mustFont(gobold.TTF)
)

func mustFont(ttf []byte) *truetype.Font {
	f, err := truetype.Parse(ttf)
	if err != nil {
		panic(err)
	}
	return f
}

func face(f *truetype.Font, size float64) font.Face {
	return truetype.NewFace(f, &truetype.Options{Size: size})
}

// ExitButton returns the on-screen rectangle of the "Exit" button for a
// screen of w x h pixels. The app uses it for touch hit-testing.
func ExitButton(w, h int) image.Rectangle {
	u := unit(w, h)
	bw, bh := int(24*u), int(8*u)
	cx := w / 2
	y := h - int(10*u) - bh
	return image.Rect(cx-bw/2, y, cx+bw/2, y+bh)
}

// unit is the base length (1% of the shorter side) so the layout scales
// across PocketBook resolutions.
func unit(w, h int) float64 {
	if w < h {
		return float64(w) / 100
	}
	return float64(h) / 100
}

// Render draws the main panel and returns an 8-bit grayscale image. A
// non-empty status is shown as a small line under the subtitle.
func Render(w, h int, status string) *image.Gray {
	u := unit(w, h)
	dc := gg.NewContext(w, h)

	// Background.
	dc.SetRGB(1, 1, 1)
	dc.Clear()

	// Panel with border.
	m := 6 * u
	dc.SetRGB(0, 0, 0)
	dc.SetLineWidth(0.7 * u)
	dc.DrawRoundedRectangle(m, m, float64(w)-2*m, float64(h)-2*m, 3*u)
	dc.Stroke()

	// Title bar.
	barH := 9 * u
	dc.DrawRoundedRectangle(m, m, float64(w)-2*m, barH, 3*u)
	dc.Fill()
	dc.DrawRectangle(m, m+barH/2, float64(w)-2*m, barH/2)
	dc.Fill()
	dc.SetRGB(1, 1, 1)
	dc.SetFontFace(face(bold, 3.6*u))
	dc.DrawStringAnchored("go-pocketbook-base", float64(w)/2, m+barH/2, 0.5, 0.35)

	// Vector logo: concentric circles with a check-like path.
	cx, cy, r := float64(w)/2, float64(h)*0.33, 16*u
	dc.SetRGB(0, 0, 0)
	dc.DrawCircle(cx, cy, r)
	dc.Fill()
	dc.SetRGB(1, 1, 1)
	dc.DrawCircle(cx, cy, r*0.82)
	dc.Fill()
	dc.SetRGB(0, 0, 0)
	dc.DrawCircle(cx, cy, r*0.64)
	dc.Fill()
	// eyes
	dc.SetRGB(1, 1, 1)
	for _, dx := range []float64{-0.25, 0.25} {
		dc.DrawCircle(cx+dx*r, cy-0.12*r, 0.17*r)
		dc.Fill()
	}
	dc.SetRGB(0, 0, 0)
	for _, dx := range []float64{-0.25, 0.25} {
		dc.DrawCircle(cx+dx*r+0.03*r, cy-0.12*r, 0.07*r)
		dc.Fill()
	}
	// smile (arc)
	dc.SetRGB(1, 1, 1)
	dc.SetLineWidth(0.35 * u)
	dc.DrawArc(cx, cy+0.05*r, 0.3*r, math.Pi*0.15, math.Pi*0.85)
	dc.Stroke()

	// Main text.
	dc.SetRGB(0, 0, 0)
	dc.SetFontFace(face(bold, 9*u))
	dc.DrawStringAnchored("Hello, Golang!", float64(w)/2, float64(h)*0.58, 0.5, 0.5)
	dc.SetFontFace(face(regular, 3.8*u))
	dc.DrawStringAnchored("Vector UI drawn in pure Go", float64(w)/2, float64(h)*0.58+9*u, 0.5, 0.5)

	if status != "" {
		dc.SetFontFace(face(regular, 3.2*u))
		dc.DrawStringWrapped(status, float64(w)/2, float64(h)*0.58+16*u, 0.5, 0, float64(w)-24*u, 1.3, gg.AlignCenter)
	}

	// Exit button.
	b := ExitButton(w, h)
	dc.SetLineWidth(0.6 * u)
	dc.DrawRoundedRectangle(float64(b.Min.X), float64(b.Min.Y), float64(b.Dx()), float64(b.Dy()), 1.5*u)
	dc.Stroke()
	dc.SetFontFace(face(bold, 4.2*u))
	dc.DrawStringAnchored("Exit", float64(b.Min.X+b.Max.X)/2, float64(b.Min.Y+b.Max.Y)/2, 0.5, 0.35)

	return toGray(dc.Image())
}

// RenderIcon draws the launcher icon (size x size). When focused is true
// the colors are inverted, as shown while the icon is tapped.
func RenderIcon(size int, focused bool) *image.Gray {
	s := float64(size)
	dc := gg.NewContext(size, size)
	bg, fg := 1.0, 0.0
	if focused {
		bg, fg = 0, 1
	}
	dc.SetRGB(bg, bg, bg)
	dc.Clear()
	dc.SetRGB(fg, fg, fg)
	dc.SetLineWidth(s * 0.04)
	dc.DrawRoundedRectangle(s*0.06, s*0.06, s*0.88, s*0.88, s*0.14)
	dc.Stroke()
	dc.SetFontFace(face(bold, s*0.42))
	dc.DrawStringAnchored("Go", s/2, s*0.46, 0.5, 0.4)
	dc.SetLineWidth(s * 0.035)
	dc.DrawLine(s*0.26, s*0.72, s*0.74, s*0.72)
	dc.Stroke()
	dc.SetFontFace(face(regular, s*0.12))
	dc.DrawStringAnchored("hello", s/2, s*0.83, 0.5, 0.35)
	return toGray(dc.Image())
}

func toGray(src image.Image) *image.Gray {
	g := image.NewGray(src.Bounds())
	draw.Draw(g, g.Bounds(), src, src.Bounds().Min, draw.Src)
	return g
}
