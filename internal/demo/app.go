// Package demo is the showcase screen of every widget. It is plain Go on top
// of the widget package, shared by the device app (cmd/hello), the desktop
// preview (cmd/preview) and the tests.
package demo

import (
	"fmt"

	"go-pocketbook-base/internal/gfx"
	"go-pocketbook-base/internal/widget"
)

// Config wires the demo to its environment.
type Config struct {
	Version  string
	Keyboard widget.Keyboard
	Exit     func()
	// Notice is shown in the footer at start (e.g. first-run install result).
	Notice string
}

// App is the demo application: a menu bar, one page at a time and a footer.
type App struct {
	Doc *widget.Document

	cfg     Config
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
	a := &App{cfg: cfg}
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
		&widget.Tab{Title: "Entradas", Build: build(a.inputsPage), Restored: restored},
		&widget.Tab{Title: "Escolhas", Build: build(a.choicesPage), Restored: restored},
		&widget.Tab{Title: "Dados", Build: build(a.dataPage), Restored: restored},
		&widget.Tab{Title: "Calendário", Build: build(a.calendarPage), Restored: restored})
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
	return a
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
	return &widget.Menu{Label: "Mais", Items: []widget.MenuItem{
		{Label: "Sobre", Icon: gfx.IconSearch, OnSelect: a.about},
		{Label: "Modal (opções)", Icon: gfx.IconMaximize, OnSelect: a.optionsModal},
		{Separator: true},
		{Label: "Sair", Icon: gfx.IconClose, OnSelect: a.exit},
	}}
}

// about shows the About modal: centered, with an OK button.
func (a *App) about() {
	name := widget.NewLabel("go-pocketbook-base")
	name.Style.Bold, name.Style.FontSize = true, widget.FontTitle
	desc := widget.NewLabel("Componentes visuais vetoriais em Go para aplicativos PocketBook (e-readers Linux ARM).")
	version := widget.NewLabel("Versão " + a.cfg.Version)
	version.Style.Color = widget.Muted
	m := widget.NewModal("Sobre", name, desc, version)
	m.ID = "modal-about"
	m.Open(a.Doc)
}

// optionsModal shows a modal with every option: OK and Cancel, and the close,
// maximize and minimize buttons (try them).
func (a *App) optionsModal() {
	nick := widget.NewInput("Apelido")
	nick.ID = "modal-input"
	notify := widget.NewSwitch("Receber avisos", true)
	m := widget.NewModal("Modal com opções",
		widget.NewLabel("OK e Cancelar no rodapé; fechar, maximizar e minimizar na barra de título."),
		nick, notify)
	m.ID = "modal-options"
	m.OKText, m.CancelText = "Salvar", "Cancelar"
	m.SetButtons(widget.ModalOKCancel)
	m.SetControls(widget.ModalClose | widget.ModalMaximize | widget.ModalMinimize)
	m.OnOK = func(e *widget.Event) { a.say("Modal: salvo (%q, avisos: %v)", nick.Value, notify.Checked) }
	m.OnCancel = func(e *widget.Event) { a.say("Modal: cancelado") }
	m.Open(a.Doc)
}

func (a *App) exit() {
	if a.cfg.Exit != nil {
		a.cfg.Exit()
	}
}

// say shows a message in the footer.
func (a *App) say(format string, args ...interface{}) {
	a.message.Text = fmt.Sprintf(format, args...)
	a.Doc.Invalidate()
}

// trackTouch shows where the last tap landed and which element received it,
// which makes it easy to verify that touch and drawing agree.
func (a *App) trackTouch(e *widget.Event) {
	name := e.Target.Elem().ID
	if name == "" {
		name = fmt.Sprintf("%T", e.Target)
	}
	a.touch.Text = fmt.Sprintf("v%s · toque (%.0f, %.0f) → %s", a.cfg.Version, e.X, e.Y, name)
}
