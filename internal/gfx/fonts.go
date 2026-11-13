package gfx

import (
	"sync"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

var (
	regularFont = mustFont(goregular.TTF)
	boldFont    = mustFont(gobold.TTF)

	faceMu    sync.Mutex
	faceCache = map[faceKey]font.Face{}
)

type faceKey struct {
	bold bool
	size int // size in quarter pixels
}

func mustFont(ttf []byte) *truetype.Font {
	f, err := truetype.Parse(ttf)
	if err != nil {
		panic(err)
	}
	return f
}

// Face returns the (cached) font face for a size in pixels.
func Face(bold bool, size float64) font.Face {
	k := faceKey{bold, int(size*4 + 0.5)}
	faceMu.Lock()
	defer faceMu.Unlock()
	if f, ok := faceCache[k]; ok {
		return f
	}
	src := regularFont
	if bold {
		src = boldFont
	}
	f := truetype.NewFace(src, &truetype.Options{Size: size})
	faceCache[k] = f
	return f
}

// TextWidth measures s (single line) at the given size, in the same units as size.
func TextWidth(s string, size float64, bold bool) float64 {
	return float64(font.MeasureString(Face(bold, size), s)) / 64
}
