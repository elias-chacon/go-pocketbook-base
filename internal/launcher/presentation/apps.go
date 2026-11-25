package presentation

import (
	"strings"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/kernel/version"
	"go-pocketbook-base/internal/launcher/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

// appsPageSize is how many apps a page of the management tables shows.
const appsPageSize = 7

// updatable reports whether an installed app is not the build the catalog publishes: the digest of
// its file differs from the AppSHA256 of the release. The version is never what decides, because a
// release may change one app and leave another as it was. Without a digest on either side nothing
// says the app changed, so it is not offered.
func (h *Home) updatable(t domain.Tile) bool {
	if !t.Installed || t.Listing == nil || t.Listing.AppSHA256 == "" || h.cfg.Installer == nil || h.cfg.InstalledSum == nil {
		return false
	}
	sum := h.cfg.InstalledSum(t.ID)
	// A build newer than the published one (a local build) is never taken back to it.
	return sum != "" && !strings.EqualFold(sum, t.Listing.AppSHA256) && !version.Older(t.Listing.Version, t.Version)
}

// Manage opens the window of an app (a long press on its tile, or the settings): update it when
// it is not the published build, install it when it is missing, hide it (installed or not) and
// remove it with all its data. back, when not nil, runs when the window closes, so the settings
// lists come back.
func (h *Home) Manage(t domain.Tile, back func()) {
	tr := h.cfg.Tr
	state := tr.T("app.state.missing")
	if t.Installed {
		state = tr.T("app.state.installed", t.Version)
		if h.updatable(t) {
			state = tr.T("app.state.update", t.Version, t.Listing.Version)
		} else if t.Listing != nil && t.Listing.AppSHA256 != "" && h.cfg.InstalledSum != nil {
			state = tr.T("app.state.current", t.Version)
		}
	}
	desc := widget.NewLabel(t.Description)
	status := widget.NewLabel(state)
	status.ID = "manage-state"
	status.Style.Bold = true
	body := []widget.Widget{desc, status}

	var m *widget.Modal
	add := func(id, key string, action func()) {
		b := widget.NewButton(tr.T(key))
		b.ID = id
		b.Style.Padding = geom.XY(0, 20)
		b.OnClick = func(*widget.Event) { action() }
		body = append(body, b)
	}
	if !t.Installed && t.Listing != nil && h.cfg.Installer != nil {
		add("manage-install", "app.install", func() { m.Close(); h.install(*t.Listing, false, back) })
	}
	if h.updatable(t) {
		add("manage-update", "app.update", func() { m.Close(); h.install(*t.Listing, true, back) })
	}
	add("manage-hide", "app.hide", func() { m.Close(); h.SetHidden(t, true); h.after(back) })
	if t.Installed && h.cfg.Uninstaller != nil {
		add("manage-remove", "app.remove", func() { m.Close(); h.askRemove(t, back) })
	}

	m = widget.NewModal(t.Title, body...)
	m.ID = "modal-app"
	m.OKText = tr.T("app.close")
	m.SetButtons(widget.ModalOK)
	m.SetControls(widget.ModalClose)
	m.OnOK = func(*widget.Event) { h.after(back) }
	m.OnCancel = m.OnOK
	m.Open(h.Doc)
}

// after runs back when there is one.
func (h *Home) after(back func()) {
	if back != nil {
		back()
	}
}

// SetHidden hides an app (or shows it again): it is kept in the preferences, so it survives a
// restart, and it applies to the apps that are not installed too.
func (h *Home) SetHidden(t domain.Tile, hidden bool) {
	prefs := h.cfg.Prefs.WithHidden(t.ID, hidden)
	if h.cfg.SavePrefs != nil {
		if err := h.cfg.SavePrefs(prefs); err != nil {
			h.say(h.cfg.Tr.T("settings.failed", err.Error()))
			return
		}
	}
	h.cfg.Prefs = prefs
	h.Refresh()
	key := "app.hidden"
	if !hidden {
		key = "app.shown"
	}
	h.say(h.cfg.Tr.T(key, t.Title))
}

// askRemove asks before removing an app with everything it keeps on the reader.
func (h *Home) askRemove(t domain.Tile, back func()) {
	tr := h.cfg.Tr
	m := widget.NewModal(tr.T("app.remove.title"), widget.NewLabel(tr.T("app.remove.ask", t.Title)))
	m.ID = "modal-remove"
	m.OKText, m.CancelText = tr.T("app.remove.ok"), tr.T("settings.cancel")
	m.SetButtons(widget.ModalOKCancel)
	m.OnOK = func(*widget.Event) { h.remove(t); h.after(back) }
	m.OnCancel = func(*widget.Event) { h.after(back) }
	m.Open(h.Doc)
}

// remove uninstalls the app and rebuilds the list; the message says what happened.
func (h *Home) remove(t domain.Tile) {
	if err := h.cfg.Uninstaller.Uninstall(t.ID); err != nil {
		h.say(h.cfg.Tr.T("app.remove.failed", t.Title, err.Error()))
		h.Refresh() // some of it may be gone
		return
	}
	h.Refresh()
	h.say(h.cfg.Tr.T("app.remove.done", t.Title))
}

// stateText is the state of a tile in the tables.
func (h *Home) stateText(t domain.Tile) string {
	if t.Installed {
		return h.cfg.Tr.T("app.col.installed")
	}
	return h.cfg.Tr.T("app.col.missing")
}

// OpenManager lists the apps (installed or not, except the hidden) to manage: select one and the
// button opens its window (update, hide, remove).
func (h *Home) OpenManager() {
	tr := h.cfg.Tr
	apps := domain.Visible(h.all, h.cfg.Prefs.HiddenApps)
	h.appsTable(appsList{
		modalID: "modal-manager", tableID: "manager-table", buttonID: "manager-open",
		title: tr.T("settings.apps.manage"), none: tr.T("settings.apps.none"), apps: apps,
		action: tr.T("app.manage"), run: func(t domain.Tile) { h.Manage(t, h.OpenManager) },
	})
}

// OpenHidden lists the hidden apps with a button that shows the selected one again.
func (h *Home) OpenHidden() {
	tr := h.cfg.Tr
	apps := domain.Hidden(h.all, h.cfg.Prefs.HiddenApps)
	h.appsTable(appsList{
		modalID: "modal-hidden", tableID: "hidden-table", buttonID: "hidden-show",
		title: tr.T("settings.apps.hidden"), none: tr.T("settings.hidden.none"), apps: apps,
		action: tr.T("app.show"), run: func(t domain.Tile) { h.SetHidden(t, false); h.OpenHidden() },
	})
}

// appsList describes a window with a list of apps in a table and one action on the selected row.
type appsList struct {
	modalID, tableID, buttonID string
	title, none, action        string
	apps                       []domain.Tile
	run                        func(domain.Tile)
}

// appsTable is the window of a list of apps in a table (app, state, version) with one action on
// the selected row.
func (h *Home) appsTable(l appsList) {
	tr := h.cfg.Tr
	apps, modalID := l.apps, l.modalID
	body := []widget.Widget{}
	if len(apps) == 0 {
		none := widget.NewLabel(l.none)
		none.ID = modalID + "-none"
		body = append(body, none)
	} else {
		table := widget.NewTable(appsPageSize,
			widget.TableColumn{Title: tr.T("app.col.app"), Flex: 3, Sortable: true},
			widget.TableColumn{Title: tr.T("app.col.state"), Flex: 2},
			widget.TableColumn{Title: tr.T("app.col.version"), Flex: 1.4})
		table.ID = l.tableID
		for _, t := range apps {
			table.Rows = append(table.Rows, []string{t.Title, h.stateText(t), t.Version})
		}
		btn := widget.NewButton(l.action)
		btn.ID = l.buttonID
		btn.Disabled = true
		btn.Style.Padding = geom.XY(0, 20)
		var m *widget.Modal
		table.OnSelect = func(*widget.Event) {
			btn.Disabled = table.Selected < 0
			m.Refresh()
		}
		btn.OnClick = func(*widget.Event) {
			if table.Selected >= 0 && table.Selected < len(apps) {
				m.Close()
				l.run(apps[table.Selected])
			}
		}
		body = append(body, table, btn)
		m = h.appsModal(modalID, l.title, body)
		m.Open(h.Doc)
		return
	}
	h.appsModal(modalID, l.title, body).Open(h.Doc)
}

func (h *Home) appsModal(id, title string, body []widget.Widget) *widget.Modal {
	m := widget.NewModal(title, body...)
	m.ID = id
	m.OKText = h.cfg.Tr.T("app.close")
	m.SetButtons(widget.ModalOK)
	m.SetControls(widget.ModalClose)
	return m
}
