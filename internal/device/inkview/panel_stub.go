//go:build !(linux && arm && cgo)

package inkview

// Desktop stubs of the panel functions: there is no panel here.

// Update does nothing off-device.
func Update(mode UpdateMode, x, y, w, h int) {
	// No panel to refresh.
}

// BlitGrayMode does nothing off-device.
func BlitGrayMode(pix []byte, stride, x, y, w, h int, mode UpdateMode) {
	// No framebuffer to draw into.
}

// ColorBGR is the byte order of a 24-bit bitmap (see the device build).
var ColorBGR = true

// BlitColorMode does nothing off-device.
func BlitColorMode(rgba []byte, stride, x, y, w, h int, mode UpdateMode) {
	// No framebuffer to draw into.
}

// ExitA2 does nothing off-device.
func ExitA2() {
	// Not in A2 off-device.
}

// WaitUpdate does nothing off-device.
func WaitUpdate() {
	// Nothing is being updated off-device.
}

// HardwareDepth is unknown off-device.
func HardwareDepth() int { return 0 }

// Waveform is unknown off-device.
func Waveform() string { return "" }
