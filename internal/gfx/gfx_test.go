package gfx

import (
	"image"
	"testing"

	"go-pocketbook-base/internal/geom"
)

func TestHex(t *testing.T) {
	good := map[string]Color{
		"#fff": RGB(255, 255, 255), "#000000": RGB(0, 0, 0), " #1a2b3c ": RGB(0x1a, 0x2b, 0x3c), "abc": RGB(0xaa, 0xbb, 0xcc),
	}
	for in, want := range good {
		if got, err := Hex(in); err != nil || got != want {
			t.Errorf("Hex(%q) = %+v, %v; want %+v", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "#12", "#gggggg", "#12345"} {
		if _, err := Hex(bad); err == nil {
			t.Errorf("Hex(%q) succeeded", bad)
		}
	}
	if !(Color{}).IsZero() || Gray(0).IsZero() {
		t.Error("IsZero: transparent is zero, black is not")
	}
}

func TestTextWidthGrowsWithTextAndSize(t *testing.T) {
	a, b := TextWidth("abc", 30, false), TextWidth("abcdef", 30, false)
	if !(b > a && a > 0) {
		t.Errorf("widths %v %v", a, b)
	}
	if TextWidth("abc", 60, false) <= a {
		t.Error("bigger font must be wider")
	}
	if TextWidth("abc", 30, true) <= a {
		t.Error("bold is wider than regular")
	}
}

func inked(img *image.Gray, r geom.Rect) bool {
	for y := int(r.Y); y < int(r.Bottom()); y++ {
		for x := int(r.X); x < int(r.Right()); x++ {
			if img.GrayAt(x, y).Y < 128 {
				return true
			}
		}
	}
	return false
}

func TestCanvasDrawsAndScales(t *testing.T) {
	for _, scale := range []float64{1, 0.5} {
		c := NewCanvas(int(200*scale), int(100*scale), scale)
		c.Clear(Gray(255))
		c.FillRoundRect(geom.R(20, 20, 60, 60), 8, Gray(0)) // design units: scaled on screen
		img := c.Image()
		inside := geom.Point{X: 50 * scale, Y: 50 * scale}
		if img.GrayAt(int(inside.X), int(inside.Y)).Y != 0 {
			t.Errorf("scale %v: the filled square must cover its center", scale)
		}
		if img.GrayAt(int(150*scale), int(50*scale)).Y != 255 {
			t.Errorf("scale %v: outside must stay white", scale)
		}
	}
}

func TestTextEllipsisAndClip(t *testing.T) {
	c := NewCanvas(300, 60, 1)
	c.Clear(Gray(255))
	c.Text("a very long text that cannot fit", geom.R(0, 0, 100, 60), TextStyle{Size: 30, Color: Gray(0), Ellipsis: true})
	img := c.Image()
	if !inked(img, geom.R(0, 0, 100, 60)) {
		t.Error("text must be drawn")
	}
	if inked(img, geom.R(105, 0, 195, 60)) {
		t.Error("ellipsized text must stay inside its box")
	}

	c2 := NewCanvas(100, 100, 1)
	c2.Clear(Gray(255))
	c2.Clip(geom.R(0, 0, 50, 100), 0)
	c2.FillRoundRect(geom.R(0, 0, 100, 100), 0, Gray(0))
	c2.ResetClip()
	img2 := c2.Image()
	if img2.GrayAt(25, 50).Y != 0 || img2.GrayAt(75, 50).Y != 255 {
		t.Error("Clip must restrict drawing")
	}
	if got := ellipsize("hello", 1, Face(false, 30)); got != "" {
		t.Errorf("nothing fits in 1px: %q", got)
	}
}

func TestTextUnderClipIsCutAndClipIsReusable(t *testing.T) {
	c := NewCanvas(200, 60, 1)
	for round := 0; round < 2; round++ { // the second round proves ResetClip left no stale mask
		c.Clear(Gray(255))
		c.Clip(geom.R(0, 0, 100, 60), 0)
		c.Text("WWWWWWWWWW", geom.R(0, 0, 200, 60), TextStyle{Size: 30, Color: Gray(0)})
		c.ResetClip()
		img := c.Image()
		if !inked(img, geom.R(0, 0, 100, 60)) {
			t.Fatalf("round %d: text must show inside the clip", round)
		}
		if inked(img, geom.R(101, 0, 99, 60)) {
			t.Fatalf("round %d: text must be cut at the clip edge", round)
		}
		c.Text("W", geom.R(150, 0, 50, 60), TextStyle{Size: 30, Color: Gray(0)}) // no clip any more
		if !inked(c.Image(), geom.R(150, 0, 50, 60)) {
			t.Fatalf("round %d: drawing after ResetClip must not be masked", round)
		}
	}
}

func TestEveryIconDrawsSomething(t *testing.T) {
	kinds := []IconKind{IconCheck, IconChevronDown, IconChevronUp, IconChevronLeft, IconChevronRight,
		IconPlus, IconMinus, IconClose, IconMenu, IconSearch, IconCalendar}
	for _, k := range kinds {
		c := NewCanvas(60, 60, 1)
		c.Clear(Gray(255))
		c.Icon(k, geom.R(0, 0, 60, 60), 4, Gray(0))
		if !inked(c.Image(), geom.R(0, 0, 60, 60)) {
			t.Errorf("icon %d drew nothing", k)
		}
	}
	c := NewCanvas(10, 10, 1)
	c.Clear(Gray(255))
	c.Icon(IconNone, geom.R(0, 0, 10, 10), 2, Gray(0))
	if inked(c.Image(), geom.R(0, 0, 10, 10)) {
		t.Error("IconNone must draw nothing")
	}
}

func TestShapesIgnoreTransparentColors(t *testing.T) {
	c := NewCanvas(40, 40, 1)
	c.Clear(Gray(255))
	c.FillRoundRect(geom.R(0, 0, 40, 40), 0, Color{})
	c.StrokeRoundRect(geom.R(0, 0, 40, 40), 0, 4, Color{})
	c.FillCircle(geom.Point{X: 20, Y: 20}, 10, Color{})
	c.Arc(geom.Point{X: 20, Y: 20}, 10, 0, 3, 4, Color{})
	if inked(c.Image(), geom.R(0, 0, 40, 40)) {
		t.Error("a transparent color must not draw")
	}
}
