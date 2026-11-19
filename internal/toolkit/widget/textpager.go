package widget

import (
	"fmt"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
)

// TextPager shows a long text a page at a time, wrapped to its width, with
// previous and next buttons and a "page / pages" indicator: a message, an article,
// the notes of an event. Text on e-ink is read by turning pages, not by scrolling.
//
// Its height is Lines lines of text plus the footer; the text is re-wrapped
// whenever the width changes, so the page count follows the screen.
type TextPager struct {
	Base
	Text string
	// Lines is how many lines a page holds (20 when zero).
	Lines  int
	Page   int     // the current page, counted from 0
	OnPage Handler // called after the page changed

	wrapped      []string
	wrappedFor   string
	wrappedWidth float64
	wrappedSize  float64
}

// NewTextPager creates a text viewer.
func NewTextPager(text string) *TextPager {
	t := &TextPager{Text: text}
	t.Init(t)
	return t
}

const (
	textPagerLines  = 20
	textPagerFooter = 72.0
)

func (t *TextPager) lines() int {
	if t.Lines < 1 {
		return textPagerLines
	}
	return t.Lines
}

func (t *TextPager) textStyle() gfx.TextStyle { return t.Style.text(FontBody) }

// wrap breaks the text into lines for a width, remembering the result while
// nothing changes (painting and touching ask for it often).
func (t *TextPager) wrap(width float64) []string {
	size := t.textStyle().Size
	if t.wrapped != nil && t.wrappedFor == t.Text && t.wrappedWidth == width && t.wrappedSize == size {
		return t.wrapped
	}
	t.wrapped = wrapText(t.Text, width, size, t.textStyle().Bold)
	t.wrappedFor, t.wrappedWidth, t.wrappedSize = t.Text, width, size
	return t.wrapped
}

// Pages is how many pages the text needs at the current width (at least one).
func (t *TextPager) Pages() int {
	n := len(t.wrap(t.ContentRect().W))
	if n == 0 {
		return 1
	}
	return (n + t.lines() - 1) / t.lines()
}

// GoTo shows page n (clamped to the pages that exist).
func (t *TextPager) GoTo(n int) {
	n = int(geom.Clamp(float64(n), 0, float64(t.Pages()-1)))
	if n == t.Page {
		return
	}
	t.Page = n
	t.Invalidate()
	Fire(t.OnPage, t)
}

// Prev and Next turn the page (they stop at the ends).
func (t *TextPager) Prev() { t.GoTo(t.Page - 1) }
func (t *TextPager) Next() { t.GoTo(t.Page + 1) }

// SetText replaces the text and goes back to the first page.
func (t *TextPager) SetText(text string) {
	t.Text, t.Page = text, 0
	t.Invalidate()
}

// Measure implements Widget: the full width and a fixed height, so the footer
// does not jump around between pages.
func (t *TextPager) Measure(avail float64) geom.Size {
	return geom.Size{W: avail, H: lineHeight(t.textStyle().Size)*float64(t.lines()) + textPagerFooter}
}

func (t *TextPager) footerRect() geom.Rect {
	r := t.ContentRect()
	return geom.R(r.X, r.Bottom()-textPagerFooter, r.W, textPagerFooter)
}

// pagerRects are the previous and next buttons.
func (t *TextPager) pagerRects() (prev, next geom.Rect) {
	f := t.footerRect()
	y := f.Y + (f.H-pagerBtn)/2
	next = geom.R(f.Right()-pagerBtn-16, y, pagerBtn, pagerBtn)
	prev = geom.R(next.X-pagerBtn-12, y, pagerBtn, pagerBtn)
	return prev, next
}

// Paint implements Widget.
func (t *TextPager) Paint(c *gfx.Canvas) {
	ts := t.textStyle()
	if t.Disabled {
		ts.Color = Disabled
	}
	ts.Ellipsis = false
	r := t.ContentRect()
	lh := lineHeight(ts.Size)
	all := t.wrap(r.W)
	pages := t.Pages()
	page := int(geom.Clamp(float64(t.Page), 0, float64(pages-1)))
	from := page * t.lines()
	for i := 0; i < t.lines() && from+i < len(all); i++ {
		c.Text(all[from+i], geom.Rect{X: r.X, Y: r.Y + float64(i)*lh, W: r.W, H: lh}, ts)
	}

	f := t.footerRect()
	c.Line(geom.Point{X: f.X, Y: f.Y}, geom.Point{X: f.Right(), Y: f.Y}, 2, Ink)
	c.Text(fmt.Sprintf("%d / %d", page+1, pages), geom.R(f.X+20, f.Y, f.W/2, f.H), gfx.TextStyle{Size: FontSmall, Color: Muted})
	prev, next := t.pagerRects()
	paintPagerButton(c, prev, gfx.IconChevronLeft, page > 0)
	paintPagerButton(c, next, gfx.IconChevronRight, page < pages-1)
}

// HandlePointer implements PointerHandler: the footer buttons turn the page.
func (t *TextPager) HandlePointer(e *PointerEvent) {
	if e.Type != PointerClick {
		return
	}
	p := geom.Point{X: e.X, Y: e.Y}
	prev, next := t.pagerRects()
	switch {
	case prev.Contains(p):
		t.Prev()
	case next.Contains(p):
		t.Next()
	}
}

// paintPagerButton draws a footer chevron: boxed and dark when it can be used, pale when it cannot.
func paintPagerButton(c *gfx.Canvas, r geom.Rect, icon gfx.IconKind, enabled bool) {
	col, line := Ink, Ink
	if !enabled {
		col, line = Disabled, Mid
	}
	c.StrokeRoundRect(r, 12, BorderThin, line)
	c.Icon(icon, r.Inset(geom.All(14)), iconStroke, col)
}
