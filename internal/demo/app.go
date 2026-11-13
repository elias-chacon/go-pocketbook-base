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
	menu    *widget.MenuBar
	body    *widget.Panel
	message *widget.Label
	touch   *widget.Label
	pages   []page
	current int
}

type page struct {
	title string
	root  widget.Widget
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

	a.pages = []page{
		{"Entradas", a.inputsPage()},
		{"Escolhas", a.choicesPage()},
		{"Dados", a.dataPage()},
		{"Calendário", a.calendarPage()},
	}
	a.menu = widget.NewMenuBar(a.menus()...)
	a.menu.ID = "menubar"

	a.body = widget.NewColumn(0)
	a.body.Style.FlexGrow = 1
	a.body.Style.Padding = pagePadding

	footer := widget.NewRow(24, a.message, a.touch)
	footer.Style.Padding = footerPadding
	footer.Style.BorderColor, footer.Style.BorderWidth = widget.Mid, 0

	root := widget.NewColumn(0, a.menu, a.body, footer)
	root.ID, root.Style.Background = "root", widget.Paper
	root.OnClick = a.trackTouch
	a.Doc.SetRoot(root)
	a.ShowPage(0)
	return a
}

// ShowPage switches to page i.
func (a *App) ShowPage(i int) {
	a.current = i
	a.body.ClearChildren()
	a.body.AppendChild(a.pages[i].root)
	for j, m := range a.menu.Menus[:len(a.pages)] {
		m.Selected = j == i
	}
	a.Doc.ClosePopups()
}

// Pages returns the page titles.
func (a *App) Pages() []string {
	out := make([]string, len(a.pages))
	for i, p := range a.pages {
		out[i] = p.title
	}
	return out
}

func (a *App) menus() []*widget.Menu {
	var out []*widget.Menu
	for i, p := range a.pages {
		i := i
		out = append(out, &widget.Menu{Label: p.title, OnSelect: func() { a.ShowPage(i) }})
	}
	return append(out, &widget.Menu{Label: "Mais", Items: []widget.MenuItem{
		{Label: "Sobre", Icon: gfx.IconSearch, OnSelect: func() {
			a.say(fmt.Sprintf("go-pocketbook-base v%s — componentes vetoriais em Go", a.cfg.Version))
		}},
		{Separator: true},
		{Label: "Sair", Icon: gfx.IconClose, OnSelect: a.exit},
	}})
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
