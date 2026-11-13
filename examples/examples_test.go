package examples

import (
	"testing"
	"time"

	"go-pocketbook-base/internal/geom"
	"go-pocketbook-base/internal/widget"
)

func newDoc(content widget.Widget) (*widget.Document, *widget.FakeKeyboard) {
	kb := &widget.FakeKeyboard{Reply: "7"}
	doc := Screen(1404, 1872, kb, content)
	doc.Layout()
	return doc, kb
}

func tapAt(doc *widget.Document, p geom.Point) {
	doc.HandlePointer(widget.PointerDown, p.X*doc.Scale, p.Y*doc.Scale)
	doc.HandlePointer(widget.PointerUp, p.X*doc.Scale, p.Y*doc.Scale)
}

func tap(doc *widget.Document, w widget.Widget) { tapAt(doc, w.Elem().Bounds.Center()) }

func inked(doc *widget.Document) int {
	n := 0
	for _, p := range doc.Render().Pix {
		if p < 128 {
			n++
		}
	}
	return n
}

// interactive returns the elements that react to touch.
func interactive(doc *widget.Document) []widget.Widget {
	var out []widget.Widget
	doc.Walk(func(w widget.Widget) bool {
		if _, ok := w.(widget.PointerHandler); ok && w.Elem().Visible() {
			out = append(out, w)
		}
		return true
	})
	return out
}

func TestEveryComponentExampleRendersAndRespondsToTouch(t *testing.T) {
	examples := map[string]func() widget.Widget{
		"FlexRow": FlexRow, "FlexGrow": FlexGrow, "FlexWrap": FlexWrap, "FlexColumn": FlexColumn,
		"GridLayout": GridLayout, "LabelExample": LabelExample, "ButtonExample": ButtonExample,
		"ToggleExample": ToggleExample, "ComboBoxExample": ComboBoxExample, "DropdownExample": DropdownExample,
		"MenuBarExample": MenuBarExample, "TextInputExample": TextInputExample, "NumberInputExample": NumberInputExample,
		"MaskedInputExample": MaskedInputExample, "SearchInputExample": SearchInputExample,
		"TextAreaExample": TextAreaExample, "TableExample": TableExample, "ProgressExample": ProgressExample,
		"GaugeExample": GaugeExample, "SliderExample": SliderExample, "SlideButtonExample": SlideButtonExample,
		"ChipsExample": ChipsExample, "CalendarExample": CalendarExample, "CardExample": CardExample,
	}
	for name, build := range examples {
		doc, _ := newDoc(build())
		if inked(doc) == 0 {
			t.Errorf("%s draws nothing", name)
		}
		for _, w := range interactive(doc) {
			if w.Elem().Disabled {
				continue // a disabled element correctly ignores touch
			}
			if got := doc.TargetAt(w.Elem().Bounds.Center()); got != w {
				t.Errorf("%s: a touch at the center of %T reaches %T", name, w, got)
			}
			tap(doc, w) // runs the example's handlers
			doc.ClosePopups()
		}
	}
}

func TestHello(t *testing.T) {
	closed := false
	doc, _ := newDoc(Hello(func() { closed = true }))
	for _, w := range interactive(doc) {
		tap(doc, w)
	}
	if !closed {
		t.Error("the button must call onClose")
	}
}

func TestLoginFormValidatesBeforeCallingOnLogin(t *testing.T) {
	var user string
	var remember bool
	f := NewLoginForm(func(u string, r bool) { user, remember = u, r })
	doc, kb := newDoc(f.Root)
	submit := interactive(doc)[len(interactive(doc))-1] // the last element is the button

	tap(doc, submit)
	if f.Message.Hidden || f.Message.Text != "Type your user name" || user != "" {
		t.Fatalf("empty user: message=%q hidden=%v", f.Message.Text, f.Message.Hidden)
	}

	kb.Reply = "ana"
	tap(doc, f.User)
	kb.Reply = "123"
	tap(doc, f.Password)
	tap(doc, submit)
	if f.Message.Text != "The password needs at least 4 characters" {
		t.Fatalf("short password: %q", f.Message.Text)
	}

	kb.Reply = "1234"
	tap(doc, f.Password)
	tap(doc, f.Remember) // untick "Remember me"
	tap(doc, submit)
	if user != "ana" || remember || !f.Message.Hidden {
		t.Errorf("login: user=%q remember=%v hidden=%v", user, remember, f.Message.Hidden)
	}
}

