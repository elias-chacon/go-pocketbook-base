//go:build !(linux && arm && cgo)

package inkview

import "errors"

// ErrNotOnDevice is returned by Run when built for a non-PocketBook target.
var ErrNotOnDevice = errors.New("inkview: only available when built for PocketBook (linux/arm + cgo); use cmd/preview on desktop")

// Desktop stubs: there is no InkView here, so these do nothing (they exist so
// the packages that use them compile and can be tested on any machine).

// Run always fails off-device.
func Run(App) error { return ErrNotOnDevice }

// ScreenSize is unknown off-device.
func ScreenSize() (w, h int) { return 0, 0 }

// Exit does nothing off-device.
func Exit() {
	// No event loop to stop.
}

// Repaint does nothing off-device.
func Repaint() {
	// No screen to refresh.
}

// Info is empty off-device.
func Info() string { return "" }

// DisablePanel does nothing off-device.
func DisablePanel() {
	// No system panel on the desktop.
}

// Blit does nothing off-device.
func Blit(pix []byte, w, h, stride int) {
	// No framebuffer to draw into.
}

// BlitRect does nothing off-device.
func BlitRect(pix []byte, stride, x, y, w, h int) {
	// No framebuffer to draw into.
}

// OpenKeyboard does nothing off-device (tests use widget.FakeKeyboard).
func OpenKeyboard(title, initial string, maxLen, flags int, done func(string, bool)) {
	// No system keyboard on the desktop.
}
