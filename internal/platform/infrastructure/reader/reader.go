// Package reader adapts the platform to the PocketBook e-reader: it implements
// the Screen port and the system keyboard with InkView, translates InkView's
// events into the runtime's, and registers apps on the home screen. It is the
// only part of the platform that knows the device exists.
package reader

import (
	"image"
	"net/http"
	"path/filepath"
	"time"

	"go-pocketbook-base/internal/device/desktop"
	"go-pocketbook-base/internal/device/diskspace"
	"go-pocketbook-base/internal/device/inkview"
	"go-pocketbook-base/internal/device/network"
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
type Screen struct {
	closed bool
	mode   inkview.UpdateMode // how the repaints of the platform refresh the panel
	// BeforeClose, when set, runs as the app asks to close and before InkView is
	// told to exit (InkView ends the process itself, so nothing can run after
	// the event loop). It may replace the process, as returning to the Launcher does.
	BeforeClose func()
}

// Show implements application.Screen.
func (*Screen) Show(img *image.Gray) {
	b := img.Bounds()
	inkview.Blit(img.Pix, b.Dx(), b.Dy(), img.Stride)
}

// ShowRect implements application.Screen.
func (s *Screen) ShowRect(img *image.Gray, r image.Rectangle) {
	if s.mode == inkview.UpdatePartial {
		inkview.BlitRect(img.Pix, img.Stride, r.Min.X, r.Min.Y, r.Dx(), r.Dy())
		return
	}
	inkview.BlitGrayMode(img.Pix, img.Stride, r.Min.X, r.Min.Y, r.Dx(), r.Dy(), s.mode)
}

// RequestRepaint implements application.Screen.
func (*Screen) RequestRepaint() { inkview.Repaint() }

// After implements application.Screen with an InkView timer.
func (*Screen) After(d time.Duration, f func()) { inkview.After(d, f) }

// Close implements application.Screen.
func (s *Screen) Close() {
	s.closed = true
	if s.BeforeClose != nil {
		s.BeforeClose()
	}
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

// Unregister implements domain.DesktopRemover: the view.json entry and the icons.
func (d DesktopRegistrar) Unregister(appID string) (bool, error) {
	return desktop.Unregister(d.Root, appID)
}

// RemoveOrphans takes out of the home screen the entries of apps whose program is gone.
func (d DesktopRegistrar) RemoveOrphans(report func(id string)) ([]string, error) {
	return desktop.RemoveOrphans(d.Root, report)
}

// IconPath is where Register writes the normal icon of app id under root.
func IconPath(root, id string) string {
	return filepath.Join(root, "applications", "icons", id+".bmp")
}

// HTTPClient is the network client of the reader (see device/network).
func HTTPClient(timeout time.Duration) *http.Client { return network.New(timeout) }

// DeviceModel is the model name the firmware reports ("" off the reader).
func DeviceModel() string { return inkview.DeviceModel() }

// DiskSpace is the total and the free bytes of the volume that holds path (ok false when unknown).
func DiskSpace(path string) (total, free uint64, ok bool) {
	s, ok := diskspace.Of(path)
	return s.Total, s.Free, ok
}

// Logf writes a trace line to the app's log, only in verbose mode (see SetVerbose).
func Logf(format string, args ...interface{}) { inkview.Logf(format, args...) }

// SetLogFile chooses the debug log file.
func SetLogFile(path string) { inkview.DebugLog = path }

// Errorf writes what went wrong to the app's log, always.
func Errorf(format string, args ...interface{}) { inkview.Errorf(format, args...) }

// SetVerbose turns the trace of events and steps on or off.
func SetVerbose(on bool) { inkview.Verbose = on }

// ResetLog empties a log that has grown too big.
func ResetLog() { inkview.ResetLog() }

// Info implements application.Panel: what the reader says about its panel.
func (*Screen) Info() application.PanelInfo {
	return application.PanelInfo{
		Model: inkview.DeviceModel(), Depth: inkview.HardwareDepth(), Waveform: inkview.Waveform(), Canvas: inkview.Info(),
	}
}

// DrawGray implements application.Panel.
func (*Screen) DrawGray(img *image.Gray, r image.Rectangle, m application.UpdateMode) {
	inkview.BlitGrayMode(img.Pix, img.Stride, r.Min.X, r.Min.Y, r.Dx(), r.Dy(), inkview.UpdateMode(m))
}

// DrawColor implements application.Panel.
func (*Screen) DrawColor(img *image.RGBA, r image.Rectangle, m application.UpdateMode) {
	inkview.BlitColorMode(img.Pix, img.Stride, r.Min.X, r.Min.Y, r.Dx(), r.Dy(), inkview.UpdateMode(m))
}

// SetMode implements application.Panel.
func (s *Screen) SetMode(m application.UpdateMode) { s.mode = inkview.UpdateMode(m) }

// Wait implements application.Panel.
func (*Screen) Wait() { inkview.WaitUpdate() }

// Settle implements application.Panel.
func (*Screen) Settle() {
	inkview.ExitA2()
	inkview.WaitUpdate()
}
