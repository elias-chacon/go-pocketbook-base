package widget

import (
	"testing"

	"go-pocketbook-base/internal/kernel/geom"
)

func TestClickBubblesAndCanBeStopped(t *testing.T) {
	var order []string
	btn := NewButton("x")
	box := NewPanel(btn)
	root := NewPanel(box)
	btn.OnClick = func(e *Event) { order = append(order, "button") }
	box.OnClick = func(e *Event) { order = append(order, "box") }
	root.OnClick = func(e *Event) { order = append(order, "root") }
	d := NewDocument(1404, 1872)
	d.SetRoot(root)
	d.Layout()

	tap(d, btn)
	if len(order) != 3 || order[0] != "button" || order[2] != "root" {
		t.Errorf("bubbling order = %v", order)
	}

	order = nil
	box.OnClick = func(e *Event) { order = append(order, "box"); e.StopPropagation() }
	tap(d, btn)
	if len(order) != 2 {
		t.Errorf("after StopPropagation = %v", order)
	}
}

func TestEventCarriesTargetAndPosition(t *testing.T) {
	btn := NewButton("x")
	var got *Event
	btn.OnClick = func(e *Event) { got = e }
	d := testDoc(btn)
	tap(d, btn)
	c := btn.Bounds.Center()
	if got == nil || got.Target != Widget(btn) || got.X != c.X || got.Y != c.Y {
		t.Errorf("event = %+v, want target button at %v", got, c)
	}
}

func TestDisabledAndHiddenElementsDoNotReceiveTouch(t *testing.T) {
	disabled, hidden := NewButton("d"), NewButton("h")
	disabled.Disabled, hidden.Hidden = true, true
	clicks := &counter{}
	disabled.OnClick, hidden.OnClick = clicks.handler(), clicks.handler()
	col := NewColumn(10, disabled, hidden)
	d := NewDocument(1404, 1872)
	d.SetRoot(col)
	d.Layout()
	tap(d, disabled)
	tapAt(d, geom.Point{X: 5, Y: 5})
	if clicks.n != 0 {
		t.Errorf("clicks = %d", clicks.n)
	}
}

func TestReleaseOutsideTargetDoesNotClick(t *testing.T) {
	btn := NewButton("x")
	clicks := &counter{}
	btn.OnClick = clicks.handler()
	d := testDoc(btn)
	c := btn.Bounds.Center()
	d.HandlePointer(PointerDown, c.X, c.Y)
	d.HandlePointer(PointerUp, c.X+2000, c.Y) // finger slid away
	if clicks.n != 0 {
		t.Error("a released-outside touch is not a click")
	}
}

func TestTouchIsScaledToTheScreenResolution(t *testing.T) {
	// A half-size screen: design pixels are drawn at half scale.
	btn := NewButton("x")
	clicks := &counter{}
	btn.OnClick = clicks.handler()
	d := NewDocument(702, 936)
	root := NewColumn(0, btn)
	root.Style.AlignItems = AlignStart
	d.SetRoot(root)
	d.Layout()
	if d.Scale != 0.5 || d.H != 1872 {
		t.Fatalf("Scale=%v H=%v", d.Scale, d.H)
	}
	c := btn.Bounds.Center()
	d.HandlePointer(PointerDown, c.X*0.5, c.Y*0.5)
	d.HandlePointer(PointerUp, c.X*0.5, c.Y*0.5)
	if clicks.n != 1 {
		t.Errorf("clicks = %d: device coordinates must be converted to design pixels", clicks.n)
	}
}

func TestQueriesAndTreeEditing(t *testing.T) {
	a, b := NewButton("a"), NewButton("b")
	a.ID, b.ID, b.ClassName = "first", "second", "special"
	root := NewColumn(0, a, b)
	d := NewDocument(1404, 1872)
	d.SetRoot(root)

	if d.GetElementByID("second") != Widget(b) || d.GetElementByID("nope") != nil {
		t.Error("GetElementByID")
	}
	if got := d.QuerySelectorAll("special"); len(got) != 1 || got[0] != Widget(b) {
		t.Errorf("QuerySelectorAll = %v", got)
	}
	if b.Document() != d {
		t.Error("children must know their document")
	}
	root.RemoveChild(a)
	if a.Document() != nil || a.Parent() != nil || len(root.Children()) != 1 {
		t.Error("RemoveChild must detach")
	}
	root.AppendChild(a)
	root.ClearChildren()
	if len(root.Children()) != 0 || b.Document() != nil {
		t.Error("ClearChildren must detach everything")
	}
}

func TestRenderProducesAnImageOfTheScreenSize(t *testing.T) {
	d := testDoc(NewButton("Hello"))
	img := d.Render()
	if b := img.Bounds(); b.Dx() != 1404 || b.Dy() != 1872 {
		t.Errorf("image = %v", b)
	}
	black := 0
	for _, p := range img.Pix {
		if p < 128 {
			black++
		}
	}
	if black == 0 {
		t.Error("nothing was drawn")
	}
}

func TestSetInvertShowsTheNegativeAndRepaintsEverything(t *testing.T) {
	doc := NewDocument(351, 468)
	doc.SetRoot(NewColumn(0, NewLabel("x")))
	invalidated := 0
	doc.OnInvalidate = func() { invalidated++ }
	doc.Update()
	doc.SetInvert(true)
	if invalidated != 1 {
		t.Errorf("turning it on must ask for a repaint, got %d", invalidated)
	}
	f := doc.Update()
	if !f.Full || f.Image.Pix[0] != 0 {
		t.Errorf("full=%v corner=%d: the negative of the paper is black", f.Full, f.Image.Pix[0])
	}
	doc.SetInvert(true)
	if invalidated != 1 {
		t.Error("setting the same value again changes nothing")
	}
	doc.SetInvert(false)
	if f := doc.Update(); f.Image.Pix[0] != 255 {
		t.Errorf("back to paper: %d", f.Image.Pix[0])
	}
}
