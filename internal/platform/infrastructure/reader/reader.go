// Package reader adapts the platform to the PocketBook e-reader: it implements
// the Screen port and the system keyboard with InkView, translates InkView's
// events into the runtime's, and registers apps on the home screen. It is the
// only part of the platform that knows the device exists.
package reader

import (
	"image"
	"path/filepath"

	"go-pocketbook-base/internal/device/desktop"
	"go-pocketbook-base/internal/device/inkview"
	"go-pocketbook-base/internal/platform/application"
	"go-pocketbook-base/internal/platform/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

// KeyOf names an InkView key code.
func KeyOf(code int) application.Key {
	switch code {
	case inkview.KeyBack:
		return application.KeyBack
	case inkview.KeyPrev:
		return application.KeyPrevious
	case inkview.KeyNext:
		return application.KeyNext
	case inkview.KeyMenu:
		return application.KeyMenu
	}
	return application.KeyOther
}

// PointerOf translates an InkView touch phase.
func PointerOf(kind inkview.PointerKind) widget.PointerType {
	switch kind {
	case inkview.PointerMove:
		return widget.PointerMove
	case inkview.PointerUp:
		return widget.PointerUp
	}
	return widget.PointerDown
}

// Screen is the application.Screen of the reader's panel. It remembers whether
// the app asked to close (as opposed to the system closing it with Home).
type Screen struct{ closed bool }

// Show implements application.Screen.
func (*Screen) Show(img *image.Gray) {
	b := img.Bounds()
	inkview.Blit(img.Pix, b.Dx(), b.Dy(), img.Stride)
}

// ShowRect implements application.Screen.
func (*Screen) ShowRect(img *image.Gray, r image.Rectangle) {
	inkview.BlitRect(img.Pix, img.Stride, r.Min.X, r.Min.Y, r.Dx(), r.Dy())
}

// RequestRepaint implements application.Screen.
func (*Screen) RequestRepaint() { inkview.Repaint() }

// Close implements application.Screen.
func (s *Screen) Close() {
	s.closed = true
	inkview.Exit()
}

// ClosedByApp reports whether the app itself asked to close.
func (s *Screen) ClosedByApp() bool { return s.closed }

// Keyboard is the system keyboard of the reader (a widget.Keyboard).
type Keyboard struct{}

// Request implements widget.Keyboard.
func (Keyboard) Request(r widget.KeyboardRequest) {
	flags := inkview.KeyboardNormal
	if r.Kind == widget.KeyboardNumeric {
		flags = inkview.KeyboardNumeric
	}
	inkview.OpenKeyboard(r.Title, r.Initial, r.MaxLen, flags, r.Done)
}

// Adapter is the inkview.App that drives a Runtime.
type Adapter struct{ rt *application.Runtime }

// NewAdapter connects InkView's events to rt.
func NewAdapter(rt *application.Runtime) *Adapter { return &Adapter{rt: rt} }

// Init turns the system panel off so the canvas is the whole screen (with the
// panel on, InkView offsets the canvas and the drawing is not centered), then
// builds the app for the real screen size.
func (a *Adapter) Init() {
	inkview.Logf("info before: %s", inkview.Info())
	inkview.DisablePanel()
	inkview.Logf("info after:  %s", inkview.Info())
	w, h := inkview.ScreenSize()
	a.rt.Start(w, h, Keyboard{})
}

// Draw implements inkview.App.
func (a *Adapter) Draw() { a.rt.Draw() }

// Key implements inkview.App.
func (a *Adapter) Key(code int) bool { return a.rt.Key(KeyOf(code)) }

// Pointer implements inkview.App.
func (a *Adapter) Pointer(kind inkview.PointerKind, x, y int) bool {
	return a.rt.Pointer(PointerOf(kind), x, y)
}

// Run starts the InkView event loop and blocks until the app exits.
func Run(rt *application.Runtime) error { return inkview.Run(NewAdapter(rt)) }

// DesktopRegistrar is the domain.DesktopRegistrar of the reader's home screen.
type DesktopRegistrar struct{ Root string }

// Register implements domain.DesktopRegistrar: icons plus the view.json entry.
func (d DesktopRegistrar) Register(r domain.Registration) (bool, error) {
	res, err := desktop.Register(desktop.Config{
		Root: d.Root, Exe: r.Exe, Title: r.Title, Group: r.Group, Icon: r.Icon,
	})
	return res.Changed, err
}

// IconPath is where Register writes the normal icon of app id under root.
func IconPath(root, id string) string {
	return filepath.Join(root, "applications", "icons", id+".bmp")
}

// Logf writes to the on-device debug log (see inkview.DebugLog).
func Logf(format string, args ...interface{}) { inkview.Logf(format, args...) }

// SetLogFile chooses the debug log file.
func SetLogFile(path string) { inkview.DebugLog = path }
