package presentation

import (
	"fmt"

	"go-pocketbook-base/internal/toolkit/widget"
)

// Fact is one line of the system information: a key that the presentation translates
// ("system.<key>") and its value. The composition root gives the ones only the reader knows.
type Fact struct{ Key, Value string }

// OpenAbout shows the About window: what the Launcher is, its version and where its releases are.
func (h *Home) OpenAbout() {
	tr := h.cfg.Tr
	name := widget.NewLabel(tr.T("about.name"))
	name.Style.Bold, name.Style.FontSize = true, widget.FontTitle
	desc := widget.NewLabel(tr.T("about.description"))
	body := []widget.Widget{name, desc}
	if h.cfg.Version != "" {
		version := widget.NewLabel(tr.T("about.version", h.cfg.Version))
		version.ID = "about-version"
		version.Style.Color = widget.Muted
		body = append(body, version)
	}
	if h.cfg.Repository != "" {
		repo := widget.NewLabel(tr.T("about.repository", h.cfg.Repository))
		repo.ID = "about-repository"
		repo.Style.FontSize, repo.Style.Color = widget.FontSmall, widget.Muted
		body = append(body, repo)
	}
	h.appsModal("modal-about", tr.T("about.title"), body).Open(h.Doc)
}

// systemFacts are the lines of the system information: what the Launcher itself knows (the apps and
// the language) followed by what the reader gives.
func (h *Home) systemFacts() []Fact {
	installed := 0
	for _, t := range h.all {
		if t.Installed {
			installed++
		}
	}
	facts := []Fact{
		{"apps", fmt.Sprintf("%d", installed)},
		{"catalog", fmt.Sprintf("%d", len(h.all))},
		{"hidden", fmt.Sprintf("%d", len(h.cfg.Prefs.HiddenApps))},
		{"language", h.cfg.Tr.Lang()},
		{"screen", fmt.Sprintf("%d x %d", h.cfg.Width, h.cfg.Height)},
	}
	if h.cfg.SystemInfo != nil {
		facts = append(facts, h.cfg.SystemInfo()...)
	}
	return facts
}

// OpenSystemInfo shows the system information in a table: the version, the device, the screen, the
// storage and the apps.
func (h *Home) OpenSystemInfo() {
	tr := h.cfg.Tr
	facts := h.systemFacts()
	table := widget.NewTable(len(facts)+1,
		widget.TableColumn{Title: tr.T("system.col.item"), Flex: 2},
		widget.TableColumn{Title: tr.T("system.col.value"), Flex: 3})
	table.ID = "system-table"
	for _, f := range facts {
		table.Rows = append(table.Rows, []string{tr.T("system." + f.Key), f.Value})
	}
	h.appsModal("modal-system", tr.T("system.title"), []widget.Widget{table}).Open(h.Doc)
}
