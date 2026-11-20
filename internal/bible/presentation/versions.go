package presentation

import (
	"go-pocketbook-base/internal/bible/domain"
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/widget"
)

// versionsPageSize is how many versions a page of the table shows.
const versionsPageSize = 7

// stored lists the versions that have text on the reader, whole or half downloaded.
func (s *Screen) stored() []domain.Version {
	var out []domain.Version
	for _, v := range s.versions {
		if v.Downloaded || v.Verses > 0 {
			out = append(out, v)
		}
	}
	return out
}

// OpenVersions opens the manager of the downloaded versions (More > Downloaded versions): a table
// with each version that has text on the reader, and a button that deletes the selected one
// completely. What was read stays counted, because progress is kept by place, not by version.
func (s *Screen) OpenVersions() {
	if err := s.reload(); err != nil {
		s.fail(err)
		return
	}
	stored := s.stored()

	table := widget.NewTable(versionsPageSize,
		widget.TableColumn{Title: s.t("bible.versions.col.version"), Flex: 3, Sortable: true},
		widget.TableColumn{Title: s.t("bible.versions.col.lang"), Flex: 1.4, Sortable: true},
		widget.TableColumn{Title: s.t("bible.versions.col.verses"), Flex: 1.2},
		widget.TableColumn{Title: s.t("bible.versions.col.state"), Flex: 1.4})
	table.ID = "versions-table"
	for _, v := range stored {
		state := s.t("bible.versions.partial")
		if v.Downloaded {
			state = s.t("bible.versions.complete")
		}
		table.Rows = append(table.Rows, []string{v.Label(), languageName(v.Language), s.number(v.Verses), state})
	}

	del := widget.NewButton(s.t("bible.versions.delete"))
	del.ID = "versions-delete"
	del.Disabled = true
	del.Style.Padding = geom.XY(0, 20)
	hint := muted(s.t("bible.versions.hint"))
	hint.ID = "versions-hint"

	body := []widget.Widget{hint, table, del}
	if len(stored) == 0 {
		none := widget.NewLabel(s.t("bible.versions.none"))
		none.ID = "versions-none"
		body = []widget.Widget{none}
	}
	m := widget.NewModal(s.t("bible.versions.title"), body...)
	m.ID = "modal-versions"
	m.OKText = s.t("bible.combo.close")
	m.SetButtons(widget.ModalOK)
	m.SetControls(widget.ModalClose)

	table.OnSelect = func(*widget.Event) {
		del.Disabled = table.Selected < 0
		m.Refresh()
	}
	del.OnClick = func(*widget.Event) {
		if table.Selected >= 0 && table.Selected < len(stored) {
			m.Close()
			s.askRemove(stored[table.Selected])
		}
	}
	m.Open(s.Doc)
}

// askRemove asks before deleting the text of a version, then comes back to the manager.
func (s *Screen) askRemove(v domain.Version) {
	m := widget.NewModal(s.t("bible.versions.confirm.title"), widget.NewLabel(s.t("bible.versions.confirm", v.Label())))
	m.ID = "modal-remove"
	m.OKText, m.CancelText = s.t("bible.versions.delete.ok"), s.t("bible.settings.cancel")
	m.SetButtons(widget.ModalOKCancel)
	m.OnOK = func(*widget.Event) { s.remove(v) }
	m.OnCancel = func(*widget.Event) { s.OpenVersions() }
	m.Open(s.Doc)
}

// remove deletes the text of a version and shows the manager again. A version being downloaded
// is not touched. When it was the one in use, the app goes back to having none (the Home offers
// the download again).
func (s *Screen) remove(v domain.Version) {
	if s.Busy() {
		s.say(s.t("bible.versions.busy"))
		s.OpenVersions()
		return
	}
	if err := s.cfg.Bible.Remove(v.Slug); err != nil {
		s.fail(err)
		s.OpenVersions()
		return
	}
	if v.Slug == s.prefs.Version {
		s.prefs.Version = ""
		s.view = viewBooks
		if err := s.cfg.Bible.SavePreferences(s.prefs); err != nil {
			s.fail(err)
		}
	}
	s.say(s.t("bible.versions.removed", v.Label()))
	s.OpenVersions() // reloads the versions, so the list is the new truth
	s.tabs.Show(s.tabs.Current())
}
