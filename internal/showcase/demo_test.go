package showcase

import (
	"testing"
	"time"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/widget"
)

const minTarget = 56.0 // design px (~6 mm on the PB741): smallest comfortable touch target

func newApp(exit func()) (*App, *widget.FakeKeyboard) {
	kb := &widget.FakeKeyboard{Reply: "x"}
	return New(1404, 1872, Config{Version: "1.2.3", Keyboard: kb, Exit: exit}), kb
}

// walkVisible visits the visible elements of the current page that are in sight (an element
// scrolled out of the window of a Scroll is not).
func walkVisible(app *App, fn func(w widget.Widget)) {
	app.Doc.Layout()
	app.Doc.Walk(func(w widget.Widget) bool {
		if w.Elem().Visible() && widget.InView(w) {
			fn(w)
		}
		return true
	})
}

// scrollOf is the Scroll of the current page (nil when the page has none).
func scrollOf(app *App) *widget.Scroll {
	var found *widget.Scroll
	app.Doc.Walk(func(w widget.Widget) bool {
		if s, ok := w.(*widget.Scroll); ok && found == nil {
			found = s
		}
		return true
	})
	return found
}

// eachScrollPosition runs fn with the page at every position of its Scroll, a page at a time
// from the top to the end (once when the page does not scroll), and puts it back at the top.
func eachScrollPosition(app *App, fn func()) {
	app.Doc.Layout()
	s := scrollOf(app)
	if s == nil {
		fn()
		return
	}
	defer func() { s.ScrollTo(0); app.Doc.Layout() }()
	for y := 0.0; ; y += s.PageSize() {
		s.ScrollTo(y)
		app.Doc.Layout()
		fn()
		if y >= s.MaxOffset() {
			return
		}
	}
}

func TestEveryInteractiveElementReceivesItsOwnTouch(t *testing.T) {
	app, _ := newApp(nil)
	for i, title := range app.Pages() {
		app.ShowPage(i)
		checked := map[widget.Widget]bool{}
		eachScrollPosition(app, func() { checkTouches(t, app, title, checked) })
		all := 0
		app.Doc.Walk(func(w widget.Widget) bool {
			if _, isScroll := w.(*widget.Scroll); !isScroll && w.Elem().Visible() {
				if _, ok := w.(widget.PointerHandler); ok {
					all++
					if !checked[w] {
						t.Errorf("%s: %T (id %q) was never reachable, even scrolling", title, w, w.Elem().ID)
					}
				}
			}
			return true
		})
		if all < 3 {
			t.Errorf("%s: only %d interactive elements found", title, all)
		}
	}
}

// checkTouches checks the touch-accuracy rule on the interactive elements in sight and records them.
func checkTouches(t *testing.T, app *App, title string, checked map[widget.Widget]bool) {
	t.Helper()
	{
		walkVisible(app, func(w widget.Widget) {
			if _, ok := w.(widget.PointerHandler); !ok {
				return
			}
			if _, isScroll := w.(*widget.Scroll); isScroll {
				return // a window onto the others: its own center is theirs
			}
			checked[w] = true
			b := w.Elem()
			c := b.Bounds.Center()
			if got := app.Doc.TargetAt(c); got != w {
				t.Errorf("%s: a touch at the center %v of %T (id %q) reaches %T", title, c, w, b.ID, got)
			}
			if b.Bounds.W < minTarget || b.Bounds.H < minTarget {
				t.Errorf("%s: %T (id %q) is %.0fx%.0f, smaller than a %v px touch target", title, w, b.ID, b.Bounds.W, b.Bounds.H, minTarget)
			}
		})
	}
}

func TestNothingOverflowsTheScreenOrCoversTheFooter(t *testing.T) {
	app, _ := newApp(nil)
	d := app.Doc
	for i, title := range app.Pages() {
		app.ShowPage(i)
		d.Layout()
		footer := d.Root.Elem().Children()[1].Elem().Bounds
		if footer.Bottom() > d.H+0.01 {
			t.Errorf("%s: the footer is pushed off the screen (bottom %.0f > %.0f): the page is too tall", title, footer.Bottom(), d.H)
		}
		body := app.body.Bounds
		eachScrollPosition(app, func() {
			walkVisible(app, func(w widget.Widget) {
				b := w.Elem().Bounds
				if b.Right() > d.W+0.01 || b.X < -0.01 {
					t.Errorf("%s: %T (id %q) is outside horizontally: %+v", title, w, w.Elem().ID, b)
				}
				if _, ok := w.(widget.PointerHandler); ok && b.Bottom() > body.Bottom()+0.01 {
					t.Errorf("%s: %T (id %q) hangs below the page area: bottom %.0f > %.0f", title, w, w.Elem().ID, b.Bottom(), body.Bottom())
				}
			})
		})
	}
}

