package presentation

import (
	"errors"
	"sync"
	"time"

	"go-pocketbook-base/internal/launcher/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

// How the screen waits for an install running in the background.
const (
	installPoll  = 300 * time.Millisecond
	installPolls = 2000 // 10 minutes: the HTTP client has its own limits, this only ends a hang
)

// outcome is the end of the install, handed from its goroutine to the event loop.
type outcome struct {
	mu   sync.Mutex
	done bool
	err  error
}

func (o *outcome) set(err error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.done, o.err = true, err
}

func (o *outcome) get() (bool, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.done, o.err
}

// install downloads and installs (or updates) an app of the catalog. A window shows it: the name,
// a bar that follows the download and, at the end, what happened (done, or the error) until the
// person closes it. The download runs in a goroutine and the event loop polls it, so the screen
// stays alive; back (may be nil) runs when the window is closed.
func (h *Home) install(l domain.Listing, update bool, back func()) {
	tr := h.cfg.Tr
	downloading, done := "install.downloading", "install.done"
	if update {
		downloading, done = "update.downloading", "update.done"
	}
	status := widget.NewLabel(tr.T(downloading, l.Title))
	status.ID = "install-status"
	bar := widget.NewProgress(0)
	bar.ID, bar.Style.Width = "install-bar", widget.Pct(100)
	bar.Text = tr.T("install.progress", 0)

	m := widget.NewModal(l.Title, status, bar)
	m.ID = "modal-install"
	m.SetButtons(widget.ModalNoButtons) // nothing to tap while it runs
	m.OKText = tr.T("app.close")
	closed := false // the Back key closed the window before the end
	m.OnCancel = func(*widget.Event) { closed = true }
	m.OnOK = func(*widget.Event) { h.after(back) }
	m.Open(h.Doc)
	h.say(tr.T(downloading, l.Title))
	if h.cfg.Flush != nil {
		h.cfg.Flush()
	}

	finish := func(err error) {
		msg := ""
		if err != nil {
			msg = tr.T("install.failed", l.Title, err.Error())
		} else {
			h.Refresh()
			msg = tr.T(done, l.Title)
		}
		h.say(msg)
		if closed {
			h.after(back)
			return
		}
		status.Text, bar.Hidden = msg, true
		m.OnCancel = m.OnOK
		m.SetControls(widget.ModalClose)
		m.SetButtons(widget.ModalOK)
		h.Doc.Invalidate()
	}

	if h.cfg.After == nil { // no event loop to poll on (tests, a preview): wait for it
		finish(h.cfg.Installer.Install(l))
		return
	}
	res := &outcome{}
	go func() { res.set(h.cfg.Installer.Install(l)) }()
	var poll func(n int)
	poll = func(n int) {
		switch ok, err := res.get(); {
		case ok:
			finish(err)
		case n >= installPolls:
			finish(errors.New("timeout"))
		default:
			h.moveInstallBar(bar)
			h.cfg.After(installPoll, func() { poll(n + 1) })
		}
	}
	poll(0)
}

// moveInstallBar puts the progress of the download on the bar (repainting only when it moved).
func (h *Home) moveInstallBar(bar *widget.Progress) {
	f, _ := h.cfg.InstallProgress.State()
	pct := float64(int(f*100 + 0.5))
	if bar.Value == pct {
		return
	}
	bar.Value, bar.Text = pct, h.cfg.Tr.T("install.progress", int(pct))
	h.Doc.Invalidate()
}
