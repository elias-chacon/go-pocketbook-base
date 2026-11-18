package components

import (
	"fmt"

	"go-pocketbook-base/internal/showcase"
	"go-pocketbook-base/internal/toolkit/gfx"
	"go-pocketbook-base/internal/toolkit/widget"
)

// ImageExample shows a bitmap at its natural size, in a wide box with each fit
// and as a tappable tile (an icon with OnClick). Load your own files with
// gfx.LoadImage(path, maxW, maxH), which shrinks big pictures.
func ImageExample() widget.Widget {
	icon := showcase.RenderIcon(128, false) // any image.Image works

	natural := widget.NewImage(icon)

	wide := func(fit gfx.Fit) *widget.Image {
		img := widget.NewImage(icon)
		img.Style.Width, img.Style.Height = widget.Px(150), widget.Px(90)
		img.Style.BorderColor, img.Style.BorderWidth = widget.Mid, 2
		img.Fit = fit // FitContain (default), FitCover or FitFill
		return img
	}

	tile := widget.NewImage(icon)
	tile.OnClick = func(e *widget.Event) { fmt.Println("icon tapped") }

	row := widget.NewRow(16, natural, wide(gfx.FitContain), wide(gfx.FitCover), wide(gfx.FitFill), tile)
	row.Style.FlexWrap = true // a row does not wrap by itself
	return row
}
