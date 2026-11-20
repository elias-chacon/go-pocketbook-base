package widget

import (
	"fmt"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
)

// ParagraphPager shows a list of numbered paragraphs (the verses of a chapter, the
// clauses of a contract, the steps of a recipe) a page at a time, wrapped to its
// width, with previous and next buttons and a "page / pages" indicator. Unlike
// TextPager it knows where each paragraph is, so the app can tell which ones are on
// the screen (Visible) and open the page of a given one (GoToParagraph).
//
// Only the strings the app hands over are held; the lines of the current width are
// kept while nothing changes.
type ParagraphPager struct {
	Base
	// Paragraphs are the texts; Numbers (optional, same length) is the small label
	// drawn in the margin of the first line of each one ("16" for verse 16).
	Paragraphs []string
	Numbers    []string
	// Lines is how many lines a page holds (20 when zero).
	Lines  int
	Page   int     // the current page, counted from 0
	OnPage Handler // called after the page changed

	laid       []pagerLine
	laidWidth  float64
	laidSize   float64
	laidFamily gfx.Family
	laidFor    int // len(Paragraphs) the lines were made for; SetParagraphs resets the lines
}

// pagerLine is one wrapped line and the paragraph it belongs to.
type pagerLine struct {
	para  int
	text  string
	first bool
}

// NewParagraphPager creates a pager of paragraphs.
func NewParagraphPager(paragraphs, numbers []string) *ParagraphPager {
	p := &ParagraphPager{Paragraphs: paragraphs, Numbers: numbers}
	p.Init(p)
	return p
}

func (p *ParagraphPager) lines() int {
	if p.Lines < 1 {
		return textPagerLines
	}
	return p.Lines
}

func (p *ParagraphPager) textStyle() gfx.TextStyle { return p.Style.text(FontBody) }

// margin is the width kept free on the left for the numbers (none without numbers).
func (p *ParagraphPager) margin() float64 {
	if len(p.Numbers) == 0 {
		return 0
	}
	widest := 0.0
	for _, n := range p.Numbers {
		widest = geom.Max(widest, gfx.TextWidth(n, p.numberSize(), true))
	}
	return widest + 16
}

func (p *ParagraphPager) numberSize() float64 { return geom.Max(p.textStyle().Size*0.7, 14) }

// wrap lays the paragraphs out for a width, keeping the result while the width, the
// text size and the typeface stay the same.
func (p *ParagraphPager) wrap(width float64) []pagerLine {
	size, fam := p.textStyle().Size, gfx.CurrentFamily()
	if p.laid != nil && p.laidWidth == width && p.laidSize == size && p.laidFamily == fam && p.laidFor == len(p.Paragraphs) {
		return p.laid
	}
	out := make([]pagerLine, 0, len(p.Paragraphs)*3)
	for i, text := range p.Paragraphs {
		for j, line := range wrapText(text, width-p.margin(), size, p.textStyle().Bold) {
			out = append(out, pagerLine{para: i, text: line, first: j == 0})
		}
	}
	p.laid, p.laidWidth, p.laidSize, p.laidFamily, p.laidFor = out, width, size, fam, len(p.Paragraphs)
	return out
}

// SetParagraphs replaces the content and goes back to the first page.
func (p *ParagraphPager) SetParagraphs(paragraphs, numbers []string) {
	p.Paragraphs, p.Numbers, p.Page, p.laid = paragraphs, numbers, 0, nil
	p.Invalidate()
}

// Pages is how many pages the paragraphs need at the current width (at least one).
func (p *ParagraphPager) Pages() int {
	n := len(p.wrap(p.ContentRect().W))
	if n == 0 {
		return 1
	}
	return (n + p.lines() - 1) / p.lines()
}

// GoTo shows page n (clamped to the pages that exist).
func (p *ParagraphPager) GoTo(n int) {
	n = int(geom.Clamp(float64(n), 0, float64(p.Pages()-1)))
	if n == p.Page {
		return
	}
	p.Page = n
	p.Invalidate()
	Fire(p.OnPage, p)
}