func TestSettings(t *testing.T) {
	s := NewSettings()
	doc, _ := newDoc(s.Root)

	tap(doc, s.WiFi)
	if s.WiFi.Checked {
		t.Error("Wi-Fi must switch off")
	}

	b := s.Brightness.Elem().Bounds
	tapAt(doc, geom.Point{X: b.Right() - 40, Y: b.Center().Y})
	if s.Brightness.Value < 90 {
		t.Errorf("brightness after a tap near the right end = %v", s.Brightness.Value)
	}

	tap(doc, s.Language)
	doc.ClosePopups()
	if s.Language.Value() != "pt" {
		t.Errorf("language = %q", s.Language.Value())
	}

	tap(doc, s.Topics.Children()[1])
	if got := s.Topics.SelectedLabels(); len(got) != 1 || got[0] != "Science" {
		t.Errorf("topics = %v", got)
	}
}

func TestCatalogSearchSortAndDetail(t *testing.T) {
	rows := [][]string{{"Dom Casmurro", "Machado", "256"}, {"Iracema", "Alencar", "160"}, {"Vidas Secas", "Graciliano", "176"}}
	c := NewCatalog(rows)
	doc, kb := newDoc(c.Root)
	if len(c.Table.Rows) != 3 {
		t.Fatalf("rows = %d", len(c.Table.Rows))
	}

	kb.Reply = "casmurro"
	tap(doc, c.Search)
	if len(c.Table.Rows) != 1 || c.Table.Rows[0][0] != "Dom Casmurro" {
		t.Fatalf("filtered rows = %v", c.Table.Rows)
	}

	firstRow := geom.Point{X: c.Table.Bounds.Center().X, Y: c.Table.Bounds.Y + 3 + 76 + 35} // header is 76 px tall
	tapAt(doc, firstRow)
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
	doc, _ := newDoc(p.Root)
	if p.chosen.Text != "No date chosen" {
		t.Errorf("initial text = %q", p.chosen.Text)
	}
	tap(doc, interactive(doc)[len(interactive(doc))-1]) // the "Today" button
	if picked.IsZero() || p.Calendar.Selected.IsZero() || p.chosen.Text == "No date chosen" {
		t.Errorf("Today: picked=%v text=%q", picked, p.chosen.Text)
	}
	tap(doc, p.Calendar) // a day in the middle of the month
	if p.Calendar.Selected.IsZero() {
		t.Error("tapping a day selects it")
	}
}

func TestOrderFormRecalculatesTheTotal(t *testing.T) {
	f := NewOrderForm()
	doc, kb := newDoc(f.Root)
	if got := f.Total(); got < 19.89 || got > 19.91 {
		t.Fatalf("initial total = %v", got)
	}
	f.Quantity.StepUp()
	if got := f.Total(); got < 39.79 || got > 39.81 {
		t.Errorf("2 items = %v", got)
	}
	kb.Reply = "123456"
	tap(doc, f.Coupon) // a complete coupon takes 10% off
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
	tapAt(doc, bar.MenuRect(1).Center())
	doc.Layout()
	if !has("Wi-Fi") || has("Sign in") || !bar.Menus[1].Selected || bar.Menus[0].Selected {
		t.Error("the second page is Settings")
	}
	tapAt(doc, bar.MenuRect(2).Center())
	doc.Layout()
	if has("Wi-Fi") || !bar.Menus[2].Selected {
		t.Error("the third page is Catalog")
	}
}

func TestCustomWidgets(t *testing.T) {
	badge := NewBadge(7)
	counter := NewTapCounter()
	changes := 0
	counter.OnChange = func(e *widget.Event) { changes++ }
	doc, _ := newDoc(widget.NewColumn(20, badge, counter))
	if badge.Bounds.W <= 0 || badge.Bounds.H <= 0 || inked(doc) == 0 {
		t.Fatal("the badge must be laid out and drawn")
	}
	if doc.TargetAt(counter.Bounds.Center()) != widget.Widget(counter) {
		t.Fatal("the counter must receive its own touches")
	}
	tap(doc, counter)
	tap(doc, counter)
	if counter.Count != 2 || changes != 2 || counter.label() != "Taps: 2" {
		t.Errorf("Count=%d changes=%d label=%q", counter.Count, changes, counter.label())
	}
	if doc.TargetAt(badge.Bounds.Center()) == widget.Widget(badge) {
		t.Error("a badge is display-only and must not capture touches")
	}
}
