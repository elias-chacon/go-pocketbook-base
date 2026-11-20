package application

import (
	"sync"
	"time"

	"go-pocketbook-base/internal/platform/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

// keyUpdateTitle is the title of every dialog of the update.
const keyUpdateTitle = "update.title"

// How the runtime waits for the background check: it polls from the event loop (a goroutine
// never touches a widget), which costs nothing while no change is made.
const (
	pollEvery = 2 * time.Second
	maxPolls  = 90 // three minutes: longer than any request is allowed to take
	// busyRetry is how long the offer waits when the person has a dialog open.
	busyRetry   = 5 * time.Second
	maxBusyWait = 60
)

// WithUpdates makes the runtime look for a newer build of the app in the background as soon
// as it starts, and offer it in a dialog (see Updater). It is part of the blueprint of every
// app and of the Launcher: platform.Run turns it on for all of them.
func (r *Runtime) WithUpdates(u Updater) *Runtime {
	r.updates = &u
	return r
}

// checkResult is the answer of the background check, handed to the event loop.
type checkResult struct {
	mu     sync.Mutex
	done   bool
	update domain.Update
	found  bool
}

func (c *checkResult) set(u domain.Update, found bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.update, c.found, c.done = u, found, true
}

func (c *checkResult) get() (domain.Update, bool, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.update, c.found, c.done
}

// watchUpdates starts the check off the event loop and the polling on it.
func (r *Runtime) watchUpdates() {
	if r.updates == nil {
		return
	}
	res := &checkResult{}
	u := *r.updates
	go func() { res.set(u.Check()) }()
	r.pollUpdate(res, 0)
}

func (r *Runtime) pollUpdate(res *checkResult, n int) {
	up, found, done := res.get()
	switch {
	case !done && n < maxPolls:
		r.screen.After(pollEvery, func() { r.pollUpdate(res, n+1) })
	case done && found:
		r.offerUpdate(up, 0)
	}
}

// offerUpdate asks the person whether to update now. A dialog already open is never covered:
// the offer waits for it to be closed.
func (r *Runtime) offerUpdate(up domain.Update, waited int) {
	if r.doc.Modal() != nil {
		if waited < maxBusyWait {
			r.screen.After(busyRetry, func() { r.offerUpdate(up, waited+1) })
		}
		return
	}
	tr := translator(r.env.Lang)
	m := widget.NewModal(tr.T(keyUpdateTitle), widget.NewLabel(tr.T("update.body", up.Version, r.env.Version)))
	m.SetButtons(widget.ModalOKCancel)
	m.OKText, m.CancelText = tr.T("update.now"), tr.T("update.later")
	m.OnCancel = func(*widget.Event) { r.updates.Postpone(up) }
	// The dialog closes after OnOK, so the work starts on the next turn of the event loop.
	m.OnOK = func(*widget.Event) { r.screen.After(0, func() { r.installUpdate(up) }) }
	m.Open(r.doc)
}

// installUpdate shows that it is working, then downloads, replaces and restarts. The download
// blocks the event loop on purpose: the dialog says so, and nothing else may change the app
// while its file is replaced.
func (r *Runtime) installUpdate(up domain.Update) {
	tr := translator(r.env.Lang)
	busy := widget.NewModal(tr.T(keyUpdateTitle), widget.NewLabel(tr.T("update.working", up.Version)))
	busy.SetButtons(widget.ModalNoButtons)
	busy.Open(r.doc)
	r.flush()
	err := r.updates.Apply(up)
	busy.Close()
	if err == nil {
		return // restarting: not reached on the reader
	}
	failed := widget.NewModal(tr.T(keyUpdateTitle), widget.NewLabel(tr.T("update.failed", err.Error())))
	failed.Open(r.doc)
}
