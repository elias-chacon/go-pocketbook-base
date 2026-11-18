// Package showcase is the showcase screen of every widget. It is plain Go on top
// of the widget package, shared by the device app (apps/hello), the desktop
// preview (cmd/preview) and the tests. Its texts are translated (pt-BR first,
// English as the fallback) through internal/localization/i18n.
package showcase

import (
	"embed"
	"fmt"

	"go-pocketbook-base/internal/localization/i18n"
	"go-pocketbook-base/internal/toolkit/gfx"
	"go-pocketbook-base/internal/toolkit/widget"
)

//go:embed locales/*.json
var locales embed.FS

// Config wires the demo to its environment.
type Config struct {
	Version  string
	Keyboard widget.Keyboard
	Exit     func()
	// Notice is shown in the footer at start (e.g. first-run install result).
	Notice string
	// Lang is the language to start in (default pt-BR).
	Lang string
	// OnLangChange runs after the user switches the language (the device app
	// saves it in the shared settings).
	OnLangChange func(lang string)
}

// pageKeys are the translation keys of the page titles, in order.
var pageKeys = []string{"menu.inputs", "menu.choices", "menu.data", "menu.calendar"}

// App is the demo application: a menu bar, one page at a time and a footer.
type App struct {
	Doc *widget.Document

	cfg     Config
	tr      *i18n.Bundle
	tabs    *widget.Tabs
	menu    *widget.MenuBar
	body    *widget.Panel
	message *widget.Label
	touch   *widget.Label

	// refresh updates content derived from restored values; the page that
	// needs it sets it while it is built (see widget.Tab.Restored).
	refresh func()
}

// New builds the demo for a deviceW x deviceH screen.
func New(deviceW, deviceH int, cfg Config) *App {
	a := &App{cfg: cfg, tr: i18n.New()}
	if err := a.tr.AddFS(locales, "locales"); err != nil {
		panic(err) // embedded files: an error is a programming mistake
	}
	if cfg.Lang != "" {
		a.tr.SetLang(cfg.Lang)
	}
	a.Doc = widget.NewDocument(deviceW, deviceH)
	a.Doc.Keyboard = cfg.Keyboard

	a.message, a.touch = widget.NewLabel(cfg.Notice), widget.NewLabel("")
	for _, l := range []*widget.Label{a.message, a.touch} {
		l.Style.FontSize, l.Style.Color = widget.FontSmall, widget.Muted
	}
	a.touch.Text = "v" + cfg.Version
	a.touch.Style.TextAlign = gfx.AlignEnd
	a.message.Style.FlexGrow, a.touch.Style.FlexGrow = 1, 1

	// Each page is dropped when another is shown; its typed values, table page,
	// selected date... are kept in the Tabs store (a few bytes) and put back.
	build := func(f func() widget.Widget) func() widget.Widget {
		return func() widget.Widget { a.refresh = nil; return f() }
	}
	restored := func() {
		if a.refresh != nil {
			a.refresh()
		}
	}
	a.tabs = widget.NewTabs(nil,
		&widget.Tab{Title: a.t(pageKeys[0]), Build: build(a.inputsPage), Restored: restored},
		&widget.Tab{Title: a.t(pageKeys[1]), Build: build(a.choicesPage), Restored: restored},
		&widget.Tab{Title: a.t(pageKeys[2]), Build: build(a.dataPage), Restored: restored},
		&widget.Tab{Title: a.t(pageKeys[3]), Build: build(a.calendarPage), Restored: restored})
	a.menu, a.body = a.tabs.Bar, a.tabs.Body
	a.menu.ID = "menubar"
	a.menu.Menus = append(a.menu.Menus, a.moreMenu())
	a.body.Style.Padding = pagePadding
	a.tabs.Style.FlexGrow = 1

	footer := widget.NewRow(24, a.message, a.touch)
	footer.Style.Padding = footerPadding
	footer.Style.BorderColor, footer.Style.BorderWidth = widget.Mid, 0

	root := widget.NewColumn(0, a.tabs, footer)
	root.ID, root.Style.Background = "root", widget.Paper
	root.OnClick = a.trackTouch
	a.Doc.SetRoot(root)

	a.tr.OnChange = a.relocalize
	return a
}

// t translates a key (with fmt arguments).
func (a *App) t(key string, args ...interface{}) string { return a.tr.T(key, args...) }

