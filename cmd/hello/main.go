// Command hello is the PocketBook showcase application: it runs the widget
// demo on the device and registers its own launcher icon on first run.
package main

import (
	"image"
	"log"
	"os"

	"go-pocketbook-base/internal/demo"
	"go-pocketbook-base/internal/inkview"
	"go-pocketbook-base/internal/install"
	"go-pocketbook-base/internal/ui"
	"go-pocketbook-base/internal/version"
	"go-pocketbook-base/internal/widget"
)

const (
	storageRoot = "/mnt/ext1"
	appTitle    = "Hello Go"
	appGroup    = "@General" // desktop group where the icon is listed
)

// app connects InkView events to the widget document.
type app struct {
	notice   string
	demo     *demo.App
	pending  bool // a repaint is already queued
	partials int  // partial panel updates since the last full one
}

// fullRefreshEvery forces a full (flashing) panel refresh after this many
// partial ones, to clear the ghosting that fast e-ink updates leave behind.
const fullRefreshEvery = 10

// Init turns the system panel off so the canvas is the whole screen (with the
// panel on, InkView offsets the canvas and the drawing is not centered), then
// builds the UI for the real screen size.
func (a *app) Init() {
	inkview.Logf("info before: %s", inkview.Info())
	inkview.DisablePanel()
	inkview.Logf("info after:  %s", inkview.Info())

	w, h := inkview.ScreenSize()
	a.demo = demo.New(w, h, demo.Config{
		Version:  version.Value,
		Keyboard: deviceKeyboard{},
		Exit:     inkview.Exit,
		Notice:   a.notice,
	})
	a.demo.Doc.OnInvalidate = a.requestRepaint
}

// requestRepaint queues at most one repaint, however many changes happen.
func (a *app) requestRepaint() {
	if !a.pending {
		a.pending = true
		inkview.Repaint()
	}
}

// Draw repaints only what changed (Document.Update) and sends just those
// areas to the panel. A draw the system asked for (not ours) may mean something
// covered the screen, so it repaints everything.
func (a *app) Draw() {
	if !a.pending {
		a.demo.Doc.ResetFrame()
	}
	a.pending = false

	f := a.demo.Doc.Update()
	img := f.Image
	if f.Full || a.partials >= fullRefreshEvery {
		b := img.Bounds()
		inkview.Blit(img.Pix, b.Dx(), b.Dy(), img.Stride)
		a.partials = 0
	} else if len(f.Rects) > 0 {
		for _, r := range f.Rects {
			inkview.BlitRect(img.Pix, img.Stride, r.Min.X, r.Min.Y, r.Dx(), r.Dy())
		}
		a.partials++
	}
}

// Key: Back closes an open popup, otherwise leaves the app.
func (a *app) Key(code int) bool {
	if code != inkview.KeyBack {
		return false
	}
	if a.demo.Doc.HasPopup() {
		a.demo.Doc.ClosePopups()
	} else {
		inkview.Exit()
	}
	return true
}

func (a *app) Pointer(kind inkview.PointerKind, x, y int) bool {
	if kind == inkview.PointerMove && a.pending {
		return true // the screen is still catching up: skip intermediate positions
	}
	t := map[inkview.PointerKind]widget.PointerType{
		inkview.PointerDown: widget.PointerDown,
		inkview.PointerMove: widget.PointerMove,
		inkview.PointerUp:   widget.PointerUp,
	}[kind]
	a.demo.Doc.HandlePointer(t, float64(x), float64(y))
	return true
}

// deviceKeyboard is the system keyboard of the reader.
type deviceKeyboard struct{}

func (deviceKeyboard) Request(r widget.KeyboardRequest) {
	flags := inkview.KeyboardNormal
	if r.Kind == widget.KeyboardNumeric {
		flags = inkview.KeyboardNumeric
	}
	inkview.OpenKeyboard(r.Title, r.Initial, r.MaxLen, flags, r.Done)
}

// firstRun registers the launcher icon on the desktop. PocketBook has no
// installer, so the app installs itself the first time it runs.
func firstRun() string {
	exe, err := os.Executable()
	if err != nil {
		exe = storageRoot + "/applications/hello.app"
	}
	res, err := install.Register(install.Config{
		Root:  storageRoot,
		Exe:   exe,
		Title: appTitle,
		Group: appGroup,
		Icon:  func(size int, focused bool) image.Image { return ui.RenderIcon(size, focused) },
	})
	inkview.Logf("install: %+v err=%v", res, err)
	switch {
	case err != nil:
		return "Falha ao registrar o ícone: " + err.Error()
	case res.Changed:
		return "Ícone registrado: aparece na tela inicial depois que o aparelho recarregar (ex.: ao tirar o cabo USB)."
	}
	return ""
}

func main() {
	// Event log for on-device debugging; remove or comment out when done.
	inkview.DebugLog = storageRoot + "/applications/hello.log"
	wd, _ := os.Getwd()
	inkview.Logf("main start version=%s args=%q cwd=%q", version.Value, os.Args, wd)

	err := inkview.Run(&app{notice: firstRun()})
	inkview.Logf("Run returned err=%v", err)
	if err != nil {
		log.Fatal(err)
	}
}
