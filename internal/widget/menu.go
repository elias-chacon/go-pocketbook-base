package widget

import (
	"go-pocketbook-base/internal/geom"
	"go-pocketbook-base/internal/gfx"
)

// MenuItem is one entry of a menu, drop-down or combo-box list.
type MenuItem struct {
	Label     string
	Value     string
	Icon      gfx.IconKind
	Disabled  bool
	Separator bool
	OnSelect  func()
	Items     []MenuItem // sub-menu (used by MenuBar)
}

const (
	menuRowH = 76.0
	menuSepH = 26.0
	menuPadX = 28.0
)

// menuList is the popup that lists menu items; it is what ComboBox, Dropdown
// and MenuBar open.
type menuList struct {
	Base
	items    []MenuItem
	selected int // highlighted row, -1 for none
	onPick   func(i int)
}

func newMenuList(items []MenuItem, selected int, onPick func(int)) *menuList {
	m := &menuList{items: items, selected: selected, onPick: onPick}
	m.init(m)
	s := &m.Style
	s.Background, s.BorderColor, s.BorderWidth, s.BorderRadius = Paper, Ink, BorderThin, Radius
	s.Padding = geom.All(6)
	return m
}

func rowHeight(it MenuItem) float64 {
	if it.Separator {
		return menuSepH
	}
	return menuRowH
}

// Measure implements Widget: as wide as the longest label.
func (m *menuList) Measure(float64) geom.Size {
	w, h := 0.0, 0.0
	for _, it := range m.items {
		h += rowHeight(it)
		iw := gfx.TextWidth(it.Label, FontBody, false) + 2*menuPadX
		if it.Icon != gfx.IconNone {
			iw += iconSize + iconGap
		}
		w = geom.Max(w, iw)
	}
	return geom.Size{W: w, H: h}
}

// rowRect returns the rectangle of item i inside the popup.
func (m *menuList) rowRect(i int) geom.Rect {
	r := m.ContentRect()
	y := r.Y
	for _, it := range m.items[:i] {
		y += rowHeight(it)
	}
	return geom.R(r.X, y, r.W, rowHeight(m.items[i]))
}

// Paint implements Widget.
func (m *menuList) Paint(c *gfx.Canvas) {
	for i, it := range m.items {
		row := m.rowRect(i)
		if it.Separator {
			c.Line(geom.Point{X: row.X + 12, Y: row.Y + row.H/2}, geom.Point{X: row.Right() - 12, Y: row.Y + row.H/2}, 2, Mid)
			continue
		}
		color := Ink
		if it.Disabled {
			color = Disabled
		}
		if i == m.selected {
			c.FillRoundRect(row, 10, Ink)
			color = Paper
		}
		x := row.X + menuPadX
		if it.Icon != gfx.IconNone {
			c.Icon(it.Icon, geom.R(x, row.Y+(row.H-iconSize)/2, iconSize, iconSize), iconStroke, color)
			x += iconSize + iconGap
		}
		c.Text(it.Label, geom.R(x, row.Y, row.Right()-x-menuPadX/2, row.H), gfx.TextStyle{Size: FontBody, Color: color, Ellipsis: true})
	}
}

// HandlePointer implements PointerHandler.
func (m *menuList) HandlePointer(e *PointerEvent) {
	if e.Type != PointerClick {
		return
	}
	for i, it := range m.items {
		if !m.rowRect(i).Contains(geom.Point{X: e.X, Y: e.Y}) {
			continue
		}
		if it.Separator || it.Disabled {
			return
		}
		m.doc.ClosePopups()
		m.onPick(i)
		return
	}
}

// openMenu shows items in a popup under anchor (at least minW wide) and calls
// onPick with the chosen index.
func openMenu(owner Widget, items []MenuItem, selected int, anchor geom.Rect, minW float64, onPick func(int)) {
	d := owner.Elem().doc
	if d == nil || len(items) == 0 {
		return
	}
	m := newMenuList(items, selected, onPick)
	size := measure(m, unbounded)
	size.W = geom.Max(size.W, minW)
	d.OpenPopup(owner, m, anchor, size)
}