// Lang is the current language.
func (a *App) Lang() string { return a.tr.Lang() }

// SetLang switches the language and rebuilds the screen in it.
func (a *App) SetLang(lang string) { a.tr.SetLang(lang) }

// relocalize puts every visible text in the new language. The current page is
// rebuilt through the Tabs, which keeps what the user typed and chose.
func (a *App) relocalize() {
	for i, k := range pageKeys {
		a.menu.Menus[i].Label = a.t(k)
	}
	a.menu.Menus[len(pageKeys)] = a.moreMenu()
	a.touch.Text = "v" + a.cfg.Version
	a.message.Text = ""
	a.tabs.Show(a.tabs.Current())
	a.Doc.Invalidate()
	if a.cfg.OnLangChange != nil {
		a.cfg.OnLangChange(a.tr.Lang())
	}
}

// toggleLanguage switches between Portuguese and English.
func (a *App) toggleLanguage() {
	if a.tr.Lang() == "en" {
		a.SetLang("pt-BR")
		return
	}
	a.SetLang("en")
}

// ShowPage switches to page i (see widget.Tabs.Show).
func (a *App) ShowPage(i int) { a.tabs.Show(i) }

// MovePage shows the page delta places away (the reader's previous/next keys).
func (a *App) MovePage(delta int) { a.tabs.Move(delta) }

// Pages returns the page titles.
func (a *App) Pages() []string {
	out := make([]string, a.tabs.Count())
	for i := range out {
		out[i] = a.menu.Menus[i].Label
	}
	return out
}

func (a *App) moreMenu() *widget.Menu {
	return &widget.Menu{Label: a.t("menu.more"), Items: []widget.MenuItem{
		{Label: a.t("menu.about"), Icon: gfx.IconSearch, OnSelect: a.about},
		{Label: a.t("menu.modal"), Icon: gfx.IconMaximize, OnSelect: a.optionsModal},
		{Label: a.t("menu.language"), Icon: gfx.IconMenu, OnSelect: a.toggleLanguage},
		{Separator: true},
		{Label: a.t("menu.exit"), Icon: gfx.IconClose, OnSelect: a.exit},
	}}
}

// about shows the About modal: centered, with an OK button.
func (a *App) about() {
	name := widget.NewLabel("go-pocketbook-base")
	name.Style.Bold, name.Style.FontSize = true, widget.FontTitle
	desc := widget.NewLabel(a.t("about.description"))
	version := widget.NewLabel(a.t("about.version", a.cfg.Version))
	version.Style.Color = widget.Muted
	m := widget.NewModal(a.t("about.title"), name, desc, version)
	m.ID = "modal-about"
	m.Open(a.Doc)
}

// optionsModal shows a modal with every option: OK and Cancel, and the close,
// maximize and minimize buttons (try them).
func (a *App) optionsModal() {
	nick := widget.NewInput(a.t("modal.nickname"))
	nick.ID = "modal-input"
	notify := widget.NewSwitch(a.t("modal.notices"), true)
	m := widget.NewModal(a.t("modal.title"), widget.NewLabel(a.t("modal.text")), nick, notify)
	m.ID = "modal-options"
	m.OKText, m.CancelText = a.t("modal.save"), a.t("modal.cancel")
	m.SetButtons(widget.ModalOKCancel)
	m.SetControls(widget.ModalClose | widget.ModalMaximize | widget.ModalMinimize)
	m.OnOK = func(e *widget.Event) { a.say(a.t("modal.saved", nick.Value, notify.Checked)) }
	m.OnCancel = func(e *widget.Event) { a.say(a.t("modal.cancelled")) }
	m.Open(a.Doc)
}

func (a *App) exit() {
	if a.cfg.Exit != nil {
		a.cfg.Exit()
	}
}

// say shows an already translated message in the footer.
func (a *App) say(text string) {
	a.message.Text = text
	a.Doc.Invalidate()
}

// trackTouch shows where the last tap landed and which element received it,
// which makes it easy to verify that touch and drawing agree.
func (a *App) trackTouch(e *widget.Event) {
	name := e.Target.Elem().ID
	if name == "" {
		name = fmt.Sprintf("%T", e.Target)
	}
	a.touch.Text = a.t("footer.touch", a.cfg.Version, e.X, e.Y, name)
}
