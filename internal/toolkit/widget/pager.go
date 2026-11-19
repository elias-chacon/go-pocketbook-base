package widget

import (
	"fmt"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
)

// Pager shows a long list of widgets a page at a time, with previous and next
// buttons and a "page / pages" indicator. It is the e-ink answer to a scrolling
// list: a page is one repaint, and nothing moves while you read.
//
// The items are any widgets (a row with a sender, a subject and a date; a task
// with a check box...): the Pager only decides which ones are on the screen. It
// is built from existing widgets, so add Pager.Root to the document.
type Pager struct {
	// Root is the element to add to the document: the page of items above the
	// footer.
	Root *Panel
	// OnPage is called after the page changed (by a button or by GoTo).
	OnPage Handler

	pageSize  int
	items     []Widget
	page      int
	body      *Panel
	indicator *Label
	prev      *Button
	next      *Button
}

// NewPager creates a pager that shows pageSize items at a time (at least 1).
func NewPager(pageSize int, items ...Widget) *Pager {
	if pageSize < 1 {
		pageSize = 1
	}
	p := &Pager{pageSize: pageSize}
	p.body = NewColumn(0)
	p.indicator = NewLabel("")
	p.indicator.Style.FontSize, p.indicator.Style.Color, p.indicator.Style.FlexGrow = FontSmall, Muted, 1
	p.prev, p.next = NewIconButton(gfx.IconChevronLeft), NewIconButton(gfx.IconChevronRight)
	p.prev.OnClick = func(*Event) { p.Prev() }
	p.next.OnClick = func(*Event) { p.Next() }
	footer := NewRow(12, p.indicator, p.prev, p.next)
	footer.Style.Padding = geom.XY(8, 0)
	p.Root = NewColumn(8, p.body, footer)
	p.SetItems(items)
	return p
}

// SetItems replaces the items. The page stays where it is when it still exists.
func (p *Pager) SetItems(items []Widget) {
	p.items = items
	p.show()
}

// Len is the number of items.
func (p *Pager) Len() int { return len(p.items) }

// PageSize is how many items a page holds.
func (p *Pager) PageSize() int { return p.pageSize }

// Pages is how many pages the items need (at least one: an empty list is a page).
func (p *Pager) Pages() int {
	if len(p.items) == 0 {
		return 1
	}
	return (len(p.items) + p.pageSize - 1) / p.pageSize
}

// Page is the current page, counted from 0.
func (p *Pager) Page() int { return p.page }

// GoTo shows page n (clamped to the pages that exist).
func (p *Pager) GoTo(n int) {
	n = int(geom.Clamp(float64(n), 0, float64(p.Pages()-1)))
	if n == p.page {
		return
	}
	p.page = n
	p.show()
	Fire(p.OnPage, p.Root)
}

// Prev and Next turn the page (they stop at the ends).
func (p *Pager) Prev() { p.GoTo(p.page - 1) }
func (p *Pager) Next() { p.GoTo(p.page + 1) }

// show puts the items of the current page in the body and updates the footer.
func (p *Pager) show() {
	if p.page >= p.Pages() {
		p.page = p.Pages() - 1
	}
	p.body.ClearChildren()
	from := p.page * p.pageSize
	to := from + p.pageSize
	if to > len(p.items) {
		to = len(p.items)
	}
	for i := from; i < to; i++ {
		p.body.AppendChild(p.items[i])
	}
	p.indicator.Text = fmt.Sprintf("%d / %d", p.page+1, p.Pages())
	p.prev.Disabled = p.page == 0
	p.next.Disabled = p.page >= p.Pages()-1
	p.Root.Invalidate()
}
