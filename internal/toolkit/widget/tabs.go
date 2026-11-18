package widget

import "strconv"

// TabMode says what happens to a tab's page when another tab is shown.
type TabMode int

// Tab modes.
const (
	// TabRemember (the default) drops the page tree and keeps only the state of
	// its Stateful elements (typed values, table page, selected date...) in the
	// Tabs' Store. Costs a few bytes per element and gives back every value when
	// the page is rebuilt. Elements need an ID to be remembered.
	TabRemember TabMode = iota
	// TabRebuild drops the page and its state: it starts fresh every time.
	TabRebuild
	// TabKeepAlive keeps the whole page tree in memory (everything survives,
	// including data), for pages that are expensive to build.
	TabKeepAlive
)

// Tab is one page of a Tabs.
type Tab struct {
	// ID scopes the saved state; defaults to the tab's position.
	ID    string
	Title string
	// Build creates the page. It runs each time the tab is shown, except for
	// TabKeepAlive tabs, where it runs once.
	Build func() Widget
	Mode  TabMode
	// Restored runs every time a TabRemember page is built, after the saved
	// state (if any) was put back, so content derived from values (a label that
	// shows a slider) can be refreshed.
	Restored func()

	kept Widget // TabKeepAlive only
}

// Tabs is a menu bar of pages plus the area where the current page is shown.
// Only the current page is in memory (unless a tab is TabKeepAlive).
type Tabs struct {
	Panel
	// Bar is the menu bar; append extra menus (e.g. "More") to Bar.Menus.
	Bar *MenuBar
	// Body holds the current page; style it (padding, grow) as needed.
	Body *Panel
	// Store keeps the state of TabRemember tabs.
	Store Store
	// OnChange runs after another tab is shown.
	OnChange Handler

	tabs    []*Tab
	current int
}

// NewTabs creates the tabs and shows the first one. store may be nil (then a
// MemoryStore is used).
func NewTabs(store Store, tabs ...*Tab) *Tabs {
	if store == nil {
		store = NewMemoryStore()
	}
	t := &Tabs{Store: store, tabs: tabs, current: -1}
	t.Init(t)
	t.Style.Display, t.Style.FlexDirection = DisplayFlex, Column
	t.Bar = NewMenuBar()
	for i, tab := range tabs {
		i := i
		t.Bar.Menus = append(t.Bar.Menus, &Menu{Label: tab.Title, OnSelect: func() { t.Show(i) }})
	}
	t.Body = NewColumn(0)
	t.Body.Style.FlexGrow = 1
	t.AppendChild(t.Bar)
	t.AppendChild(t.Body)
	if len(tabs) > 0 {
		t.Show(0)
	}
	return t
}

// Count is the number of tabs.
func (t *Tabs) Count() int { return len(t.tabs) }

// Current is the index of the tab being shown.
func (t *Tabs) Current() int { return t.current }

// Page returns the page being shown, or nil.
func (t *Tabs) Page() Widget {
	if len(t.Body.children) == 0 {
		return nil
	}
	return t.Body.children[0]
}

func (t *Tabs) scope(i int) string {
	if id := t.tabs[i].ID; id != "" {
		return id
	}
	return "tab" + strconv.Itoa(i)
}

// Save writes the state of the current page to the Store (Show does it when
// leaving a tab; call Save before the app exits if the Store is persistent).
func (t *Tabs) Save() {
	if p := t.Page(); p != nil && t.tabs[t.current].Mode == TabRemember {
		SaveState(p, t.Store, t.scope(t.current))
	}
}

// Move shows the tab delta places away (-1 previous, +1 next), wrapping around
// (for the previous/next page keys of the reader).
func (t *Tabs) Move(delta int) {
	if n := len(t.tabs); n > 0 {
		t.Show(((t.current+delta)%n + n) % n)
	}
}

// Show switches to tab i: the current page is saved (TabRemember), dropped
// (or kept, TabKeepAlive) and page i is built and restored.
func (t *Tabs) Show(i int) {
	if i < 0 || i >= len(t.tabs) {
		return
	}
	t.Save()
	t.Body.ClearChildren()

	tab := t.tabs[i]
	page := tab.kept
	if page == nil {
		page = tab.Build()
		if tab.Mode == TabKeepAlive {
			tab.kept = page
		}
		if tab.Mode == TabRemember {
			RestoreState(page, t.Store, t.scope(i))
			if tab.Restored != nil {
				tab.Restored()
			}
		}
	}
	t.Body.AppendChild(page)

	changed := t.current != i
	t.current = i
	for j, m := range t.Bar.Menus[:len(t.tabs)] {
		m.Selected = j == i
	}
	if t.doc != nil {
		t.doc.ClosePopups()
	}
	if changed {
		Fire(t.OnChange, t)
	}
}
