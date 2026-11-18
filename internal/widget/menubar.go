package widget

import (
	"fmt"
	"strings"

	"go-pocketbook-base/internal/geom"
	"go-pocketbook-base/internal/gfx"
)

// Menu is one entry of a MenuBar: it either runs OnSelect or opens Items.
type Menu struct {
	Label    string
	Items    []MenuItem
	OnSelect func()
	Selected bool // drawn as the current one (e.g. the active page)
}

// MenuBar is the horizontal bar of menus at the top of a screen.
type MenuBar struct {
	Base
	Menus []*Menu
}

// NewMenuBar creates a menu bar.
func NewMenuBar(menus ...*Menu) *MenuBar {
	m := &MenuBar{Menus: menus}
	m.Init(m)
	m.Style.Background, m.Style.MinHeight = Paper, 104
	m.Style.Padding = geom.XY(10, 16)
	return m
}

// StateSignature implements signer: the menus are behind pointers.
func (m *MenuBar) StateSignature() string {
	var sb strings.Builder
	for _, mn := range m.Menus {
		fmt.Fprintf(&sb, "%s|%t|%d;", mn.Label, mn.Selected, len(mn.Items))
	}
	return sb.String()
}

const menuGap = 8.0

func (m *MenuBar) menuWidth(mn *Menu) float64 {
	w := gfx.TextWidth(mn.Label, FontBody, true) + 2*menuPadX
	if len(mn.Items) > 0 {
		w += 26
	}
	return w
}

// menuRects returns the rectangle of each menu, left to right.
func (m *MenuBar) menuRects() []geom.Rect {
	r := m.ContentRect()
	out := make([]geom.Rect, len(m.Menus))
	x := r.X
	for i, mn := range m.Menus {
		w := m.menuWidth(mn)
		out[i] = geom.R(x, r.Y, w, r.H)
		x += w + menuGap
	}
	return out
}

// Measure implements Widget.
func (m *MenuBar) Measure(avail float64) geom.Size {
	w := menuGap * float64(len(m.Menus))
	for _, mn := range m.Menus {
		w += m.menuWidth(mn)
	}
	return geom.Size{W: geom.Min(w, avail), H: ControlHeight}
}

// Paint implements Widget.
func (m *MenuBar) Paint(c *gfx.Canvas) {
	for i, rc := range m.menuRects() {
		mn := m.Menus[i]
		color := Ink
		if mn.Selected {
			c.FillRoundRect(rc, Radius, Ink)
			color = Paper
		}
		label := geom.R(rc.X, rc.Y, rc.W, rc.H)
		if len(mn.Items) > 0 {
			label.W -= 26
			c.Icon(gfx.IconChevronDown, geom.R(rc.Right()-menuPadX-14, rc.Y+(rc.H-26)/2, 26, 26), iconStroke-1, color)
		}
		c.Text(mn.Label, label, gfx.TextStyle{Size: FontBody, Bold: true, Color: color, Align: gfx.AlignCenter})
	}
	b := m.Bounds
	c.Line(geom.Point{X: b.X, Y: b.Bottom() - BorderThin/2}, geom.Point{X: b.Right(), Y: b.Bottom() - BorderThin/2}, BorderThin, Ink)
}

// HandlePointer implements PointerHandler.
func (m *MenuBar) HandlePointer(e *PointerEvent) {
	if e.Type != PointerClick {
		return
	}
	p := geom.Point{X: e.X, Y: e.Y}
	for i, rc := range m.menuRects() {
		if !rc.Contains(p) {
			continue
		}
		mn := m.Menus[i]
		if len(mn.Items) == 0 {
			if mn.OnSelect != nil {
				mn.OnSelect()
			}
			return
		}
		openMenu(m, mn.Items, -1, rc, rc.W, func(j int) {
			if f := mn.Items[j].OnSelect; f != nil {
				f()
			}
		})
		return
	}
}

// MenuRect is the touch rectangle of menu i (design pixels).
func (m *MenuBar) MenuRect(i int) geom.Rect { return m.menuRects()[i] }
