package gfx

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/gif"  // registers the GIF decoder
	_ "image/jpeg" // registers the JPEG decoder
	_ "image/png"  // registers the PNG decoder
	"io"
	"math"
	"os"

	_ "golang.org/x/image/bmp" // registers the BMP decoder
	xdraw "golang.org/x/image/draw"

	"go-pocketbook-base/internal/geom"
)

// Fit says how an image is placed in a rectangle of another shape.
type Fit int

// Image fits (the zero value is Cover, like CSS background-size: cover).
const (
	FitCover   Fit = iota // fills the rectangle, keeping the proportion; the overflow is cropped (centered)
	FitContain            // shows the whole image, keeping the proportion; the rest stays empty (centered)
	FitFill               // stretches to the rectangle, ignoring the proportion
)

// fitRects returns the part of the source to read and the device rectangle to
// write (relative to the canvas) for placing a srcW x srcH image in dst.
func fitRects(srcW, srcH int, dst image.Rectangle, fit Fit) (srcRect, dstRect image.Rectangle) {
	srcRect = image.Rect(0, 0, srcW, srcH)
	dstRect = dst
	if srcW == 0 || srcH == 0 || dst.Empty() || fit == FitFill {
		return srcRect, dstRect
	}
	sx, sy := float64(dst.Dx())/float64(srcW), float64(dst.Dy())/float64(srcH)
	switch fit {
	case FitContain:
		k := math.Min(sx, sy)
		w, h := int(math.Round(float64(srcW)*k)), int(math.Round(float64(srcH)*k))
		x, y := dst.Min.X+(dst.Dx()-w)/2, dst.Min.Y+(dst.Dy()-h)/2
		dstRect = image.Rect(x, y, x+w, y+h)
	default: // FitCover: scale up to the larger factor and crop the source
		k := math.Max(sx, sy)
		w, h := int(math.Round(float64(dst.Dx())/k)), int(math.Round(float64(dst.Dy())/k))
		x, y := (srcW-w)/2, (srcH-h)/2
		srcRect = image.Rect(x, y, x+w, y+h).Intersect(srcRect)
	}
	return srcRect, dstRect
}

// DrawImage draws img into r (design units) according to fit. Transparent
// pixels let what is below show through, the active Clip is respected, and an
// image already at its final size is copied without any resampling (scale it
// once with Resize when it is going to be drawn often, e.g. a background).
func (c *Canvas) DrawImage(img image.Image, r geom.Rect, fit Fit) {
	if img == nil || r.W <= 0 || r.H <= 0 {
		return
	}
	b := img.Bounds()
	dst := image.Rect(
		int(math.Round(c.s(r.X)-c.ox)), int(math.Round(c.s(r.Y)-c.oy)),
		int(math.Round(c.s(r.Right())-c.ox)), int(math.Round(c.s(r.Bottom())-c.oy)))
	srcRect, dstRect := fitRects(b.Dx(), b.Dy(), dst, fit)
	srcRect = srcRect.Add(b.Min)
	if srcRect.Empty() || dstRect.Intersect(c.rgba.Bounds()).Empty() {
		return
	}

	if srcRect.Dx() == dstRect.Dx() && srcRect.Dy() == dstRect.Dy() { // 1:1
		c.copyImage(img, srcRect, dstRect)
		return
	}
	var opts *xdraw.Options
	if !c.clip.Empty() {
		opts = &xdraw.Options{DstMask: c.mask}
	}
	xdraw.BiLinear.Scale(c.rgba, dstRect, img, srcRect, xdraw.Over, opts)
}

// copyImage draws src (the srcRect part) at dstRect with no resampling.
func (c *Canvas) copyImage(src image.Image, srcRect, dstRect image.Rectangle) {
	if c.clip.Empty() {
		draw.Draw(c.rgba, dstRect, src, srcRect.Min, draw.Over)
		return
	}
	draw.DrawMask(c.rgba, dstRect, src, srcRect.Min, c.mask, dstRect.Min, draw.Over)
}

// Resize returns img scaled to w x h device pixels according to fit; the result
// is an *image.RGBA of exactly that size (the empty margins of FitContain are
// transparent). Drawing it back at 1:1 is a plain copy.
func Resize(img image.Image, w, h int, fit Fit) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	b := img.Bounds()
	srcRect, dstRect := fitRects(b.Dx(), b.Dy(), out.Rect, fit)
	srcRect = srcRect.Add(b.Min)
	if !srcRect.Empty() && !dstRect.Empty() {
		xdraw.CatmullRom.Scale(out, dstRect, img, srcRect, xdraw.Src, nil)
	}
	return out
}

// Decode reads a PNG, JPEG, GIF or BMP image. When maxW and maxH are positive
// and the image is bigger it is shrunk (keeping the proportion) to fit them:
// a 12-megapixel photo would take ~48 MB of the reader's 1 GB of RAM.
func Decode(r io.Reader, maxW, maxH int) (image.Image, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("gfx: decode image: %w", err)
	}
	b := img.Bounds()
	if maxW <= 0 || maxH <= 0 || (b.Dx() <= maxW && b.Dy() <= maxH) {
		return img, nil
	}
	k := math.Min(float64(maxW)/float64(b.Dx()), float64(maxH)/float64(b.Dy()))
	return Resize(img, int(math.Max(1, math.Round(float64(b.Dx())*k))), int(math.Max(1, math.Round(float64(b.Dy())*k))), FitFill), nil
}

// LoadImage opens the file at path and decodes it (see Decode).
func LoadImage(path string, maxW, maxH int) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("gfx: %w", err)
	}
	defer f.Close()
	return Decode(f, maxW, maxH)
}
