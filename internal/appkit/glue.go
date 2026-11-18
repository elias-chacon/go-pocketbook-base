package appkit

import (
	"image"

	"go-pocketbook-base/internal/inkview"
	"go-pocketbook-base/internal/widget"
)

// output is how the glue talks to the reader; tests replace it.
type output struct {
	Blit     func(img *image.Gray)                    // the whole screen
	BlitRect func(img *image.Gray, r image.Rectangle) // one changed area
	Repaint  func()                                   // ask InkView for a Draw
	Exit     func()
}

func deviceOutput() output {
	return output{
		Blit: func(img *image.Gray) {
			b := img.Bounds()
			inkview.Blit(img.Pix, b.Dx(), b.Dy(), img.Stride)
		},
		BlitRect: func(img *image.Gray, r image.Rectangle) {
			inkview.BlitRect(img.Pix, img.Stride, r.Min.X, r.Min.Y, r.Dx(), r.Dy())
		},
		Repaint: inkview.Repaint,
		Exit:    inkview.Exit,
	}
}

// glue connects InkView's events to the app's document. It implements
// inkview.App.
type glue struct {
	env     Env
	factory Factory
	out     output

	app      App
	doc      *widget.Document
	pending  bool // a repaint is already queued
	partials int  // partial panel updates since the last full one
}

func newGlue(env Env, factory Factory, out output) *glue {
	return &glue{env: env, factory: factory, out: out}
}

// Init turns the system panel off so the canvas is the whole screen (with the
// panel on, InkView offsets the canvas and the drawing is not centered), then
// builds the app for the real screen size.
func (g *glue) Init() {
	inkview.Logf("info before: %s", inkview.Info())
	inkview.DisablePanel()
	inkview.Logf("info after:  %s", inkview.Info())
	g.env.Width, g.env.Height = inkview.ScreenSize()
	g.env.Keyboard = deviceKeyboard{}
	g.start()
}

// start builds the app (the part of Init that does not need the device).
func (g *glue) start() {
	g.env.Exit = g.out.Exit
	g.app = g.factory(g.env)
	g.doc = g.app.Document()
	g.doc.OnInvalidate = g.requestRepaint
}

// requestRepaint queues at most one repaint, however many changes happen.
func (g *glue) requestRepaint() {
	if !g.pending {
		g.pending = true
		g.out.Repaint()
	}
}

// Draw repaints only what changed (Document.Update) and sends just those areas
// to the panel. A draw the system asked for (not ours) may mean something
// covered the screen, so it repaints everything.
func (g *glue) Draw() {
	if !g.pending {
		g.doc.ResetFrame()
	}
	g.pending = false

	f := g.doc.Update()
	switch {
	case f.Full || g.partials >= fullRefreshEvery:
		g.out.Blit(f.Image)
		g.partials = 0
	case len(f.Rects) > 0:
		for _, r := range f.Rects {
			g.out.BlitRect(f.Image, r)
		}
		g.partials++
	}
}

// Key handles the hardware keys. The PB741 has no Back key (Home, left, right
// and Menu), so left and Menu close an open popup, then an open modal. What is
// left goes to the app (KeyHandler); a real Back key then leaves. Home is
// handled by the system, which closes the app.
func (g *glue) Key(code int) bool {
	if inkview.IsBackKey(code) && g.doc.HandleBack() {
		return true
	}
	if kh, ok := g.app.(KeyHandler); ok && kh.Key(code) {
		return true
	}
	if code == inkview.KeyBack {
		g.out.Exit() // devices that do have a Back key
		return true
	}
	return false
}

// Pointer routes a touch (device pixels) to the document.
func (g *glue) Pointer(kind inkview.PointerKind, x, y int) bool {
	if kind == inkview.PointerMove && g.pending {
		return true // the screen is still catching up: skip intermediate positions
	}
	t := map[inkview.PointerKind]widget.PointerType{
		inkview.PointerDown: widget.PointerDown,
		inkview.PointerMove: widget.PointerMove,
		inkview.PointerUp:   widget.PointerUp,
	}[kind]
	g.doc.HandlePointer(t, float64(x), float64(y))
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
