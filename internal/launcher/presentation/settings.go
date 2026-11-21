package presentation

import (
	"fmt"
	"strconv"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/launcher/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

// Language is a language the person can choose: the tag the apps read from the
// shared settings and the name it is shown by (in its own language).
type Language struct{ Tag, Label string }

// DefaultLanguages are the languages of the ecosystem.
var DefaultLanguages = []Language{{"pt-BR", "Português (Brasil)"}, {"en", "English"}}

// iconStep is the step of the icon size slider (the default size is a multiple of it from the minimum).
const iconStep = 6

// noBackground is the value of the "none" option of the background list.
const noBackground = ""

// settingsForm holds the fields of the settings screen.
type settingsForm struct {
	columns, rows *widget.Input
	size          *widget.Slider
	sizeLabel     *widget.Label
	lang, bg      *widget.ComboBox
	problem       *widget.Label
}

// InSettings reports whether the settings screen is the one showing.
func (h *Home) InSettings() bool { return h.settings != nil }

// OpenSettings shows the settings screen, filled with the current values.
func (h *Home) OpenSettings() {
	if h.settings != nil {
		return
	}
	root := h.buildSettings()
	h.show2(root)
}

// CloseSettings goes back to the home screen without saving.
func (h *Home) CloseSettings() {
	if h.settings == nil {
		return
	}
	h.settings = nil
	h.show2(h.root)
}

// show2 puts root on the screen and repaints all of it.
func (h *Home) show2(root widget.Widget) {
	h.Doc.SetRoot(root)
	h.Doc.ResetFrame()
	h.Doc.Invalidate()
}

func (h *Home) buildSettings() widget.Widget {
	tr, prefs := h.cfg.Tr, h.cfg.Prefs
	f := &settingsForm{}
	h.settings = f

	f.columns = h.numberField("settings-columns", prefs.Grid.Columns)
	f.rows = h.numberField("settings-rows", prefs.Grid.Rows)
	grid := widget.NewCard(tr.T("settings.grid"), "",
		h.captioned(tr.T("settings.columns"), f.columns), h.captioned(tr.T("settings.rows"), f.rows))

	f.size = widget.NewSlider(domain.MinIconSize, domain.MaxIconSize, iconStep, float64(prefs.IconSize))
	f.size.ID = "settings-size"
	f.size.Style.FlexGrow = 1
	f.sizeLabel = widget.NewLabel(fmt.Sprintf("%.0f", f.size.Value))
	f.sizeLabel.Style.Width = widget.Px(90)
	f.size.OnInput = func(*widget.Event) { f.sizeLabel.Text = fmt.Sprintf("%.0f", f.size.Value) }
	icons := widget.NewCard(tr.T("settings.icons"), "", widget.NewRow(12, f.size, f.sizeLabel))

	langs := make([]widget.Option, len(h.cfg.Languages))
	for i, l := range h.cfg.Languages {
		langs[i] = widget.Option{Label: l.Label, Value: l.Tag}
	}
	f.lang = widget.NewComboBox(tr.T("settings.language"), langs...)
	f.lang.ID = "settings-language"
	f.lang.SetValue(tr.Lang())
	language := widget.NewCard(tr.T("settings.language"), "", f.lang)

	bgOptions := []widget.Option{{Label: tr.T("settings.background.none"), Value: noBackground}}
	var names []string
	if h.cfg.Backgrounds != nil {
		names = h.cfg.Backgrounds()
	}
	for _, name := range names {
		bgOptions = append(bgOptions, widget.Option{Label: name, Value: name})
	}
	f.bg = widget.NewComboBox(tr.T("settings.background"), bgOptions...)
	f.bg.ID = "settings-background"
	f.bg.SetValue(prefs.Image)
	background := []widget.Widget{f.bg}
	if len(names) == 0 {
		hint := widget.NewLabel(tr.T("settings.background.hint"))
		hint.Style.FontSize, hint.Style.Color = widget.FontSmall, widget.Muted
		background = append(background, hint)
	}
	bgCard := widget.NewCard(tr.T("settings.background"), "", background...)

	manage := h.button("settings-apps", tr.T("settings.apps.manage"), h.OpenManager)
	hidden := h.button("settings-hidden", tr.T("settings.apps.hidden"), h.OpenHidden)
	appsCard := widget.NewCard(tr.T("settings.apps"), "", widget.NewRow(12, manage, hidden))

	about := h.button("settings-about", tr.T("about.title"), h.OpenAbout)
	system := h.button("settings-system", tr.T("system.title"), h.OpenSystemInfo)
	aboutCard := widget.NewCard(tr.T("settings.about"), "", widget.NewRow(12, about, system))

	f.problem = widget.NewLabel("")
	f.problem.ID = "settings-problem"
	f.problem.Style.Color = widget.Muted

	save := h.button("settings-save", tr.T("settings.save"), h.saveSettings)
	cancel := h.button("settings-cancel", tr.T("settings.cancel"), h.CloseSettings)
	reset := h.button("settings-reset", tr.T("settings.reset"), h.resetSettings)
	buttons := widget.NewRow(12, reset, cancel, save)

	title := widget.NewLabel(tr.T("settings.title"))
	title.Style.FontSize, title.Style.Bold = widget.FontTitle, true
	title.Style.Padding = geom.XY(28, 28)

	// The cards are taller than the screen: a window with a scroll bar shows them.
	body := widget.NewScroll(grid, icons, language, bgCard, appsCard, aboutCard, f.problem)
	body.ID = "settings-scroll"
	body.Content.Style.Gap = 20
	body.Content.Style.Padding = geom.XY(0, 28)
	body.Style.FlexGrow = 1
	buttons.Style.Padding = geom.XY(20, 28)

	root := widget.NewColumn(0, title, body, buttons)
	root.Style.Background = widget.Paper
	return root
}

// numberField is a stepper for a side of the grid.
func (h *Home) numberField(id string, value int) *widget.Input {
	in := widget.NewNumberInput(float64(value), 0)
	in.ID = id
	in.Min, in.Max, in.Step = domain.MinSide, domain.MaxSide, 1
	in.Style.FlexGrow = 1
	return in
}

// captioned puts a small caption above a field.
func (h *Home) captioned(caption string, field widget.Widget) widget.Widget {
	l := widget.NewLabel(caption)
	l.Style.FontSize, l.Style.Color = widget.FontSmall, widget.Muted
	col := widget.NewColumn(6, l, field)
	col.Style.FlexGrow = 1
	return col
}

func (h *Home) button(id, text string, onClick func()) *widget.Button {
	b := widget.NewButton(text)
	b.ID = id
	b.OnClick = func(*widget.Event) { onClick() }
	return b
}

// resetSettings puts the default values in the form (nothing is saved until Save).
func (h *Home) resetSettings() {
	f, d := h.settings, domain.DefaultPreferences()
	f.columns.Value = strconv.Itoa(d.Grid.Columns)
	f.rows.Value = strconv.Itoa(d.Grid.Rows)
	f.size.SetValue(float64(d.IconSize))
	f.sizeLabel.Text = fmt.Sprintf("%d", d.IconSize)
	f.bg.SetValue(noBackground)
	f.lang.SetValue(h.cfg.Languages[0].Tag)
	h.Doc.Invalidate()
}

// saveSettings validates the form, keeps the preferences and the language, applies
// them and goes back to the home screen. A failure stays on the form and says why.
func (h *Home) saveSettings() {
	f := h.settings
	cols, err1 := strconv.Atoi(f.columns.Value)
	rows, err2 := strconv.Atoi(f.rows.Value)
	if err1 != nil || err2 != nil {
		h.settingsProblem(h.cfg.Tr.T("settings.invalid"))
		return
	}
	prefs := domain.Preferences{
		Grid: domain.Grid{Columns: cols, Rows: rows}, IconSize: int(f.size.Value), Image: f.bg.Value(),
		HiddenApps: h.cfg.Prefs.HiddenApps, // kept: the settings form does not edit them
	}.Normalized()
	lang := f.lang.Value()
	if lang == "" {
		lang = h.cfg.Tr.Lang()
	}

	if h.cfg.SavePrefs != nil {
		if err := h.cfg.SavePrefs(prefs); err != nil {
			h.settingsProblem(h.cfg.Tr.T("settings.failed", err.Error()))
			return
		}
	}
	if h.cfg.SaveLang != nil && lang != h.cfg.Tr.Lang() {
		if err := h.cfg.SaveLang(lang); err != nil {
			h.settingsProblem(h.cfg.Tr.T("settings.failed", err.Error()))
			return
		}
	}
	h.settings = nil
	h.Apply(prefs, lang)
}

func (h *Home) settingsProblem(text string) {
	h.settings.problem.Text = text
	h.Doc.Invalidate()
}
