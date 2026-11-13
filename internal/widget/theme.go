package widget

import "go-pocketbook-base/internal/gfx"

// Design tokens. The reference screen is 1404 px wide (PB741); the Document
// scales everything for other resolutions. The palette is high-contrast
// grayscale, which is what e-ink renders best.
var (
	Paper    = gfx.Gray(255) // page / control background
	Ink      = gfx.Gray(17)  // text, borders, selected fills
	Soft     = gfx.Gray(236) // subtle surfaces (hover-less "pressed" look, zebra rows)
	Mid      = gfx.Gray(190) // disabled borders, tracks
	Muted    = gfx.Gray(96)  // secondary text, placeholders
	Disabled = gfx.Gray(150) // disabled text
)

// Metrics in design pixels.
const (
	ControlHeight = 84.0 // minimum touch target height
	Radius        = 16.0
	BorderThin    = 3.0
	FontBody      = 32.0
	FontSmall     = 26.0
	FontTitle     = 44.0
	Space         = 20.0
	MinTouch      = 64.0 // smallest touch target edge for controls drawn smaller than this
)
