package widget

import (
	"testing"

	"go-pocketbook-base/internal/geom"
)

// pageWithModal builds a page with a button behind a modal and returns them.
func pageWithModal() (*Document, *Button, *Modal) {
	behind := NewButton("Behind")
	d := testDoc(behind)
	m := NewModal("About", NewLabel("Some content"))
	return d, behind, m
}

// buttonLabeled finds the button of the modal with that text.
func buttonLabeled(m *Modal, text string) *Button {
	var found *Button
	walk(m, func(w Widget) bool {
		if b, ok := w.(*Button); ok && b.Text == text {
			found = b
		}
		return found == nil
	})
	return found
}

// iconButtons lists the title bar's icon-only buttons, left to right.
func iconButtons(m *Modal) []*Button {
	var out []*Button
	walk(m.children[0], func(w Widget) bool {
		if b, ok := w.(*Button); ok {
			out = append(out, b)
		}
		return true
	})
	return out
}

func TestModalIsCenteredAndOKClosesIt(t *testing.T) {
	d, _, m := pageWithModal()
	ok := &counter{}
	m.OnOK = ok.handler()
	m.Open(d)
	d.Layout()

	if !d.HasModal() || d.Modal() != m {
		t.Fatal("the modal must be open")
	}
	c := m.Bounds.Center()
	if c.X < d.W/2-1 || c.X > d.W/2+1 || c.Y < d.H/2-1 || c.Y > d.H/2+1 {
		t.Errorf("not centered: %+v on %vx%v", m.Bounds, d.W, d.H)
	}
	if m.Bounds.W != modalDefaultW {
		t.Errorf("default width = %v", m.Bounds.W)
	}
	tap(d, buttonLabeled(m, "OK"))
	if ok.n != 1 || d.HasModal() {
		t.Errorf("OK must run OnOK once and close: ran %d, open %v", ok.n, d.HasModal())
	}
}

func TestModalBlocksThePageBehindIt(t *testing.T) {
	d, behind, m := pageWithModal()
	clicks := &counter{}
	behind.OnClick = clicks.handler()
	m.Open(d)
	d.Layout()

	tap(d, behind) // the button is under the screen area, outside the modal
	tapAt(d, geomPt(5, d.H-5))
	if clicks.n != 0 {
		t.Fatal("touches outside an open modal must not reach the page")
	}
	if !d.HasModal() {
		t.Error("touching outside must not close the modal")
	}
	m.Close()
	tap(d, behind)
	if clicks.n != 1 {
		t.Error("after closing, the page works again")
	}
}

func TestModalCancelCloseAndBack(t *testing.T) {
	ways := map[string]func(d *Document, m *Modal){
		"cancel button": func(d *Document, m *Modal) { tap(d, buttonLabeled(m, "Cancel")) },
		"close button":  func(d *Document, m *Modal) { tap(d, iconButtons(m)[0]) },
		"back key":      func(d *Document, m *Modal) { d.HandleBack() },
	}
	for name, way := range ways {
		d, _, m := pageWithModal()
		m.SetButtons(ModalOKCancel)
		m.SetControls(ModalClose)
		cancelled, ok := &counter{}, &counter{}
		m.OnCancel, m.OnOK = cancelled.handler(), ok.handler()
		m.Open(d)
		d.Layout()

		way(d, m)
		if cancelled.n != 1 || ok.n != 0 || d.HasModal() {
			t.Errorf("%s: cancelled=%d ok=%d open=%v", name, cancelled.n, ok.n, d.HasModal())
		}
	}
}

func TestBackIsNotHandledWhenNothingIsOpen(t *testing.T) {
	if NewDocument(100, 100).HandleBack() {
		t.Error("Back with nothing open is not handled (the app leaves)")
	}
	d, _, m := pageWithModal()
	m.Open(d)
	if !d.HandleBack() {
		t.Error("Back is handled while a modal is open")
	}
}

