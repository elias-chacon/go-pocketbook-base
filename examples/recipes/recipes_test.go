package recipes

import (
	"testing"
	"time"

	"go-pocketbook-base/examples/harness"
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/widget"
)

func TestHello(t *testing.T) {
	closed := false
	doc, _ := harness.NewDoc(Hello(func() { closed = true }))
	for _, w := range harness.Interactive(doc) {
		harness.Tap(doc, w)
	}
	if !closed {
		t.Error("the button must call onClose")
	}
}

func TestLoginFormValidatesBeforeCallingOnLogin(t *testing.T) {
	var user string
	var remember bool
	f := NewLoginForm(func(u string, r bool) { user, remember = u, r })
	doc, kb := harness.NewDoc(f.Root)
	submit := harness.Interactive(doc)[len(harness.Interactive(doc))-1] // the last element is the button

	harness.Tap(doc, submit)
	if f.Message.Hidden || f.Message.Text != "Type your user name" || user != "" {
		t.Fatalf("empty user: message=%q hidden=%v", f.Message.Text, f.Message.Hidden)
	}

	kb.Reply = "ana"
	harness.Tap(doc, f.User)
	kb.Reply = "123"
	harness.Tap(doc, f.Password)
	harness.Tap(doc, submit)
	if f.Message.Text != "The password needs at least 4 characters" {
		t.Fatalf("short password: %q", f.Message.Text)
	}

	kb.Reply = "1234"
	harness.Tap(doc, f.Password)
	harness.Tap(doc, f.Remember) // untick "Remember me"
	harness.Tap(doc, submit)
	if user != "ana" || remember || !f.Message.Hidden {
		t.Errorf("login: user=%q remember=%v hidden=%v", user, remember, f.Message.Hidden)
	}
}

func TestSettings(t *testing.T) {
	s := NewSettings()
	doc, _ := harness.NewDoc(s.Root)

	harness.Tap(doc, s.WiFi)
	if s.WiFi.Checked {
		t.Error("Wi-Fi must switch off")
	}

	b := s.Brightness.Elem().Bounds
	harness.TapAt(doc, geom.Point{X: b.Right() - 40, Y: b.Center().Y})
	if s.Brightness.Value < 90 {
		t.Errorf("brightness after a tap near the right end = %v", s.Brightness.Value)
	}

	harness.Tap(doc, s.Language)
	doc.ClosePopups()
	if s.Language.Value() != "pt" {
		t.Errorf("language = %q", s.Language.Value())
	}

	harness.Tap(doc, s.Topics.Children()[1])
	if got := s.Topics.SelectedLabels(); len(got) != 1 || got[0] != "Science" {
		t.Errorf("topics = %v", got)
	}
}

func TestCatalogSearchSortAndDetail(t *testing.T) {
	rows := [][]string{{"Dom Casmurro", "Machado", "256"}, {"Iracema", "Alencar", "160"}, {"Vidas Secas", "Graciliano", "176"}}
	c := NewCatalog(rows)
	doc, kb := harness.NewDoc(c.Root)
	if len(c.Table.Rows) != 3 {
		t.Fatalf("rows = %d", len(c.Table.Rows))
	}

	kb.Reply = "casmurro"
	harness.Tap(doc, c.Search)
	if len(c.Table.Rows) != 1 || c.Table.Rows[0][0] != "Dom Casmurro" {
		t.Fatalf("filtered rows = %v", c.Table.Rows)
	}

	firstRow := geom.Point{X: c.Table.Bounds.Center().X, Y: c.Table.Bounds.Y + 3 + 76 + 35} // header is 76 px tall
	harness.TapAt(doc, firstRow)
	if c.Table.Selected != 0 || c.Detail.Text != "Dom Casmurro — Machado (256 pages)" {
		t.Errorf("Selected=%d detail=%q", c.Table.Selected, c.Detail.Text)
	}

	c.filter("")
	if len(c.Table.Rows) != 3 || c.Table.Selected != -1 {
		t.Error("an empty query shows everything again")
	}
}

func TestDatePicker(t *testing.T) {
	var picked time.Time
	p := NewDatePicker(time.Time{}, func(t time.Time) { picked = t })
	doc, _ := harness.NewDoc(p.Root)
	if p.chosen.Text != "No date chosen" {
		t.Errorf("initial text = %q", p.chosen.Text)
	}
	harness.Tap(doc, harness.Interactive(doc)[len(harness.Interactive(doc))-1]) // the "Today" button
	if picked.IsZero() || p.Calendar.Selected.IsZero() || p.chosen.Text == "No date chosen" {
		t.Errorf("Today: picked=%v text=%q", picked, p.chosen.Text)
	}
	harness.Tap(doc, p.Calendar) // a day in the middle of the month
	if p.Calendar.Selected.IsZero() {
		t.Error("tapping a day selects it")
	}
}

func TestOrderFormRecalculatesTheTotal(t *testing.T) {
	f := NewOrderForm()
	doc, kb := harness.NewDoc(f.Root)
	if got := f.Total(); got < 19.89 || got > 19.91 {
		t.Fatalf("initial total = %v", got)
	}
	f.Quantity.StepUp()
	if got := f.Total(); got < 39.79 || got > 39.81 {
		t.Errorf("2 items = %v", got)
	}
	kb.Reply = "123456"
	harness.Tap(doc, f.Coupon) // a complete coupon takes 10% off
	if got := f.Total(); got < 35.81 || got > 35.83 {
		t.Errorf("with coupon = %v", got)
	}
	if f.total.Text != "Total: R$ 35,82" {
		t.Errorf("label = %q", f.total.Text)
	}
}

func TestPagedAppNavigatesBetweenPages(t *testing.T) {
	doc := NewPagedApp(1404, 1872, &widget.FakeKeyboard{})
	doc.Layout()
	var bar *widget.MenuBar
	doc.Walk(func(w widget.Widget) bool {
		if b, ok := w.(*widget.MenuBar); ok {
			bar = b
		}
		return true
	})
	has := func(label string) bool {
		found := false
		doc.Walk(func(w widget.Widget) bool {
			switch x := w.(type) {
			case *widget.Toggle:
				found = found || x.Label == label
			case *widget.Button:
				found = found || x.Text == label
			}
			return !found
		})
		return found
	}

	if !has("Sign in") || !bar.Menus[0].Selected {
		t.Fatal("the first page is Login")
	}
	harness.TapAt(doc, bar.MenuRect(1).Center())
	doc.Layout()
	if !has("Wi-Fi") || has("Sign in") || !bar.Menus[1].Selected || bar.Menus[0].Selected {
		t.Error("the second page is Settings")
	}
	harness.TapAt(doc, bar.MenuRect(2).Center())
	doc.Layout()
	if has("Wi-Fi") || !bar.Menus[2].Selected {
		t.Error("the third page is Catalog")
	}
}
