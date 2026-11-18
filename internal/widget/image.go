package widget

import (
	"image"

	"go-pocketbook-base/internal/geom"
	"go-pocketbook-base/internal/gfx"
)

// Image shows a bitmap (<img>): an application icon, a photo, a logo. It is
// drawn inside its content box according to Fit and, like any element, can be
// tapped (set OnClick) and sized with Style.Width/Height.
//
// Load files with gfx.LoadImage (it shrinks big pictures so they do not eat the
// reader's RAM). For a picture behind a whole panel use Style.BackgroundImage.
type Image struct {
	Base
	Source image.Image // nil draws nothing
	Fit    gfx.Fit     // FitContain by default
}

// NewImage creates an element that shows src (which may be nil for now).
func NewImage(src image.Image) *Image {
	i := &Image{Source: src, Fit: gfx.FitContain}
	i.Init(i)
	return i
}

// Measure implements Widget: the natural size, one image pixel per design
// pixel, shrunk to the available width (keeping the proportion) when wider.
func (i *Image) Measure(avail float64) geom.Size {
	if i.Source == nil {
		return geom.Size{}
	}
	b := i.Source.Bounds()
	w, h := float64(b.Dx()), float64(b.Dy())
	if w > avail && w > 0 {
		h, w = h*avail/w, avail
	}
	return geom.Size{W: w, H: h}
}

// Paint implements Widget.
func (i *Image) Paint(c *gfx.Canvas) {
	c.DrawImage(i.Source, i.ContentRect(), i.Fit)
}
