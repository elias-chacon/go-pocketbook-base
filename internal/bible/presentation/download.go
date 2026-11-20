package presentation

import (
	"time"

	"go-pocketbook-base/internal/bible/domain"
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/widget"
)

// job is a long task done in small steps between two touches: the screen stays usable, and a strip
// above the footer shows what is going on, with a bar and a button to cancel.
type job struct {
	// text is what the strip says at a percentage.
	text func(percent int) string
	// step does a piece of the work. It reports the percentage reached and whether the task ended.
	step func() (percent int, finished bool, err error)
	// done runs when the task ended or failed (not when it was cancelled); it may start another.
	done func(err error)

	percent int
}

// jobDelay is the pause between two steps: the reader answers the touches and paints the bar in it.
const jobDelay = 40 * time.Millisecond

// jobStripHeight is what the strip takes from the page while a task runs.
const jobStripHeight = 150.0

// buildStrip makes the strip of the running task, hidden until there is one.
func (s *Screen) buildStrip() widget.Widget {
	s.stripLabel = widget.NewLabel("")
	s.stripLabel.ID = "job-label"
	s.stripLabel.Style.Bold = true
	s.stripBar = widget.NewProgress(0)
	s.stripBar.ID = "job-bar"
	s.stripBar.Style.Width = widget.Pct(100)
	s.stripBar.Style.MinHeight = 40
	cancel := widget.NewButton(s.t("bible.download.cancel"))
	cancel.ID = "job-cancel"
	cancel.OnClick = func(*widget.Event) { s.cancelJob() }
	top := widget.NewRow(16, s.stripLabel, cancel)
	top.Style.AlignItems = widget.AlignCenter
	s.stripLabel.Style.FlexGrow = 1
	s.strip = widget.NewColumn(8, top, s.stripBar)
	s.strip.ID = "job"
	s.strip.Style.Padding = geom.XY(28, 12)
	s.strip.Style.Background = widget.Paper
	s.showJob()
	return s.strip
}

// showJob puts the running task (or none) on the strip.
func (s *Screen) showJob() {
	if s.strip == nil {
		return
	}
	s.strip.Hidden = s.job == nil
	if s.job != nil {
		s.stripLabel.Text = s.job.text(s.job.percent)
		s.stripBar.Value = float64(s.job.percent)
		s.stripBar.Text = s.percentText(float64(s.job.percent))
	}
	s.Doc.Invalidate()
}

// Busy reports whether a task (a download) is running.
func (s *Screen) Busy() bool { return s.job != nil }

// startJob runs a task in small steps scheduled on the event loop (all at once when the platform
// gives no timer, as in the tests).
func (s *Screen) startJob(j *job) {
	s.job = j
	s.showJob()
	if s.cfg.After == nil {
		for s.job == j {
			s.stepJob(j)
		}
		return
	}
	s.cfg.After(jobDelay, func() { s.stepJob(j) })
}

// stepJob does one step and schedules the next, or ends the task.
func (s *Screen) stepJob(j *job) {
	if s.job != j { // cancelled
		return
	}
	pct, finished, err := j.step()
	if err != nil || finished {
		s.job = nil
		s.showJob()
		j.done(err)
		return
	}
	j.percent = pct
	s.showJob()
	if s.cfg.After != nil {
		s.cfg.After(jobDelay, func() { s.stepJob(j) })
	}
}

// cancelJob stops the running task; a download goes on from where it stopped when started again.
func (s *Screen) cancelJob() {
	if s.job == nil {
		return
	}
	s.job = nil
	s.showJob()
	s.say(s.t("bible.download.cancelled"))
}

// refreshPage builds the page being shown again, wherever the person is.
func (s *Screen) refreshPage() {
	s.tabs.Show(s.tabs.Current())
	s.Doc.Invalidate()
}

// askDownload asks before downloading a version (it takes a few megabytes and the network).
func (s *Screen) askDownload(v domain.Version) {
	m := widget.NewModal(s.t("bible.download.title"), widget.NewLabel(s.t("bible.download.ask", v.Label())))
	m.ID = "modal-download"
	m.OKText, m.CancelText = s.t("bible.download"), s.t("bible.settings.cancel")
	m.SetButtons(widget.ModalOKCancel)
	m.OnOK = func(*widget.Event) { s.startDownload(v) }
	m.OnCancel = func(*widget.Event) { s.rebuildBible() } // the picker goes back to the version in use
	m.Open(s.Doc)
}

// startDownload downloads a version in the background of the screen: the bar on the strip moves
// and the person can go on reading or browsing. When it ends the version is put in use, unless the
// person is in the middle of a chapter.
func (s *Screen) startDownload(v domain.Version) {
	if s.Busy() {
		s.say(s.t("bible.download.busy"))
		s.rebuildBible()
		return
	}
	dl, err := s.cfg.Bible.StartDownload(v.Slug)
	if err != nil {
		s.fail(err)
		s.rebuildBible()
		return
	}
	s.startJob(&job{
		text: func(p int) string { return s.t("bible.download.progress", v.Label(), p) },
		step: func() (int, bool, error) {
			if err := dl.Step(); err != nil {
				return 0, false, err
			}
			done, total := dl.Progress()
			pct := 0
			if total > 0 {
				pct = done * 100 / total
			}
			return pct, dl.Finished(), nil
		},
		done: func(err error) { s.downloaded(v, err) },
	})
}

// downloaded is what happens when a download ends.
func (s *Screen) downloaded(v domain.Version, err error) {
	if err != nil {
		s.fail(err)
		s.refreshPage()
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
	if s.tabs.Current() == 1 && s.view == viewReader {
		return // the chapter being read stays; the version can be picked from the list
	}
	s.adoptVersion(v)
	s.refreshPage()
}

// RefreshVersions fetches the list of versions from the catalog.
func (s *Screen) RefreshVersions() { s.refreshThen(nil) }

// refreshThen fetches the list of versions as a task and then runs next (when it worked).
func (s *Screen) refreshThen(next func()) {
	if s.Busy() {
		s.say(s.t("bible.download.busy"))
		return
	}
	s.startJob(&job{
		text: func(int) string { return s.t("bible.download.first") },
		step: func() (int, bool, error) { return 100, true, s.cfg.Bible.Refresh() },
		done: func(err error) {
			if err != nil {
				s.say(s.t("bible.refresh.failed", err.Error()))
				return
			}
			if err := s.reload(); err != nil {
				s.fail(err)
				return
			}
			s.refreshPage()
			if next != nil {
				next()
			}
		},
	})
}

// getFirstVersion is the one tap of the first run: fetch the list if needed, then download the
// version of the person's language and start reading it.
func (s *Screen) getFirstVersion() {
	if len(s.versions) == 0 {
		s.refreshThen(func() {
			if len(s.versions) > 0 {
				s.getFirstVersion()
			}
		})
		return
	}
	v, ok := s.defaultVersion()
	if !ok {
		return
	}
	if v.Downloaded {
		s.adoptVersion(v)
		s.Show(0)
		return
	}
	s.startDownload(v)
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
