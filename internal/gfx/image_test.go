package gfx

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/image/bmp"

	"go-pocketbook-base/internal/geom"
)

// halves is a w x h image: black on the left half, white on the right half.
func halves(w, h int) *image.Gray {
	g := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x >= w/2 {
				g.SetGray(x, y, color.Gray{Y: 255})
			}
		}
	}
	return g
}

func paperCanvas(w, h int) *Canvas {
	c := NewCanvas(w, h, 1)
	c.Clear(Gray(200)) // a visible "paper": untouched pixels stay 200
	return c
}

func TestFitRects(t *testing.T) {
	dst := image.Rect(0, 0, 100, 50) // twice as wide as tall
	src, out := fitRects(40, 40, dst, FitContain)
	if src != image.Rect(0, 0, 40, 40) || out != image.Rect(25, 0, 75, 50) {
		t.Errorf("contain: %v -> %v", src, out)
	}
	src, out = fitRects(40, 40, dst, FitCover)
	if out != dst || src.Dx() != 40 || src.Dy() != 20 || src.Min.Y != 10 {
		t.Errorf("cover must crop the source's middle: %v -> %v", src, out)
	}
	src, out = fitRects(40, 40, dst, FitFill)
	if src != image.Rect(0, 0, 40, 40) || out != dst {
		t.Errorf("fill: %v -> %v", src, out)
	}
	if _, o := fitRects(0, 0, dst, FitCover); o != dst {
		t.Errorf("an empty source must not divide by zero: %v", o)
	}
}

func TestDrawImageFits(t *testing.T) {
	img := halves(40, 40)
	c := paperCanvas(100, 50)
	c.DrawImage(img, geom.R(0, 0, 100, 50), FitContain)
	g := c.Image()
	if g.GrayAt(5, 25).Y != 200 || g.GrayAt(95, 25).Y != 200 {
		t.Error("contain keeps the proportion: the sides of a wide box stay empty")
	}
	if g.GrayAt(35, 25).Y > 10 || g.GrayAt(65, 25).Y < 245 {
		t.Errorf("contain draws the image in the middle: %d %d", g.GrayAt(35, 25).Y, g.GrayAt(65, 25).Y)
	}

	c = paperCanvas(100, 50)
	c.DrawImage(img, geom.R(0, 0, 100, 50), FitCover)
	g = c.Image()
	if g.GrayAt(2, 2).Y > 10 || g.GrayAt(97, 47).Y < 245 {
		t.Error("cover fills the whole box")
	}

	c = paperCanvas(100, 50)
	c.DrawImage(img, geom.R(10, 10, 40, 20), FitFill)
	g = c.Image()
	if g.GrayAt(15, 20).Y > 10 || g.GrayAt(45, 20).Y < 245 || g.GrayAt(5, 20).Y != 200 || g.GrayAt(60, 20).Y != 200 {
		t.Error("fill stretches over exactly the rectangle")
	}
	c.DrawImage(nil, geom.R(0, 0, 10, 10), FitFill)   // no image: nothing
	c.DrawImage(img, geom.R(0, 0, 0, 10), FitFill)    // empty rectangle: nothing
	c.DrawImage(img, geom.R(500, 500, 9, 9), FitFill) // off the canvas: nothing
}

