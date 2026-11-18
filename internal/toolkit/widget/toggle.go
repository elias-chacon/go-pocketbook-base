package widget

import (
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
)

// ToggleKind selects how a Toggle looks and behaves.
type ToggleKind int

// Toggle kinds.
const (
	KindCheckbox ToggleKind = iota // independent on/off square
	KindRadio                      // one of a group (same Name)
	KindSwitch                     // on/off pill
)

// Toggle is a labelled on/off control: checkbox, radio button or switch. The
// whole row is the touch target, not just the indicator.
type Toggle struct {
	Base
	Kind     ToggleKind
	Label    string
	Checked  bool
	Name     string // radio group
	Value    string
	OnChange Handler
}

func newToggle(kind ToggleKind, label string, checked bool) *Toggle {
	t := &Toggle{Kind: kind, Label: label, Checked: checked}
	t.Init(t)
	t.Style.MinHeight = 72
	return t
}

// NewCheckbox creates a checkbox.
func NewCheckbox(label string, checked bool) *Toggle { return newToggle(KindCheckbox, label, checked) }

// NewRadio creates a radio button; radios with the same name are exclusive.
func NewRadio(label, name, value string) *Toggle {
	t := newToggle(KindRadio, label, false)
	t.Name, t.Value = name, value
	return t
}

// NewSwitch creates an on/off switch.
func NewSwitch(label string, on bool) *Toggle { return newToggle(KindSwitch, label, on) }

// indicator is the size of the check box, radio ring or switch pill.
func (t *Toggle) indicator() geom.Size {
	if t.Kind == KindSwitch {
		return geom.Size{W: 92, H: 52}
	}
	return geom.Size{W: 46, H: 46}
}

// Measure implements Widget.
func (t *Toggle) Measure(float64) geom.Size {
	ind := t.indicator()
	w := ind.W
	if t.Label != "" {
		w += Space + gfx.TextWidth(t.Label, FontBody, false)
	}
	return geom.Size{W: w, H: ind.H}
}

// Paint implements Widget.
func (t *Toggle) Paint(c *gfx.Canvas) {
	r := t.ContentRect()
	ind := t.indicator()
	box := geom.R(r.X, r.Y+(r.H-ind.H)/2, ind.W, ind.H)
	line, fill := t.fg(Ink), t.fg(Ink)
	switch t.Kind {
	case KindCheckbox:
		paintCheckbox(c, box, t.Checked, line, fill)
	case KindRadio:
		paintRadio(c, box, t.Checked, line, fill)
	default:
		paintSwitch(c, box, t.Checked, line, fill)
	}
	ts := t.Style.text(FontBody)
	ts.Color = t.fg(ts.Color)
	c.Text(t.Label, geom.R(box.Right()+Space, r.Y, r.W-ind.W-Space, r.H), ts)
}

func paintCheckbox(c *gfx.Canvas, box geom.Rect, on bool, line, fill gfx.Color) {
	if on {
		c.FillRoundRect(box, 10, fill)
		c.Icon(gfx.IconCheck, box.Inset(geom.All(6)), 5, Paper)
		return
	}
	c.StrokeRoundRect(box, 10, BorderThin+1, line)
}

func paintRadio(c *gfx.Canvas, box geom.Rect, on bool, line, fill gfx.Color) {
	ctr := box.Center()
	c.StrokeCircle(ctr, box.W/2, BorderThin+1, line)
	if on {
		c.FillCircle(ctr, box.W/2-11, fill)
	}
}

func paintSwitch(c *gfx.Canvas, box geom.Rect, on bool, line, fill gfx.Color) {
	r := box.H / 2
	knobR := r - 10
	if on {
		c.FillRoundRect(box, r, fill)
		c.FillCircle(geom.Point{X: box.Right() - r, Y: box.Y + r}, knobR, Paper)
		return
	}
	c.StrokeRoundRect(box, r, BorderThin+1, line)
	c.FillCircle(geom.Point{X: box.X + r, Y: box.Y + r}, knobR, line)
}

// HandlePointer implements PointerHandler.
func (t *Toggle) HandlePointer(e *PointerEvent) {
	if e.Type != PointerClick {
		return
	}
	switch t.Kind {
	case KindRadio:
		if t.Checked {
			return // a radio cannot be unchecked by tapping it again
		}
		t.uncheckGroup()
		t.Checked = true
	default:
		t.Checked = !t.Checked
	}
	Fire(t.OnChange, t)
}

// uncheckGroup clears the other radios that share this one's Name.
func (t *Toggle) uncheckGroup() {
	if t.doc == nil || t.Name == "" {
		return
	}
	t.doc.Walk(func(w Widget) bool {
		if o, ok := w.(*Toggle); ok && o != t && o.Kind == KindRadio && o.Name == t.Name {
			o.Checked = false
		}
		return true
	})
}