// Prev and Next turn the page (they stop at the ends).
func (p *ParagraphPager) Prev() { p.GoTo(p.Page - 1) }
func (p *ParagraphPager) Next() { p.GoTo(p.Page + 1) }

// GoToParagraph shows the page where paragraph i starts.
func (p *ParagraphPager) GoToParagraph(i int) {
	for n, l := range p.wrap(p.ContentRect().W) {
		if l.para >= i {
			p.GoTo(n / p.lines())
			return
		}
	}
}

// Visible is the first and last paragraph with a line on the current page, and
// false when there is none (no text).
func (p *ParagraphPager) Visible() (first, last int, ok bool) {
	all := p.wrap(p.ContentRect().W)
	from := int(geom.Clamp(float64(p.Page), 0, float64(p.Pages()-1))) * p.lines()
	if from >= len(all) {
		return 0, 0, false
	}
	to := from + p.lines()
	if to > len(all) {
		to = len(all)
	}
	return all[from].para, all[to-1].para, true
}

// Measure implements Widget: the full width and a fixed height.
func (p *ParagraphPager) Measure(avail float64) geom.Size {
	return geom.Size{W: avail, H: lineHeight(p.textStyle().Size)*float64(p.lines()) + textPagerFooter}
}

func (p *ParagraphPager) footerRect() geom.Rect {
	r := p.ContentRect()
	return geom.R(r.X, r.Bottom()-textPagerFooter, r.W, textPagerFooter)
}

func (p *ParagraphPager) pagerRects() (prev, next geom.Rect) {
	f := p.footerRect()
	y := f.Y + (f.H-pagerBtn)/2
	next = geom.R(f.Right()-pagerBtn-16, y, pagerBtn, pagerBtn)
	prev = geom.R(next.X-pagerBtn-12, y, pagerBtn, pagerBtn)
	return prev, next
}

// Paint implements Widget.
func (p *ParagraphPager) Paint(c *gfx.Canvas) {
	ts := p.textStyle()
	if p.Disabled {
		ts.Color = Disabled
	}
	ts.Ellipsis = false
	r := p.ContentRect()
	lh := lineHeight(ts.Size)
	margin := p.margin()
	all := p.wrap(r.W)
	pages := p.Pages()
	page := int(geom.Clamp(float64(p.Page), 0, float64(pages-1)))
	from := page * p.lines()
	for i := 0; i < p.lines() && from+i < len(all); i++ {
		l := all[from+i]
		y := r.Y + float64(i)*lh
		if l.first && l.para < len(p.Numbers) {
			c.Text(p.Numbers[l.para], geom.Rect{X: r.X, Y: y, W: margin - 10, H: lh}, gfx.TextStyle{Size: p.numberSize(), Bold: true, Color: Muted, Align: gfx.AlignEnd})
		}
		c.Text(l.text, geom.Rect{X: r.X + margin, Y: y, W: r.W - margin, H: lh}, ts)
	}

	f := p.footerRect()
	c.Line(geom.Point{X: f.X, Y: f.Y}, geom.Point{X: f.Right(), Y: f.Y}, 2, Ink)
	c.Text(fmt.Sprintf("%d / %d", page+1, pages), geom.R(f.X+20, f.Y, f.W/2, f.H), gfx.TextStyle{Size: FontSmall, Color: Muted})
	prev, next := p.pagerRects()
	paintPagerButton(c, prev, gfx.IconChevronLeft, page > 0)
	paintPagerButton(c, next, gfx.IconChevronRight, page < pages-1)
}

// HandlePointer implements PointerHandler: the footer buttons turn the page.
func (p *ParagraphPager) HandlePointer(e *PointerEvent) {
	if e.Type != PointerClick {
		return
	}
	pt := geom.Point{X: e.X, Y: e.Y}
	prev, next := p.pagerRects()
	switch {
	case prev.Contains(pt):
		p.Prev()
	case next.Contains(pt):
		p.Next()
	}
}
