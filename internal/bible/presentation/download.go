package presentation

import (
	"go-pocketbook-base/internal/bible/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

// askDownload asks before downloading a version (it takes a few megabytes and the network).
func (s *Screen) askDownload(v domain.Version) {
	m := widget.NewModal(s.t("bible.download.title"), widget.NewLabel(s.t("bible.download.ask", v.Label())))
	m.ID = "modal-download"
	m.OKText, m.CancelText = s.t("bible.download"), s.t("bible.settings.cancel")
	m.SetButtons(widget.ModalOKCancel)
	m.OnOK = func(*widget.Event) { s.runDownload(v) }
	m.OnCancel = func(*widget.Event) { s.rebuildBible() } // the picker goes back to the version in use
	m.Open(s.Doc)
}

// working opens a modal with a line of text and no buttons, paints it, and returns the function that
// changes the line and the one that closes it. The calls that follow block the screen.
func (s *Screen) working(title, text string) (update func(string), done func()) {
	label := widget.NewLabel(text)
	label.ID = "working"
	m := widget.NewModal(title, label)
	m.ID = "modal-working"
	m.SetButtons(widget.ModalNoButtons)
	m.Open(s.Doc)
	s.flush()
	return func(t string) { label.Text = t; s.Doc.Invalidate(); s.flush() }, m.Close
}

// runDownload downloads a version with its progress on the screen, and starts using it.
func (s *Screen) runDownload(v domain.Version) {
	update, done := s.working(s.t("bible.download.title"), s.t("bible.download.progress", v.Label(), 0))
	err := s.cfg.Bible.Download(v.Slug, func(d, total int) {
		pct := 0
		if total > 0 {
			pct = d * 100 / total
		}
		update(s.t("bible.download.progress", v.Label(), pct))
	})
	done()
	if err != nil {
		s.fail(err)
		s.rebuildBible()
		return
	}
	if err := s.reload(); err != nil {
		s.fail(err)
		return
	}
	for _, nv := range s.versions {
		if nv.Slug == v.Slug {
			v = nv
		}
	}
	s.say(s.t("bible.download.done", v.Label()))
	s.useVersion(v)
}

// RefreshVersions fetches the list of versions from the catalog.
func (s *Screen) RefreshVersions() {
	update, done := s.working(s.t("bible.download.title"), s.t("bible.download.first"))
	_ = update
	err := s.cfg.Bible.Refresh()
	done()
	if err != nil {
		s.say(s.t("bible.refresh.failed", err.Error()))
		return
	}
	if err := s.reload(); err != nil {
		s.fail(err)
		return
	}
	s.rebuildBible()
}

// getFirstVersion is the one tap of the first run: fetch the list if needed, then download the
// version of the person's language and start reading it.
func (s *Screen) getFirstVersion() {
	if len(s.versions) == 0 {
		s.RefreshVersions()
		if len(s.versions) == 0 {
			return
		}
	}
	v, ok := s.defaultVersion()
	if !ok {
		return
	}
	if v.Downloaded {
		s.useVersion(v)
		s.Show(0)
		return
	}
	s.runDownload(v)
	s.Show(0)
}

// defaultVersion is the version the preferences name, or the first one of the language.
func (s *Screen) defaultVersion() (domain.Version, bool) {
	if v, ok := s.currentVersion(); ok {
		return v, true
	}
	if in := domain.VersionsIn(s.versions, s.activeLanguage()); len(in) > 0 {
		return in[0], true
	}
	if len(s.versions) > 0 {
		return s.versions[0], true
	}
	return domain.Version{}, false
}
