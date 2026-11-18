package application

import (
	"go-pocketbook-base/internal/toolkit/widget"
)

// fullRefreshEvery forces a full (flashing) panel refresh after this many
// partial ones, to clear the ghosting that fast e-ink updates leave behind.
const fullRefreshEvery = 10

// Runtime runs one app: it connects the reader's events (draw, key, touch) to
// the app's document.
type Runtime struct {
	env     Env
	factory Factory
	screen  Screen

	app      App
	doc      *widget.Document
	pending  bool // a repaint is already queued
	partials int  // partial panel updates since the last full one
}

// NewRuntime prepares the runtime; Start builds the app.
func NewRuntime(env Env, factory Factory, screen Screen) *Runtime {
	return &Runtime{env: env, factory: factory, screen: screen}
}

// Start builds the app for the real screen size (known only once the reader is
// running) and wires its repaints.
func (r *Runtime) Start(width, height int, keyboard widget.Keyboard) {
	r.env.Width, r.env.Height, r.env.Keyboard = width, height, keyboard
	r.env.Exit = r.screen.Close
	r.env.Flush = r.flush
	r.app = r.factory(r.env)
	r.doc = r.app.Document()
	r.doc.OnInvalidate = r.requestRepaint
}

// requestRepaint queues at most one repaint, however many changes happen.
func (r *Runtime) requestRepaint() {
	if !r.pending {
		r.pending = true
		r.screen.RequestRepaint()
	}
}

// flush paints now what has changed (see Env.Flush).
func (r *Runtime) flush() {
	r.pending = true
	r.Draw()
}

// Draw repaints only what changed (Document.Update) and sends just those areas
// to the panel. A draw the system asked for (not ours) may mean something
// covered the screen, so it repaints everything.
func (r *Runtime) Draw() {
	if !r.pending {
		r.doc.ResetFrame()
	}
	r.pending = false

	f := r.doc.Update()
	switch {
	case f.Full || r.partials >= fullRefreshEvery:
		r.screen.Show(f.Image)
		r.partials = 0
	case len(f.Rects) > 0:
		for _, rect := range f.Rects {
			r.screen.ShowRect(f.Image, rect)
		}
		r.partials++
	}
}

// Key handles a hardware key. Back keys close an open popup, then an open modal.
// What is left goes to the app (KeyHandler); a real Back key then leaves.
func (r *Runtime) Key(k Key) bool {
	if k.IsBack() && r.doc.HandleBack() {
		return true
	}
	if h, ok := r.app.(KeyHandler); ok && h.Key(k) {
		return true
	}
	if k == KeyBack {
		r.screen.Close() // devices that do have a Back key
		return true
	}
	return false
}

// Pointer routes a touch (device pixels) to the document.
func (r *Runtime) Pointer(t widget.PointerType, x, y int) bool {
	if t == widget.PointerMove && r.pending {
		return true // the screen is still catching up: skip intermediate positions
	}
	r.doc.HandlePointer(t, float64(x), float64(y))
	return true
}