func tapPoint(app *App, p geom.Point) {
	app.Doc.HandlePointer(widget.PointerDown, p.X*app.Doc.Scale, p.Y*app.Doc.Scale)
	app.Doc.HandlePointer(widget.PointerUp, p.X*app.Doc.Scale, p.Y*app.Doc.Scale)
}

func byID(t *testing.T, app *App, id string) widget.Widget {
	t.Helper()
	w := app.Doc.GetElementByID(id)
	if w == nil {
		t.Fatalf("element %q not found on the current page", id)
	}
	return w
}

func TestMenuBarSwitchesPagesAndExits(t *testing.T) {
	exited := false
	app, _ := newApp(func() { exited = true })
	bar := byID(t, app, "menubar").(*widget.MenuBar)
	app.Doc.Layout()

	tapPoint(app, bar.MenuRect(2).Center()) // "Dados"
	if app.tabs.Current() != 2 || !bar.Menus[2].Selected || bar.Menus[0].Selected {
		t.Fatalf("current=%d selected flags wrong", app.tabs.Current())
	}
	if byID(t, app, "table-books") == nil {
		t.Fatal("the Dados page must be shown")
	}

	tapPoint(app, bar.MenuRect(app.tabs.Count()).Center()) // "Mais" opens its menu
	if !app.Doc.HasPopup() {
		t.Fatal("Mais must open a popup")
	}
	// the last row is "Sair"
	pop := app.Doc.TargetAt(geom.Point{X: bar.MenuRect(app.tabs.Count()).X + 20, Y: bar.Bounds.Bottom() + 6 + 3*76 + 26 /* About, Modal, Language, then a separator */ + 76/2 + 6})
	if pop == nil {
		t.Fatal("no popup row under the touch")
	}
	tapPoint(app, geom.Point{X: bar.MenuRect(app.tabs.Count()).X + 20, Y: bar.Bounds.Bottom() + 6 + 6 + 3*76 + 26 /* About, Modal, Language, then a separator */ + 76/2})
	if !exited {
		t.Error("“Sair” must call Exit")
	}
}

func TestInputsPageFlows(t *testing.T) {
	app, kb := newApp(nil)
	app.Doc.Layout()

	name := byID(t, app, "input-name")
	kb.Reply = "Ana"
	tapPoint(app, name.Elem().Bounds.Center())
	if got := name.(*widget.Input).Value; got != "Ana" || app.message.Text != "Nome: Ana" {
		t.Errorf("Value=%q message=%q", got, app.message.Text)
	}

	qty := byID(t, app, "input-qty").(*widget.Input)
	plus := qty.Elem().Bounds
	tapPoint(app, geom.Point{X: plus.Right() - 40, Y: plus.Center().Y}) // the “+” button
	if qty.Value != "2" {
		t.Errorf("qty = %q, want 2", qty.Value)
	}

	if app.touch.Text == "" || app.touch.Text[:2] != "v1" {
		t.Errorf("touch footer = %q", app.touch.Text)
	}
}

func TestEveryPageRendersAtOtherResolutions(t *testing.T) {
	for _, size := range [][2]int{{758, 1024}, {1072, 1448}, {1404, 1872}} {
		app := New(size[0], size[1], Config{Version: "x", Keyboard: &widget.FakeKeyboard{}})
		for i := range app.Pages() {
			app.ShowPage(i)
			if img := app.Doc.Render(); img.Bounds().Dx() != size[0] || img.Bounds().Dy() != size[1] {
				t.Errorf("%v page %d: image %v", size, i, img.Bounds())
			}
		}
	}
}

