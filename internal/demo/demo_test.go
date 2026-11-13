package demo

import (
	"testing"

	"go-pocketbook-base/internal/geom"
	"go-pocketbook-base/internal/widget"
)

const minTarget = 56.0 // design px (~6 mm on the PB741): smallest comfortable touch target

func newApp(exit func()) (*App, *widget.FakeKeyboard) {
	kb := &widget.FakeKeyboard{Reply: "x"}
	return New(1404, 1872, Config{Version: "1.2.3", Keyboard: kb, Exit: exit}), kb
}

// walkVisible visits the visible elements of the current page.
func walkVisible(app *App, fn func(w widget.Widget)) {
	app.Doc.Layout()
	app.Doc.Walk(func(w widget.Widget) bool {
		if w.Elem().Visible() {
			fn(w)
		}
		return true
	})
}

func TestEveryInteractiveElementReceivesItsOwnTouch(t *testing.T) {
	app, _ := newApp(nil)
	for i, title := range app.Pages() {
		app.ShowPage(i)
		checked := 0
		walkVisible(app, func(w widget.Widget) {
			if _, ok := w.(widget.PointerHandler); !ok {
				return
			}
			checked++
			b := w.Elem()
			c := b.Bounds.Center()
			if got := app.Doc.TargetAt(c); got != w {
				t.Errorf("%s: a touch at the center %v of %T (id %q) reaches %T", title, c, w, b.ID, got)
			}
			if b.Bounds.W < minTarget || b.Bounds.H < minTarget {
				t.Errorf("%s: %T (id %q) is %.0fx%.0f, smaller than a %v px touch target", title, w, b.ID, b.Bounds.W, b.Bounds.H, minTarget)
			}
		})
		if checked < 3 {
			t.Errorf("%s: only %d interactive elements found", title, checked)
		}
	}
}

func TestNothingOverflowsTheScreenOrCoversTheFooter(t *testing.T) {
	app, _ := newApp(nil)
	d := app.Doc
	for i, title := range app.Pages() {
		app.ShowPage(i)
		d.Layout()
		footer := d.Root.Elem().Children()[2].Elem().Bounds
		if footer.Bottom() > d.H+0.01 {
			t.Errorf("%s: the footer is pushed off the screen (bottom %.0f > %.0f): the page is too tall", title, footer.Bottom(), d.H)
		}
		body := app.body.Bounds
		walkVisible(app, func(w widget.Widget) {
			b := w.Elem().Bounds
			if b.Right() > d.W+0.01 || b.X < -0.01 {
				t.Errorf("%s: %T (id %q) is outside horizontally: %+v", title, w, w.Elem().ID, b)
			}
			if _, ok := w.(widget.PointerHandler); ok && b.Bottom() > body.Bottom()+0.01 {
				t.Errorf("%s: %T (id %q) hangs below the page area: bottom %.0f > %.0f", title, w, w.Elem().ID, b.Bottom(), body.Bottom())
			}
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
	if app.current != 2 || !bar.Menus[2].Selected || bar.Menus[0].Selected {
		t.Fatalf("current=%d selected flags wrong", app.current)
	}
	if byID(t, app, "table-books") == nil {
		t.Fatal("the Dados page must be shown")
	}

	tapPoint(app, bar.MenuRect(len(app.pages)).Center()) // "Mais" opens its menu
	if !app.Doc.HasPopup() {
		t.Fatal("Mais must open a popup")
	}
	// the last row is "Sair"
	pop := app.Doc.TargetAt(geom.Point{X: bar.MenuRect(len(app.pages)).X + 20, Y: bar.Bounds.Bottom() + 6 + 76 + 26 + 76/2 + 6})
	if pop == nil {
		t.Fatal("no popup row under the touch")
	}
	tapPoint(app, geom.Point{X: bar.MenuRect(len(app.pages)).X + 20, Y: bar.Bounds.Bottom() + 6 + 6 + 76 + 26 + 76/2})
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