// countButtons splits the modal's buttons into text buttons and icon buttons.
func countButtons(m *Modal) (text, icons int) {
	walk(m, func(w Widget) bool {
		b, ok := w.(*Button)
		switch {
		case !ok:
		case b.Text == "":
			icons++
		default:
			text++
		}
		return true
	})
	return text, icons
}

func TestModalButtonsAndControlsOptions(t *testing.T) {
	cases := []struct {
		name        string
		buttons     ModalButtons
		controls    ModalControls
		text, icons int
	}{
		{"default: OK only", ModalOK, 0, 1, 0},
		{"OK/Cancel and three controls", ModalOKCancel, ModalClose | ModalMaximize | ModalMinimize, 2, 3},
		{"nothing", ModalNoButtons, 0, 0, 0},
		{"only close", ModalNoButtons, ModalClose, 0, 1},
	}
	for _, c := range cases {
		_, _, m := pageWithModal()
		m.SetButtons(c.buttons)
		m.SetControls(c.controls)
		if text, icons := countButtons(m); text != c.text || icons != c.icons {
			t.Errorf("%s: got %d text buttons and %d icons, want %d and %d", c.name, text, icons, c.text, c.icons)
		}
	}
}

func TestModalButtonTextsAreConfigurable(t *testing.T) {
	_, _, m := pageWithModal()
	m.OKText, m.CancelText = "Entendi", "Voltar"
	m.SetButtons(ModalOKCancel)
	if buttonLabeled(m, "Entendi") == nil || buttonLabeled(m, "Voltar") == nil {
		t.Error("custom button texts")
	}
}

func TestModalMaximizeMinimizeRestore(t *testing.T) {
	d, behind, m := pageWithModal()
	clicks := &counter{}
	behind.OnClick = clicks.handler()
	m.SetControls(ModalClose | ModalMaximize | ModalMinimize)
	m.Open(d)
	d.Layout()
	normal := m.Bounds

	// maximize: fullscreen, and the OK button stays at the bottom
	tap(d, iconButtons(m)[1])
	d.Layout()
	if m.State() != ModalMaximized || m.Bounds != (geom.Rect{W: d.W, H: d.H}) {
		t.Fatalf("maximize: state=%v bounds=%+v", m.State(), m.Bounds)
	}
	if ok := buttonLabeled(m, "OK"); ok.Bounds.Bottom() < d.H-150 {
		t.Errorf("the footer must sit at the bottom of a fullscreen modal: OK bottom=%v", ok.Bounds.Bottom())
	}
	tap(d, iconButtons(m)[1]) // the same button restores
	d.Layout()
	if m.State() != ModalNormal || m.Bounds != normal {
		t.Errorf("restore from maximized: %v %+v", m.State(), m.Bounds)
	}

	// minimize: only the title bar, at the bottom, and the page works again
	tap(d, iconButtons(m)[0])
	d.Layout()
	if m.State() != ModalMinimized || m.Bounds.H > 160 || m.Bounds.Bottom() < d.H-modalMargin-1 {
		t.Fatalf("minimize: state=%v bounds=%+v", m.State(), m.Bounds)
	}
	if buttonLabeled(m, "OK") != nil && buttonLabeled(m, "OK").Elem().Visible() && m.Body.Visible() {
		t.Error("the body and footer are hidden while minimized")
	}
	tap(d, behind)
	if clicks.n != 1 {
		t.Error("a minimized modal must not block the page")
	}
	if len(iconButtons(m)) != 2 {
		t.Fatalf("minimized shows restore + close, got %d buttons", len(iconButtons(m)))
	}
	tap(d, iconButtons(m)[0]) // restore
	d.Layout()
	if m.State() != ModalNormal || m.Bounds != normal || !m.Body.Visible() {
		t.Errorf("restore from minimized: %v %+v", m.State(), m.Bounds)
	}
}

