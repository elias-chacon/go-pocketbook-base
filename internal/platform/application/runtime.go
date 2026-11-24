package application

import (
	"time"

	"go-pocketbook-base/internal/toolkit/widget"
)

// fullRefreshEvery forces a full (flashing) panel refresh after this many
// partial ones, to clear the ghosting that fast e-ink updates leave behind.
const fullRefreshEvery = 10

// staleRequests is how long a repaint we asked for may stay unanswered before we stop counting
// it: the reader may merge two requests into one event, and a count that never comes down would
// hide a draw the system really asked for.
const staleRequests = 10 * time.Second

// Runtime runs one app: it connects the reader's events (draw, key, touch) to
// the app's document.
type Runtime struct {
	env     Env
	factory Factory
	screen  Screen

	updates *Updater // looks for a newer build in the background (nil: no check)

	app      App
	doc      *widget.Document
	pending  bool // a change is waiting to be painted
	watcher  func(PaintStats)
	partials int // partial panel updates since the last full one

	// queued is how many repaint events we asked the reader for that have not arrived yet. When
	// one arrives it is ours (a cheap partial update, often nothing at all); only a draw that
	// nobody here asked for means the system covered the screen. Counting them matters: the
	// app may paint by itself (Env.Flush) while its earlier requests are still on the way, and
	// each of those leftovers taken for a system draw would flash the whole panel.
	queued  int
	lastReq time.Time
	clock   func() time.Time // time.Now, a test replaces it
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
	r.env.After = r.screen.After
	r.env.Watch = func(f func(PaintStats)) { r.watcher = f }
	if p, ok := r.screen.(Panel); ok {
		r.env.Panel = p
	}
	r.app = r.factory(r.env)
	r.doc = r.app.Document()
	r.doc.OnInvalidate = r.requestRepaint
	r.watchUpdates()
}

// requestRepaint queues at most one repaint, however many changes happen.
func (r *Runtime) requestRepaint() {
	if !r.pending {
		r.pending = true
		r.queued++
		r.lastReq = r.now()
		r.screen.RequestRepaint()
	}
}

func (r *Runtime) now() time.Time {
	if r.clock != nil {
		return r.clock()
	}
	return time.Now()
}

// flush paints now what has changed (see Env.Flush). The repaint events already requested stay
// counted: they will arrive and find nothing left to do.
func (r *Runtime) flush() { r.paint() }

// Draw repaints only what changed (Document.Update) and sends just those areas
// to the panel. A draw the system asked for (not ours) may mean something
// covered the screen, so it repaints everything.
func (r *Runtime) Draw() {
	if r.queued > 0 && r.now().Sub(r.lastReq) > staleRequests {
		r.queued = 0
	}
	if r.queued > 0 {
		r.queued-- // an event we asked for
	} else {
		r.doc.ResetFrame()
	}
	r.paint()
}

// paint sends what changed to the panel.
func (r *Runtime) paint() {
	r.pending = false
	if r.painted() {
		return
	}

	f := r.doc.Update()
	shown := time.Now()
	switch {
	case f.Full || r.partials >= fullRefreshEvery:
		r.screen.Show(f.Image)
		r.partials = 0
	case len(f.Rects) > 0:
		for _, rect := range f.Rects {
			r.screen.ShowRect(f.Image, rect)
		}
		r.partials++
	default:
		return
	}
	if r.watcher != nil {
		r.watcher(PaintStats{UpdateStats: r.doc.LastUpdate(), Show: time.Since(shown), Areas: len(f.Rects)})
	}
}

// painted lets an app that draws the panel itself (Raw) do it instead of the platform.
func (r *Runtime) painted() bool {
	raw, ok := r.app.(Raw)
	if !ok || r.env.Panel == nil {
		return false
	}
	return raw.Raw(r.env.Panel)
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
