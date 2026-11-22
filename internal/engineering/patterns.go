package engineering

import (
	"image"
	"image/color"
	"math"
)

// Pattern is a full-screen test picture. Its ID names it in the log and in the texts ("pat.<id>").
type Pattern struct {
	ID   string
	Draw func(w, h int) *image.RGBA
}

// ColorPatterns are the pictures that test color: bars, primaries, gray steps and ramps, hues.
func ColorPatterns() []Pattern {
	return []Pattern{
		{"bars", bars}, {"primaries", primaries}, {"gray16", gray16}, {"gray256", gray256},
		{"hue", hue}, {"pastel", pastel}, {"white", solid(255, 255, 255)}, {"black", solid(0, 0, 0)},
	}
}

// ScreenPatterns are the pictures that test the panel itself: fills, fine lines and checkers (dead
// pixels, ghosting, the refresh), a grid with the edges and diagonals (geometry).
func ScreenPatterns() []Pattern {
	return []Pattern{
		{"white", solid(255, 255, 255)}, {"black", solid(0, 0, 0)}, {"gray50", solid(128, 128, 128)},
		{"checker1", checker(1)}, {"checker8", checker(8)}, {"hlines", lines(true)}, {"vlines", lines(false)},
		{"grid", grid},
	}
}

func newImage(w, h int) *image.RGBA { return image.NewRGBA(image.Rect(0, 0, w, h)) }

func put(img *image.RGBA, x, y int, c color.RGBA) {
	o := img.PixOffset(x, y)
	img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = c.R, c.G, c.B, 255
}

// fill paints the rectangle [x0,x1) x [y0,y1) with c.
func fill(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	b := img.Bounds()
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > b.Max.X {
		x1 = b.Max.X
	}
	if y1 > b.Max.Y {
		y1 = b.Max.Y
	}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			put(img, x, y, c)
		}
	}
}

func solid(r, g, b uint8) func(w, h int) *image.RGBA {
	return func(w, h int) *image.RGBA {
		img := newImage(w, h)
		fill(img, 0, 0, w, h, color.RGBA{r, g, b, 255})
		return img
	}
}

// bars is the classic color bars (white, yellow, cyan, green, magenta, red, blue, black) over a
// strip of sixteen grays.
func bars(w, h int) *image.RGBA {
	img := newImage(w, h)
	colors := []color.RGBA{
		{255, 255, 255, 255}, {255, 255, 0, 255}, {0, 255, 255, 255}, {0, 255, 0, 255},
		{255, 0, 255, 255}, {255, 0, 0, 255}, {0, 0, 255, 255}, {0, 0, 0, 255},
	}
	top := h * 7 / 10
	for i, c := range colors {
		fill(img, w*i/len(colors), 0, w*(i+1)/len(colors), top, c)
	}
	for i := 0; i < 16; i++ {
		v := uint8(i * 255 / 15)
		fill(img, w*i/16, top, w*(i+1)/16, h, color.RGBA{v, v, v, 255})
	}
	return img
}

// primaries are eight horizontal bands: red, green, blue, cyan, magenta, yellow, black, white.
func primaries(w, h int) *image.RGBA {
	img := newImage(w, h)
	colors := []color.RGBA{
		{255, 0, 0, 255}, {0, 255, 0, 255}, {0, 0, 255, 255}, {0, 255, 255, 255},
		{255, 0, 255, 255}, {255, 255, 0, 255}, {0, 0, 0, 255}, {255, 255, 255, 255},
	}
	for i, c := range colors {
		fill(img, 0, h*i/len(colors), w, h*(i+1)/len(colors), c)
	}
	return img
}

// gray16 is sixteen vertical bands from black to white: the gray levels the panel tells apart.
func gray16(w, h int) *image.RGBA {
	img := newImage(w, h)
	for i := 0; i < 16; i++ {
		v := uint8(i * 255 / 15)
		fill(img, w*i/16, 0, w*(i+1)/16, h, color.RGBA{v, v, v, 255})
	}
	return img
}

// gray256 is a smooth ramp from black to white across the screen (banding, dithering).
func gray256(w, h int) *image.RGBA {
	img := newImage(w, h)
	for x := 0; x < w; x++ {
		v := uint8(x * 255 / (w - 1))
		fill(img, x, 0, x+1, h, color.RGBA{v, v, v, 255})
	}
	return img
}

// hue sweeps the hue across the screen, bright at the top and dark at the bottom.
func hue(w, h int) *image.RGBA {
	img := newImage(w, h)
	for y := 0; y < h; y++ {
		v := 1 - float64(y)/float64(h-1)*0.9
		for x := 0; x < w; x++ {
			put(img, x, y, hsv(float64(x)/float64(w)*360, 1, v))
		}
	}
	return img
}

// pastel sweeps the hue across the screen with the saturation growing downward: from white to the
// pure color, how the panel renders weak colors.
func pastel(w, h int) *image.RGBA {
	img := newImage(w, h)
	for y := 0; y < h; y++ {
		s := float64(y) / float64(h-1)
		for x := 0; x < w; x++ {
			put(img, x, y, hsv(float64(x)/float64(w)*360, s, 1))
		}
	}
	return img
}

// hsv converts a hue (degrees), a saturation and a value (0..1) to RGB.
func hsv(h, s, v float64) color.RGBA {
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := v - c
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	return color.RGBA{uint8((r + m) * 255), uint8((g + m) * 255), uint8((b + m) * 255), 255}
}

// checker is a black and white chessboard of squares of size pixels.
func checker(size int) func(w, h int) *image.RGBA {
	return func(w, h int) *image.RGBA {
		img := newImage(w, h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				v := uint8(255)
				if (x/size+y/size)%2 == 0 {
					v = 0
				}
				put(img, x, y, color.RGBA{v, v, v, 255})
			}
		}
		return img
	}
}

// lines is alternating one-pixel black and white lines, horizontal or vertical.
func lines(horizontal bool) func(w, h int) *image.RGBA {
	return func(w, h int) *image.RGBA {
		img := newImage(w, h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				i := x
				if horizontal {
					i = y
				}
				v := uint8(255)
				if i%2 == 0 {
					v = 0
				}
				put(img, x, y, color.RGBA{v, v, v, 255})
			}
		}
		return img
	}
}

// grid is a white page with a grid every 100 pixels, a frame on the very edge, the two diagonals
// and a square in each corner: to see the real size of the drawing area and straight lines.
func grid(w, h int) *image.RGBA {
	img := newImage(w, h)
	fill(img, 0, 0, w, h, color.RGBA{255, 255, 255, 255})
	black := color.RGBA{0, 0, 0, 255}
	gray := color.RGBA{160, 160, 160, 255}
	for x := 0; x < w; x += 100 {
		fill(img, x, 0, x+1, h, gray)
	}
	for y := 0; y < h; y += 100 {
		fill(img, 0, y, w, y+1, gray)
	}
	fill(img, 0, 0, w, 3, black)
	fill(img, 0, h-3, w, h, black)
	fill(img, 0, 0, 3, h, black)
	fill(img, w-3, 0, w, h, black)
	for x := 0; x < w; x++ {
		y := x * (h - 1) / (w - 1)
		fill(img, x, y, x+2, y+2, black)
		fill(img, x, h-1-y-1, x+2, h-1-y+1, black)
	}
	const sq = 40
	for _, c := range [][2]int{{0, 0}, {w - sq, 0}, {0, h - sq}, {w - sq, h - sq}} {
		fill(img, c[0], c[1], c[0]+sq, c[1]+sq, black)
	}
	return img
}