func TestDrawImageOneToOneIsACopyAndRespectsClipAndWindow(t *testing.T) {
	img := halves(40, 40)
	c := paperCanvas(100, 100)
	c.DrawImage(img, geom.R(10, 10, 40, 40), FitContain)
	if g := c.Image(); g.GrayAt(15, 20).Y != 0 || g.GrayAt(45, 20).Y != 255 {
		t.Errorf("1:1 must be an exact copy: %d %d", g.GrayAt(15, 20).Y, g.GrayAt(45, 20).Y)
	}

	c = paperCanvas(100, 100)
	c.Clip(geom.R(0, 0, 30, 100), 0)
	c.DrawImage(img, geom.R(10, 10, 40, 40), FitContain) // 1:1 through the mask
	c.DrawImage(img, geom.R(10, 60, 80, 30), FitFill)    // scaled through the mask
	c.ResetClip()
	g := c.Image()
	if g.GrayAt(15, 20).Y != 0 || g.GrayAt(45, 20).Y != 200 || g.GrayAt(15, 70).Y != 0 || g.GrayAt(60, 70).Y != 200 {
		t.Error("an image must not be drawn outside the clip")
	}

	full := paperCanvas(100, 100)
	full.DrawImage(img, geom.R(10, 10, 80, 80), FitFill)
	win := NewCanvasAt(60, 60, 1, 20, 20)
	win.Clear(Gray(200))
	win.DrawImage(img, geom.R(10, 10, 80, 80), FitFill)
	fi, wi := full.Image(), win.Image()
	for y := 0; y < 60; y++ {
		for x := 0; x < 60; x++ {
			a, b := int(fi.GrayAt(20+x, 20+y).Y), int(wi.GrayAt(x, y).Y)
			if a-b > 6 || b-a > 6 {
				t.Fatalf("window pixel (%d,%d) = %d, full canvas = %d", x, y, b, a)
			}
		}
	}
}

func TestTransparentPixelsShowWhatIsBelow(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	img.SetNRGBA(5, 5, color.NRGBA{A: 255}) // one opaque black pixel
	c := paperCanvas(10, 10)
	c.DrawImage(img, geom.R(0, 0, 10, 10), FitFill)
	g := c.Image()
	if g.GrayAt(5, 5).Y != 0 || g.GrayAt(1, 1).Y != 200 {
		t.Errorf("alpha: opaque pixel %d, transparent %d", g.GrayAt(5, 5).Y, g.GrayAt(1, 1).Y)
	}
}

func TestResizeAndDecode(t *testing.T) {
	r := Resize(halves(40, 40), 20, 10, FitContain)
	if r.Bounds() != image.Rect(0, 0, 20, 10) {
		t.Fatalf("size %v", r.Bounds())
	}
	if r.RGBAAt(0, 5).A != 0 || r.RGBAAt(10, 5).A == 0 { // letterbox is transparent, content is not
		t.Error("contain must leave transparent margins")
	}
	Resize(image.NewGray(image.Rect(0, 0, 0, 0)), 5, 5, FitCover) // an empty source is harmless

	for name, enc := range map[string]func(*bytes.Buffer, image.Image) error{
		"png": func(b *bytes.Buffer, i image.Image) error { return png.Encode(b, i) },
		"jpg": func(b *bytes.Buffer, i image.Image) error { return jpeg.Encode(b, i, nil) },
		"bmp": func(b *bytes.Buffer, i image.Image) error { return bmp.Encode(b, i) },
	} {
		var buf bytes.Buffer
		if err := enc(&buf, halves(64, 32)); err != nil {
			t.Fatal(err)
		}
		img, err := Decode(bytes.NewReader(buf.Bytes()), 0, 0)
		if err != nil || img.Bounds().Dx() != 64 {
			t.Errorf("%s: %v %v", name, img, err)
		}
		small, err := Decode(bytes.NewReader(buf.Bytes()), 32, 32) // shrunk keeping the proportion
		if err != nil || small.Bounds().Dx() != 32 || small.Bounds().Dy() != 16 {
			t.Errorf("%s shrink: %v %v", name, small.Bounds(), err)
		}
		same, _ := Decode(bytes.NewReader(buf.Bytes()), 200, 200) // already small enough: untouched
		if same.Bounds().Dx() != 64 {
			t.Errorf("%s: an image within the limit must not be resized", name)
		}
	}
	if _, err := Decode(bytes.NewReader([]byte("not an image")), 0, 0); err == nil {
		t.Error("garbage must be an error")
	}
}

func TestLoadImageFromAFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.png")
	var buf bytes.Buffer
	if err := png.Encode(&buf, halves(30, 30)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	img, err := LoadImage(path, 0, 0)
	if err != nil || img.Bounds().Dx() != 30 {
		t.Fatalf("%v %v", img, err)
	}
	if _, err := LoadImage(filepath.Join(t.TempDir(), "missing.png"), 0, 0); err == nil {
		t.Error("a missing file must be an error")
	}
}
