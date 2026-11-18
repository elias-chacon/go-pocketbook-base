package widget

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
)

// TableColumn describes a table column.
type TableColumn struct {
	Title    string
	Flex     float64 // share of the width (default 1)
	Align    gfx.TextAlign
	Sortable bool
}

// Table is a paged data grid: tap a header to sort, a row to select it, and
// use the arrows in the footer to change page.
type Table struct {
	Base
	Columns  []TableColumn
	Rows     [][]string
	PageSize int
	Page     int
	Selected int // index in Rows, -1 for none
	SortCol  int // -1 for natural order
	SortDesc bool
	// Summary formats the footer text; the default is "1–5 of 12".
	Summary  func(from, to, total int) string
	OnSelect Handler
}

const (
	tableHeaderH = 76.0
	tableRowH    = 70.0
	tableFooterH = 84.0
	pagerBtn     = 60.0
)

// NewTable creates a table showing pageSize rows per page.
func NewTable(pageSize int, columns ...TableColumn) *Table {
	t := &Table{Columns: columns, PageSize: pageSize, Selected: -1, SortCol: -1}
	t.Init(t)
	t.Style.BorderColor, t.Style.BorderWidth, t.Style.BorderRadius = Ink, BorderThin, Radius
	t.Style.Background = Paper
	return t
}

// PageCount is the number of pages (at least 1).
func (t *Table) PageCount() int {
	n := (len(t.Rows) + t.PageSize - 1) / t.PageSize
	if n < 1 {
		n = 1
	}
	return n
}

// order returns the row indexes in display order.
func (t *Table) order() []int {
	idx := make([]int, len(t.Rows))
	for i := range idx {
		idx[i] = i
	}
	if t.SortCol < 0 || t.SortCol >= len(t.Columns) {
		return idx
	}
	col := t.SortCol
	sort.SliceStable(idx, func(a, b int) bool {
		x, y := t.Rows[idx[a]][col], t.Rows[idx[b]][col]
		if t.SortDesc {
			x, y = y, x
		}
		return cellLess(x, y)
	})
	return idx
}

// cellLess compares numerically when both cells are numbers, else as text.
func cellLess(a, b string) bool {
	x, ea := strconv.ParseFloat(a, 64)
	y, eb := strconv.ParseFloat(b, 64)
	if ea == nil && eb == nil {
		return x < y
	}
	return strings.ToLower(a) < strings.ToLower(b)
}

// Measure implements Widget.
func (t *Table) Measure(avail float64) geom.Size {
	return geom.Size{W: avail, H: tableHeaderH + tableRowH*float64(t.PageSize) + tableFooterH}
}

// colRects splits r horizontally by the columns' Flex.
func (t *Table) colRects(r geom.Rect) []geom.Rect {
	total := 0.0
	for _, c := range t.Columns {
		total += geom.Max(c.Flex, 1)
	}
	out := make([]geom.Rect, len(t.Columns))
	x := r.X
	for i, c := range t.Columns {
		w := r.W * geom.Max(c.Flex, 1) / total
		out[i] = geom.R(x, r.Y, w, r.H)
		x += w
	}
	return out
}

func (t *Table) headerRect() geom.Rect {
	r := t.ContentRect()
	return geom.R(r.X, r.Y, r.W, tableHeaderH)
}

func (t *Table) rowRect(k int) geom.Rect {
	r := t.ContentRect()
	return geom.R(r.X, r.Y+tableHeaderH+float64(k)*tableRowH, r.W, tableRowH)
}

func (t *Table) footerRect() geom.Rect {
	r := t.ContentRect()
	return geom.R(r.X, r.Y+tableHeaderH+float64(t.PageSize)*tableRowH, r.W, tableFooterH)
}

// pagerRects are the previous and next buttons.
func (t *Table) pagerRects() (prev, next geom.Rect) {
	f := t.footerRect()
	y := f.Y + (f.H-pagerBtn)/2
	next = geom.R(f.Right()-pagerBtn-16, y, pagerBtn, pagerBtn)
	prev = geom.R(next.X-pagerBtn-12, y, pagerBtn, pagerBtn)
	return prev, next
}

func (t *Table) summary(from, to, total int) string {
	if t.Summary != nil {
		return t.Summary(from, to, total)
	}
	return fmt.Sprintf("%d–%d of %d", from, to, total)
}

// Paint implements Widget.
func (t *Table) Paint(c *gfx.Canvas) {
	inner := t.Bounds.Inset(geom.All(BorderThin))
	c.Clip(inner, t.Style.BorderRadius-BorderThin)
	t.paintHeader(c)
	t.paintRows(c)
	c.ResetClip()
	t.paintFooter(c)
}

