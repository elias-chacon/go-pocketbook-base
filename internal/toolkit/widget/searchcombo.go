package widget

import (
	"strconv"
	"strings"

	"go-pocketbook-base/internal/kernel/fold"
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
)

// SearchComboBox is a ComboBox for a list too long to look through: tapping it opens a window with
// a search field and the options a page at a time (there is no scrolling on e-ink). It opens on the
// page where the chosen option is, typing filters the options by any part of their text, ignoring
// the case and the accents, and a tap on an option chooses it and closes the window.
//
// It is a ComboBox (Options, Index, Value, SetValue, OnChange), so it replaces one anywhere.
//
//	box := widget.NewSearchComboBox("Language", options...)
//	box.Title, box.SearchPlaceholder = "Pick a language", "Search"
//	box.OnChange = func(e *widget.Event) { fmt.Println(box.Value()) }
type SearchComboBox struct {
	ComboBox
	// Title is the title of the window (the placeholder when empty).
	Title string
	// SearchPlaceholder is the hint of the search field; CancelText the label of the button that
	// closes the window without choosing; NoMatchText what shows when nothing matches.
	SearchPlaceholder, CancelText, NoMatchText string

	modal *Modal
	pager *Pager
	query string
}

// NewSearchComboBox creates a searchable combo box with nothing selected.
func NewSearchComboBox(placeholder string, options ...Option) *SearchComboBox {
	sb := &SearchComboBox{SearchPlaceholder: "Search", CancelText: "Cancel", NoMatchText: "Nothing found"}
	sb.Options, sb.Index, sb.Placeholder = options, -1, placeholder
	sb.Init(sb)
	applyFieldStyle(&sb.Style)
	return sb
}

// HandlePointer implements PointerHandler: a tap opens the window.
func (sb *SearchComboBox) HandlePointer(e *PointerEvent) {
	if e.Type == PointerClick {
		sb.Open()
	}
}

// Open shows the window of options (what a tap does).
func (sb *SearchComboBox) Open() {
	if sb.doc == nil {
		return
	}
	sb.query = ""
	title := sb.Title
	if title == "" {
		title = sb.Placeholder
	}
	search := NewSearchInput(sb.SearchPlaceholder)
	search.ID = sb.ID + "-search"
	search.OnChange = func(*Event) { sb.filter(search.Value) }
	search.OnSubmit = search.OnChange

	sb.pager = NewPager(sb.rowsPerPage())
	sb.pager.body.Style.Gap = 8
	sb.modal = NewModal(title, search, sb.pager.Root)
	sb.modal.ID = sb.ID + "-picker"
	sb.modal.OKText = sb.CancelText
	sb.modal.SetButtons(ModalOK)
	sb.modal.Body.Style.Gap = 12
	sb.fill(sb.Index)
	sb.modal.Open(sb.doc)
}

// rowsPerPage is how many options fit the window on this screen.
func (sb *SearchComboBox) rowsPerPage() int {
	n := int((sb.doc.H - 640) / (ControlHeight + 10))
	return int(geom.Clamp(float64(n), 3, 12))
}

// filter shows only the options that have every word of the query.
func (sb *SearchComboBox) filter(query string) {
	sb.query = strings.TrimSpace(query)
	sb.fill(sb.Index)
}

// matches lists the indexes of the options the query lets through (all of them when it is empty).
func (sb *SearchComboBox) matches() []int {
	words := fold.Words(sb.query)
	var out []int
	for i, o := range sb.Options {
		if len(words) == 0 || fold.MatchesAll(o.Label, words) {
			out = append(out, i)
		}
	}
	return out
}

// fill puts the matching options in the pager, on the page that has option focus (the first page
// when it is not among them).
func (sb *SearchComboBox) fill(focus int) {
	idx := sb.matches()
	items := make([]Widget, 0, len(idx))
	page := 0
	for n, i := range idx {
		i := i
		b := NewButton(sb.Options[i].Label)
		b.ID = sb.ID + "-opt-" + strconv.Itoa(i)
		b.Style.TextAlign = gfx.AlignStart
		b.Style.Bold = i == sb.Index
		b.Style.Padding = geom.XY(0, 18)
		if i == sb.Index {
			b.SetVariant(VariantPrimary)
		}
		b.OnClick = func(*Event) { sb.choose(i) }
		items = append(items, b)
		if i == focus {
			page = n / sb.pager.PageSize()
		}
	}
	if len(items) == 0 {
		none := NewLabel(sb.NoMatchText)
		none.Style.Color = Muted
		items = append(items, none)
	}
	sb.pager.SetItems(items)
	sb.pager.GoTo(page)
	sb.pager.show() // the page may not have changed, the items did
}

// choose selects option i, closes the window and tells OnChange (in that order: the handler may
// open another window).
func (sb *SearchComboBox) choose(i int) {
	sb.Index = i
	sb.modal.Close()
	sb.modal = nil
	Fire(sb.OnChange, sb)
	sb.Invalidate()
}