func TestDemoRemembersValuesAcrossTabs(t *testing.T) {
	app, _ := newApp(nil)
	byID(t, app, "input-name").(*widget.Input).Value = "Ana"
	byID(t, app, "input-qty").(*widget.Input).Value = "7"

	app.ShowPage(2) // Dados
	tbl := byID(t, app, "table-books").(*widget.Table)
	tbl.Page, tbl.SortCol = 1, 2
	app.ShowPage(3) // Calendário
	cal := byID(t, app, "calendar").(*widget.Calendar)
	cal.Selected = time.Date(2026, 5, 20, 0, 0, 0, 0, time.Local)
	app.ShowPage(1) // Escolhas
	app.ShowPage(0)

	if byID(t, app, "input-name").(*widget.Input).Value != "Ana" || byID(t, app, "input-qty").(*widget.Input).Value != "7" {
		t.Error("typed values must come back when returning to the tab")
	}
	app.ShowPage(2)
	if tbl := byID(t, app, "table-books").(*widget.Table); tbl.Page != 1 || tbl.SortCol != 2 {
		t.Errorf("table page/sort lost: %d %d", tbl.Page, tbl.SortCol)
	}
	app.ShowPage(3)
	if got := byID(t, app, "calendar").(*widget.Calendar).Selected.Format("2006-01-02"); got != "2026-05-20" {
		t.Errorf("date lost: %s", got)
	}
	if got := app.Doc.GetElementByID("calendar"); got == nil {
		t.Fatal("calendar missing")
	}
	app.Doc.Layout()
}

func TestAboutOpensAModalWithOK(t *testing.T) {
	app, _ := newApp(nil)
	app.Doc.Layout()
	bar := byID(t, app, "menubar").(*widget.MenuBar)
	more := bar.MenuRect(app.tabs.Count())

	tapPoint(app, more.Center())                                                     // "Mais"
	tapPoint(app, geom.Point{X: more.X + 20, Y: bar.Bounds.Bottom() + 6 + 6 + 76/2}) // "Sobre"
	if !app.Doc.HasModal() || app.Doc.Modal().ID != "modal-about" {
		t.Fatal("“Sobre” must open the About modal")
	}
	app.Doc.Layout()
	m := app.Doc.Modal()
	if m.Title != "Sobre" || !m.Elem().Visible() {
		t.Errorf("title=%q", m.Title)
	}
	// the page behind does not react while the modal is open
	before := app.tabs.Current()
	tapPoint(app, bar.MenuRect(1).Center())
	if app.tabs.Current() != before {
		t.Error("the page behind a modal must not receive touches")
	}
	// the OK button is the only footer button and closes it
	var ok *widget.Button
	app.Doc.Walk(func(w widget.Widget) bool {
		if b, isButton := w.(*widget.Button); isButton && b.Text == "OK" {
			ok = b
		}
		return true
	})
	if ok == nil {
		t.Fatal("the About modal has an OK button")
	}
	tapPoint(app, ok.Bounds.Center())
	if app.Doc.HasModal() {
		t.Error("OK must close the modal")
	}
}

func TestOptionsModalHasEveryControlAndReportsItsResult(t *testing.T) {
	app, _ := newApp(nil)
	app.optionsModal()
	app.Doc.Layout()
	m := app.Doc.Modal()
	if byID(t, app, "modal-input") == nil {
		t.Fatal("the modal's content is reachable through GetElementByID")
	}
	buttons := 0
	app.Doc.Walk(func(w widget.Widget) bool {
		if _, isButton := w.(*widget.Button); isButton {
			buttons++
		}
		return true
	})
	if buttons != 5 { // minimize, maximize, close, Cancelar, Salvar
		t.Errorf("buttons = %d, want 5", buttons)
	}
	m.Confirm()
	if app.message.Text != "Modal: salvo (\"\", avisos: true)" {
		t.Errorf("message = %q", app.message.Text)
	}
	app.optionsModal()
	app.Doc.Modal().Cancel()
	if app.message.Text != "Modal: cancelado" || app.Doc.HasModal() {
		t.Errorf("message = %q open=%v", app.message.Text, app.Doc.HasModal())
	}
}

func TestPageKeysMoveBetweenPagesAndWrap(t *testing.T) {
	app, _ := newApp(nil)
	app.MovePage(1)
	if app.tabs.Current() != 1 {
		t.Fatalf("next = %d", app.tabs.Current())
	}
	app.MovePage(-2) // wraps to the last page
	if app.tabs.Current() != app.tabs.Count()-1 {
		t.Errorf("previous from the first = %d", app.tabs.Current())
	}
}

func TestEveryTextIsTranslatedAndEnglishIsComplete(t *testing.T) {
	app, _ := newApp(nil)
	for _, lang := range []string{"pt-BR", "en"} {
		if missing := app.tr.Missing(lang); len(missing) != 0 {
			t.Errorf("%s lacks translations: %v", lang, missing)
		}
	}
}

