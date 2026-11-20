package gfx

import (
	"sync"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/goregular"
)

// Family is one of the typefaces the toolkit can draw with. The fonts are the Go
// fonts embedded in the binary, so they are the same on every reader.
type Family string

// The available families. FamilyGo is the default.
const (
	FamilyGo     Family = "go"     // Go Regular and Bold: the default
	FamilyMedium Family = "medium" // Go Medium: heavier strokes, easier on a dim e-ink screen
	FamilyMono   Family = "mono"   // Go Mono: fixed width
)

// Families lists the available families, the default first.
func Families() []Family { return []Family{FamilyGo, FamilyMedium, FamilyMono} }

// fontData is the regular and bold TrueType data of a family.
var fontData = map[Family][2][]byte{
	FamilyGo:     {goregular.TTF, gobold.TTF},
	FamilyMedium: {gomedium.TTF, gobold.TTF},
	FamilyMono:   {gomono.TTF, gomonobold.TTF},
}

var (
	faceMu    sync.Mutex
	family    = FamilyGo
	fonts     = map[Family][2]*truetype.Font{} // parsed on first use: a family nobody picks costs nothing
	faceCache = map[faceKey]font.Face{}
)

type faceKey struct {
	family Family
	bold   bool
	size   int // size in quarter pixels
}

func mustFont(ttf []byte) *truetype.Font {
	f, err := truetype.Parse(ttf)
	if err != nil {
		panic(err)
	}
	return f
}

// SetFamily chooses the typeface every text is drawn with from now on (an unknown
// family keeps the current one). Rebuild or repaint the screens after changing it:
// text that was measured with another family keeps its old size until then.
func SetFamily(f Family) {
	if _, ok := fontData[f]; !ok {
		return
	}
	faceMu.Lock()
	defer faceMu.Unlock()
	if f != family {
		family = f
		faceCache = map[faceKey]font.Face{} // the faces of the old family are not needed any more
	}
}

// CurrentFamily is the typeface in use.
func CurrentFamily() Family {
	faceMu.Lock()
	defer faceMu.Unlock()
	return family
}

// Face returns the (cached) font face for a size in pixels.
func Face(bold bool, size float64) font.Face {
	faceMu.Lock()
	defer faceMu.Unlock()
	k := faceKey{family, bold, int(size*4 + 0.5)}
	if f, ok := faceCache[k]; ok {
		return f
	}
	pair, ok := fonts[family]
	if !ok {
		data := fontData[family]
		pair = [2]*truetype.Font{mustFont(data[0]), mustFont(data[1])}
		fonts[family] = pair
	}
	src := pair[0]
	if bold {
		src = pair[1]
	}
	f := truetype.NewFace(src, &truetype.Options{Size: size})
	faceCache[k] = f
	return f
}

// TextWidth measures s (single line) at the given size, in the same units as size.
func TextWidth(s string, size float64, bold bool) float64 {
	return float64(font.MeasureString(Face(bold, size), s)) / 64
}
