package presentation

import (
	"go-pocketbook-base/internal/tasks/application"
	"go-pocketbook-base/internal/toolkit/widget"
)

// Syncer is what the screen needs to keep the tasks in step with Microsoft To Do or
// Google Tasks. The app implements it over the account it is connected to.
type Syncer interface {
	// Account is the label of the connected account ("" when there is none).
	Account() string
	// Run synchronizes once.
	Run() (application.SyncResult, error)
	// Connect opens the screen that signs in to a service.
	Connect()
	// Disconnect forgets the connected account (the tasks stay on the reader).
	Disconnect() error
}

// syncMenu is the Sync menu of the title row: sync now, connect or switch the account,
// disconnect.
func (s *Screen) syncMenu() *widget.Dropdown {
	items := []widget.MenuItem{{Label: s.t("tasks.sync.now"), OnSelect: s.SyncNow}}
	label := s.t("tasks.sync.connect")
	if s.cfg.Syncer.Account() != "" {
		label = s.t("tasks.sync.switch")
	}
	items = append(items, widget.MenuItem{Label: label, OnSelect: s.cfg.Syncer.Connect})
	if s.cfg.Syncer.Account() != "" {
		items = append(items, widget.MenuItem{Label: s.t("tasks.sync.disconnect"), OnSelect: s.Disconnect})
	}
	d := widget.NewDropdown(s.t("tasks.sync"), items...)
	d.ID = "tasks-sync"
	return d
}

// say shows a line in the footer.
func (s *Screen) say(text string) {
	s.notice = text
	if s.note != nil {
		s.note.Text = text
	}
	s.Doc.Invalidate()
}

// SyncNow synchronizes, or opens the sign-in when no account is connected.
func (s *Screen) SyncNow() {
	if s.cfg.Syncer == nil {
		return
	}
	if s.cfg.Syncer.Account() == "" {
		s.cfg.Syncer.Connect()
		return
	}
	s.say(s.t("tasks.syncing"))
	if s.cfg.Flush != nil {
		s.cfg.Flush()
	}
	res, err := s.cfg.Syncer.Run()
	if err != nil {
		s.say(s.t("tasks.sync.failed", err.Error()))
		return
	}
	s.reload()
	if len(res.Errors) > 0 {
		s.say(s.t("tasks.synced.partial", len(res.Errors), res.Errors[0].Error()))
		return
	}
	s.say(s.t("tasks.synced", res.Sent, res.Received, res.Removed))
}

// Disconnect forgets the connected account.
func (s *Screen) Disconnect() {
	if err := s.cfg.Syncer.Disconnect(); err != nil {
		s.say(s.t("tasks.sync.failed", err.Error()))
		return
	}
	s.showList()
}