func TestModalCanOpenFullscreenAndRespectsWidthAndScreen(t *testing.T) {
	d, _, m := pageWithModal()
	m.SetState(ModalMaximized)
	m.Open(d)
	d.Layout()
	if m.Bounds.W != d.W || m.Bounds.H != d.H {
		t.Errorf("opened maximized: %+v", m.Bounds)
	}
	m.Close()

	m.SetState(ModalNormal)
	m.Style.Width = Px(600)
	m.Open(d)
	d.Layout()
	if m.Bounds.W != 600 {
		t.Errorf("Style.Width must be respected: %v", m.Bounds.W)
	}
	m.Close()

	m.Style.Width = Px(5000) // wider than the screen
	m.Open(d)
	d.Layout()
	if m.Bounds.X < modalMargin-0.01 || m.Bounds.Right() > d.W-modalMargin+0.01 {
		t.Errorf("the modal must stay inside the screen margins: %+v", m.Bounds)
	}
}

func TestModalHoldsInteractiveContentAndPopupsOpenAboveIt(t *testing.T) {
	combo := NewComboBox("City", Option{Label: "A", Value: "a"}, Option{Label: "B", Value: "b"})
	field := NewInput("Name")
	m := NewModal("Form", combo, field)
	d, _, _ := pageWithModal()
	m.Open(d)
	d.Layout()

	// every interactive element of the modal is hit at its own center
	for _, w := range []Widget{combo, field, buttonLabeled(m, "OK")} {
		if got := d.TargetAt(w.Elem().Bounds.Center()); got != w {
			t.Errorf("%T at its center: target = %T", w, got)
		}
	}
	tap(d, combo)
	if !d.HasPopup() || !d.HasModal() {
		t.Fatal("the combo opens its list above the modal")
	}
	if d.HandleBack(); d.HasPopup() || !d.HasModal() {
		t.Error("Back closes the popup first and leaves the modal open")
	}
	field.Value = "typed"
	d.Keyboard = &FakeKeyboard{Reply: "from keyboard"}
	tap(d, field)
	if field.Value != "from keyboard" {
		t.Errorf("the system keyboard works inside a modal: %q", field.Value)
	}
}

func TestModalIsPaintedAboveThePageAndPartialRepaintFollowsIt(t *testing.T) {
	d, _, m := pageWithModal()
	d.Update()
	m.Open(d)
	f := d.Update()
	if f.Full && len(f.Rects) != 1 {
		t.Errorf("unexpected frame: %+v", f.Rects)
	}
	b := m.Bounds
	img := d.Render()
	if img.GrayAt(int(b.X)+2, int(b.Y+b.H/2)).Y > 60 { // the thick left border is dark
		t.Error("the modal border must be painted over the page")
	}
	m.Close()
	f = d.Update()
	if len(f.Rects) == 0 {
		t.Fatal("closing must repaint the area the modal covered")
	}
	got, want := f.Image, d.Render()
	for i := range got.Pix {
		if diff := int(got.Pix[i]) - int(want.Pix[i]); diff > 8 || diff < -8 {
			t.Fatalf("after closing, pixel %d differs from a full render: %d != %d", i, got.Pix[i], want.Pix[i])
		}
	}
}

func TestModalTouchTargetsAreComfortable(t *testing.T) {
	_, _, m := pageWithModal()
	m.SetButtons(ModalOKCancel)
	m.SetControls(ModalClose | ModalMaximize | ModalMinimize)
	d, _, _ := pageWithModal()
	m.Open(d)
	d.Layout()
	walk(m, func(w Widget) bool {
		if b, ok := w.(*Button); ok && (b.Bounds.W < MinTouch || b.Bounds.H < MinTouch) {
			t.Errorf("button %q is %vx%v, smaller than %v", b.Text, b.Bounds.W, b.Bounds.H, MinTouch)
		}
		return true
	})
}