func (t *Table) paintHeader(c *gfx.Canvas) {
	h := t.headerRect()
	c.FillRoundRect(h, 0, Soft)
	c.Line(geom.Point{X: h.X, Y: h.Bottom()}, geom.Point{X: h.Right(), Y: h.Bottom()}, 2, Ink)
	for i, cr := range t.colRects(h) {
		col := t.Columns[i]
		text := geom.R(cr.X+18, cr.Y, cr.W-36, cr.H)
		if i == t.SortCol {
			text.W -= 34
			kind := gfx.IconChevronUp
			if t.SortDesc {
				kind = gfx.IconChevronDown
			}
			c.Icon(kind, geom.R(cr.Right()-18-28, cr.Y+(cr.H-28)/2, 28, 28), iconStroke-1, Ink)
		}
		c.Text(col.Title, text, gfx.TextStyle{Size: FontSmall, Bold: true, Color: Ink, Align: col.Align, Ellipsis: true})
	}
}

func (t *Table) paintRows(c *gfx.Canvas) {
	order := t.order()
	for k := 0; k < t.PageSize; k++ {
		row := t.rowRect(k)
		pos := t.Page*t.PageSize + k
		if pos >= len(order) {
			break
		}
		idx := order[pos]
		color := Ink
		if idx == t.Selected {
			c.FillRoundRect(row, 0, Ink)
			color = Paper
		} else if k < t.PageSize-1 {
			c.Line(geom.Point{X: row.X, Y: row.Bottom()}, geom.Point{X: row.Right(), Y: row.Bottom()}, 1.5, Mid)
		}
		for i, cr := range t.colRects(row) {
			c.Text(t.Rows[idx][i], geom.R(cr.X+18, cr.Y, cr.W-36, cr.H), gfx.TextStyle{Size: FontSmall + 2, Color: color, Align: t.Columns[i].Align, Ellipsis: true})
		}
	}
}

func (t *Table) paintFooter(c *gfx.Canvas) {
	f := t.footerRect()
	c.Line(geom.Point{X: f.X, Y: f.Y}, geom.Point{X: f.Right(), Y: f.Y}, 2, Ink)
	from, to := t.pageRange()
	c.Text(t.summary(from, to, len(t.Rows)), geom.R(f.X+20, f.Y, f.W/2, f.H), gfx.TextStyle{Size: FontSmall, Color: Muted})
	prev, next := t.pagerRects()
	t.paintPager(c, prev, gfx.IconChevronLeft, t.Page > 0)
	t.paintPager(c, next, gfx.IconChevronRight, t.Page < t.PageCount()-1)
}

func (t *Table) paintPager(c *gfx.Canvas, r geom.Rect, icon gfx.IconKind, enabled bool) {
	col, line := Ink, Ink
	if !enabled {
		col, line = Disabled, Mid
	}
	c.StrokeRoundRect(r, 12, BorderThin, line)
	c.Icon(icon, r.Inset(geom.All(14)), iconStroke, col)
}

// HandlePointer implements PointerHandler.
func (t *Table) HandlePointer(e *PointerEvent) {
	if e.Type != PointerClick {
		return
	}
	p := geom.Point{X: e.X, Y: e.Y}
	switch {
	case t.headerRect().Contains(p):
		t.clickHeader(p)
	case t.footerRect().Contains(p):
		t.clickFooter(p)
	default:
		t.clickRow(p)
	}
}

func (t *Table) clickHeader(p geom.Point) {
	for i, cr := range t.colRects(t.headerRect()) {
		if cr.Contains(p) && t.Columns[i].Sortable {
			t.SortDesc = t.SortCol == i && !t.SortDesc
			t.SortCol = i
			t.Page = 0
		}
	}
}

func (t *Table) clickFooter(p geom.Point) {
	prev, next := t.pagerRects()
	switch {
	case prev.Contains(p) && t.Page > 0:
		t.Page--
	case next.Contains(p) && t.Page < t.PageCount()-1:
		t.Page++
	}
}

func (t *Table) clickRow(p geom.Point) {
	order := t.order()
	for k := 0; k < t.PageSize; k++ {
		pos := t.Page*t.PageSize + k
		if t.rowRect(k).Contains(p) && pos < len(order) {
			t.Selected = order[pos]
			Fire(t.OnSelect, t)
			return
		}
	}
}

// pageRange is the 1-based first and last row number shown on the current page.
func (t *Table) pageRange() (from, to int) {
	if len(t.Rows) == 0 {
		return 0, 0
	}
	from = t.Page*t.PageSize + 1
	to = from + t.PageSize - 1
	if to > len(t.Rows) {
		to = len(t.Rows)
	}
	return from, to
}
