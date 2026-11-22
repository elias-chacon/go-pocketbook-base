// Package engineering is the content of the engineering app (apps/eng): a menu of tests of the
// reader's panel and touch, like the engineering menu of a phone. The tests show color and screen
// patterns, compare the update modes of InkView, time a scroll and a touch trail, and describe the
// device. Every measure is written to a log file (one JSON object per line) so it can be read later
// from the USB drive and analysed.
//
// The package knows nothing of InkView: the panel is the Panel port, which the app's composition
// root implements with the platform's panel (and a fake in the tests).
package engineering

import (
	"image"
)

// Mode is how a part of the panel is refreshed after it was drawn (the update functions of
// InkView). The numbers are those of the platform's UpdateMode.
type Mode int

// The update modes, from the default one to the fast ones that trade quality for speed.
const (
	Partial Mode = iota
	Dynamic
	A2
	DU4
	BW
	HQ
	Full
	modes
)

// Modes is how many update modes there are.
const Modes = int(modes)

var modeNames = [...]string{"partial", "dynamic", "a2", "du4", "bw", "hq", "full"}

// String is the short name of the mode, the one the log uses.
func (m Mode) String() string {
	if m < 0 || int(m) >= len(modeNames) {
		return "unknown"
	}
	return modeNames[m]
}

// Next is the mode after m, in order, coming back to the first one.
func (m Mode) Next() Mode { return (m + 1) % modes }

// Info is what the reader says about its panel.
type Info struct {
	Model    string
	Depth    int // bits per pixel: 8 gray, 24 color
	Waveform string
	Canvas   string
}

// Panel is the reader's display, as the tests need it.
type Panel interface {
	Info() Info
	// DrawGray draws the part r of a grayscale image and refreshes it with the mode.
	DrawGray(img *image.Gray, r image.Rectangle, m Mode)
	// DrawColor draws the part r of a color image and refreshes it with the mode.
	DrawColor(img *image.RGBA, r image.Rectangle, m Mode)
	// SetMode is the mode the platform's own repaints use from now on.
	SetMode(m Mode)
	// Wait blocks until the panel finished the updates asked for: the draw calls return before
	// the panel is done, so this is what tells how long the refresh really takes.
	Wait()
	// Settle leaves the fast (A2) state with a clean update and waits for the panel.
	Settle()
}

// Fact is one line of the system page: a key the screen translates ("sys.<key>") and its value.
type Fact struct{ Key, Value string }
