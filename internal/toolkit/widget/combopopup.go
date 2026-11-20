package widget

import (
	"fmt"

	"go-pocketbook-base/internal/kernel/fold"
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
)

// popupGap is the space between the field and the popup under it (see Document.OpenPopup).
const popupGap = 6.0

// comboChrome is what a paged popup spends besides its rows: the padding, the search field, the
// page footer and the gaps between them.
const comboChrome = 12 + ControlHeight + 6 + 6 + ControlHeight

// comboPopup is the list a ComboBox opens when its options do not fit the screen: the same
// popup under the field, but with a search field on top and the options a page at a time
// (e-ink does not scroll), with previous and next buttons and a "page / pages" indicator. It is
// a popup like the short list's, not a window: the page below stays visible and a tap outside
// closes it. Its size is fixed when it opens, so filtering never makes it jump.
type comboPopup struct {
	Panel
	cb        *ComboBox
	search    *Input
	body      *Panel
	indicator *Label
	prev      *Button
	next      *Button

	matches []int // indexes of the options the search lets through
	page    int
	rows    int // options per page
}

// comboRows is how many options a paged popup shows for a screen where room design pixels are free
// next to the field (0 when not even one fits).
func comboRows(room float64) int {
	return int((room - comboChrome) / menuRowH)
}

// newComboPopup builds the popup with rows options per page, as wide as width.
func newComboPopup(cb *ComboBox, rows int, width float64) *comboPopup {
	p := &comboPopup{cb: cb, rows: rows}
	p.Init(p)
	s := &p.Style
	s.Display, s.FlexDirection, s.Gap = DisplayFlex, Column, 6
	s.Background, s.BorderColor, s.BorderWidth, s.BorderRadius = Paper, Ink, BorderThin, Radius
	s.Padding = geom.All(6)
	s.Width, s.Height = Px(width), Px(comboChrome+float64(rows)*menuRowH)

	p.search = NewSearchInput(cb.SearchPlaceholder)
	p.search.ID = cb.ID + "-search"
	p.search.OnChange = func(*Event) { p.filter(p.search.Value) }
	p.search.OnSubmit = p.search.OnChange

	p.body = NewColumn(0)
	p.body.Style.Height = Px(float64(rows) * menuRowH)

	p.indicator = NewLabel("")
	p.indicator.Style.FontSize, p.indicator.Style.Color, p.indicator.Style.FlexGrow = FontSmall, Muted, 1
	p.prev, p.next = NewIconButton(gfx.IconChevronLeft), NewIconButton(gfx.IconChevronRight)
	p.prev.ID, p.next.ID = cb.ID+"-prev", cb.ID+"-next"
	p.prev.OnClick = func(*Event) { p.goTo(p.page - 1) }
	p.next.OnClick = func(*Event) { p.goTo(p.page + 1) }
	footer := NewRow(12, p.indicator, p.prev, p.next)
	footer.Style.Padding = geom.XY(8, 0)

	p.AppendChild(p.search)
	p.AppendChild(p.body)
	p.AppendChild(footer)
	p.filter("")
	if i := p.position(cb.Index); i >= 0 {
		p.page = i / rows // opens on the page of the chosen option
		p.show()
	}
	return p
}

// position is where option i is among the matches (-1 when it is not).
func (p *comboPopup) position(i int) int {
	for n, m := range p.matches {
		if m == i {
			return n
		}
	}
	return -1
}

// filter keeps the options that have every word of the query, ignoring case and accents.
func (p *comboPopup) filter(query string) {
	words := fold.Words(query)
	p.matches = p.matches[:0]
	for i, o := range p.cb.Options {
		if len(words) == 0 || fold.MatchesAll(o.Label, words) {
			p.matches = append(p.matches, i)
		}
	}
	p.page = 0
	p.show()
}

func (p *comboPopup) pages() int {
	if len(p.matches) == 0 {
		return 1
	}
	return (len(p.matches) + p.rows - 1) / p.rows
}

func (p *comboPopup) goTo(n int) {
	p.page = int(geom.Clamp(float64(n), 0, float64(p.pages()-1)))
	p.show()
}

// show puts the rows of the current page in the body and updates the footer.
func (p *comboPopup) show() {
	from := p.page * p.rows
	to := from + p.rows
	if to > len(p.matches) {
		to = len(p.matches)
	}
	if from > to {
		from = to
	}
	page := p.matches[from:to]
	items := make([]MenuItem, len(page))
	selected := -1
	for n, i := range page {
		items[n] = MenuItem{Label: p.cb.Options[i].Label, Value: p.cb.Options[i].Value}
		if i == p.cb.Index {
			selected = n
		}
	}
	p.body.ClearChildren()
	if len(items) == 0 {
		none := NewLabel(p.cb.NoMatchText)
		none.Style.Color, none.Style.Padding = Muted, geom.All(menuPadX)
		p.body.AppendChild(none)
	} else {
		list := newMenuList(items, selected, func(n int) { p.cb.choose(page[n]) })
		list.Style.BorderWidth, list.Style.Padding = 0, geom.Insets{}
		p.body.AppendChild(list)
	}
	p.indicator.Text = fmt.Sprintf("%d / %d", p.page+1, p.pages())
	p.prev.Disabled = p.page == 0
	p.next.Disabled = p.page >= p.pages()-1
	if p.doc != nil {
		attach(p.body, p.doc)
		layoutWidget(p, p.Bounds) // the popup keeps its place and size; only the rows changed
		p.Invalidate()
	}
}