func TestSwitchingLanguageTranslatesTheScreenAndKeepsWhatWasTyped(t *testing.T) {
	var saved string
	kb := &widget.FakeKeyboard{Reply: "Ana"}
	app := New(1404, 1872, Config{Version: "1.2.3", Keyboard: kb, OnLangChange: func(l string) { saved = l }})
	app.Doc.Layout()
	tapPoint(app, byID(t, app, "input-name").Elem().Bounds.Center()) // type a name in Portuguese
	if app.Pages()[0] != "Entradas" {
		t.Fatalf("starts in Portuguese: %v", app.Pages())
	}

	app.toggleLanguage()
	if app.Lang() != "en" || saved != "en" {
		t.Fatalf("lang=%q saved=%q", app.Lang(), saved)
	}
	if got := app.Pages(); got[0] != "Inputs" || got[3] != "Calendar" {
		t.Errorf("page titles = %v", got)
	}
	if got := byID(t, app, "input-name").(*widget.Input); got.Value != "Ana" || got.Placeholder != "Full name" {
		t.Errorf("the typed value survives the language change and the placeholder is translated: %q %q", got.Value, got.Placeholder)
	}
	if byID(t, app, "input-price").(*widget.Input).Prefix != "$ " {
		t.Error("number formats follow the language")
	}

	app.ShowPage(3)
	if byID(t, app, "calendar").(*widget.Calendar).Locale != widget.LocaleEN {
		t.Error("the calendar speaks English")
	}
	app.ShowPage(2)
	if tbl := byID(t, app, "table-books").(*widget.Table); tbl.Columns[0].Title != "Book" || tbl.Summary(1, 5, 12) != "1–5 of 12" {
		t.Errorf("table: %q %q", tbl.Columns[0].Title, tbl.Summary(1, 5, 12))
	}

	app.toggleLanguage() // and back
	if app.Lang() != "pt-BR" || saved != "pt-BR" || app.Pages()[2] != "Dados" {
		t.Errorf("back to Portuguese: %q %q %v", app.Lang(), saved, app.Pages())
	}
}

func TestStartingInEnglishAndTheLanguageMenuItem(t *testing.T) {
	app := New(1404, 1872, Config{Version: "1", Lang: "en"})
	if app.Pages()[1] != "Choices" {
		t.Errorf("Config.Lang starts the demo in English: %v", app.Pages())
	}
	app.Doc.Layout()
	bar := byID(t, app, "menubar").(*widget.MenuBar)
	more := bar.MenuRect(app.tabs.Count())
	tapPoint(app, more.Center())
	// the third row of the "More" menu is the language switch (it names the OTHER language)
	tapPoint(app, geom.Point{X: more.X + 20, Y: bar.Bounds.Bottom() + 6 + 6 + 2*76 + 76/2})
	if app.Lang() != "pt-BR" || app.Pages()[1] != "Escolhas" {
		t.Errorf("the menu item must switch back to Portuguese: %q %v", app.Lang(), app.Pages())
	}
}

func TestAccountPageSignsInWithTheSimulatedProviders(t *testing.T) {
	app, _ := newApp(nil)
	app.SetLang("en")
	app.ShowPage(4)
	app.Doc.Layout()
	tapPoint(app, byID(t, app, "oauth-microsoft").Elem().Bounds.Center())
	app.Doc.Layout()
	if byID(t, app, "oauth-code").(*widget.Label).Text != "HELLO-2024" {
		t.Fatal("the device code is shown")
	}
	for i, want := range []string{"Not approved yet. Try again in a moment.", "Signed in."} {
		tapPoint(app, byID(t, app, "oauth-complete").Elem().Bounds.Center())
		app.Doc.Layout()
		if got := byID(t, app, "oauth-note").(*widget.Label).Text; got != want {
			t.Errorf("check %d: %q, want %q", i+1, got, want)
		}
	}
}

func TestChoicesPageScrollsAndHoldsTheLongListCard(t *testing.T) {
	app, _ := newApp(nil)
	app.SetLang("en")
	app.ShowPage(1)
	app.Doc.Layout()
	s := scrollOf(app)
	if s == nil || s.MaxOffset() <= 0 {
		t.Fatal("the Choices page is taller than the screen and scrolls")
	}
	combo := byID(t, app, "combo-language")
	if widget.InView(combo) {
		t.Fatal("the long list card is below the first window")
	}
	s.ScrollIntoView(combo)
	app.Doc.Layout()
	if !widget.InView(combo) {
		t.Fatal("scrolling brings it into view")
	}
	tapPoint(app, combo.Elem().Bounds.Center())
	if app.Doc.HasModal() {
		t.Error("the long combo box opens a popup, not a window")
	}
}
