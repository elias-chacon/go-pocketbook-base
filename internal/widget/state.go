package widget

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Stateful is implemented by elements whose user-changed state (typed text, a
// checked box, the table page...) can be saved and restored as a short string.
// Data (a table's Rows, a combo's Options) is NOT state: the code that builds
// the element provides it again.
type Stateful interface {
	Widget
	SaveState() string
	LoadState(state string)
}

// Store keeps saved element state by key. The default MemoryStore costs a few
// bytes per element; another implementation could persist it to a file.
type Store interface {
	Get(key string) (string, bool)
	Set(key, value string)
}

// MemoryStore is a Store in RAM.
type MemoryStore struct{ m map[string]string }

// NewMemoryStore creates an empty MemoryStore.
func NewMemoryStore() *MemoryStore { return &MemoryStore{m: map[string]string{}} }

// Get implements Store.
func (s *MemoryStore) Get(key string) (string, bool) { v, ok := s.m[key]; return v, ok }

// Set implements Store.
func (s *MemoryStore) Set(key, value string) { s.m[key] = value }

// Len is the number of saved entries.
func (s *MemoryStore) Len() int { return len(s.m) }

// SaveState stores the state of every Stateful element under root that has an
// ID, as scope + "/" + ID. Elements without an ID are not saved.
func SaveState(root Widget, store Store, scope string) {
	walk(root, func(w Widget) bool {
		if st, ok := w.(Stateful); ok && w.Elem().ID != "" {
			store.Set(scope+"/"+w.Elem().ID, st.SaveState())
		}
		return true
	})
}

// RestoreState is the inverse of SaveState: elements with no saved entry keep
// the state their builder gave them.
func RestoreState(root Widget, store Store, scope string) {
	walk(root, func(w Widget) bool {
		if st, ok := w.(Stateful); ok && w.Elem().ID != "" {
			if v, found := store.Get(scope + "/" + w.Elem().ID); found {
				st.LoadState(v)
			}
		}
		return true
	})
}

func atoi(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

func boolState(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// ---- Input / TextArea: the text ----

// SaveState implements Stateful. Passwords are never saved.
func (in *Input) SaveState() string {
	if in.Type == TypePassword {
		return ""
	}
	return in.Value
}

// LoadState implements Stateful.
func (in *Input) LoadState(s string) {
	if in.Type != TypePassword {
		in.Value = s
	}
}

// SaveState implements Stateful.
func (t *TextArea) SaveState() string { return t.Value }

// LoadState implements Stateful.
func (t *TextArea) LoadState(s string) { t.Value = s }

// ---- Toggle (checkbox, radio, switch) ----

// SaveState implements Stateful.
func (t *Toggle) SaveState() string { return boolState(t.Checked) }

// LoadState implements Stateful.
func (t *Toggle) LoadState(s string) { t.Checked = s == "1" }

// ---- ComboBox: the selected index ----

// SaveState implements Stateful.
func (cb *ComboBox) SaveState() string { return strconv.Itoa(cb.Index) }

// LoadState implements Stateful.
func (cb *ComboBox) LoadState(s string) {
	cb.Index = atoi(s, -1)
	if cb.Index >= len(cb.Options) {
		cb.Index = -1
	}
}

// ---- Slider ----

// SaveState implements Stateful.
func (s *Slider) SaveState() string { return strconv.FormatFloat(s.Value, 'g', -1, 64) }

// LoadState implements Stateful.
func (s *Slider) LoadState(v string) {
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		s.SetValue(f)
	}
}

// ---- ChipGroup: which chips are selected (by position) ----

// SaveState implements Stateful.
func (g *ChipGroup) SaveState() string {
	var sel []string
	for i, k := range g.children {
		if c, ok := k.(*Chip); ok && c.Selected {
			sel = append(sel, strconv.Itoa(i))
		}
	}
	return strings.Join(sel, ",")
}

// LoadState implements Stateful. Chips the user closed are not brought back:
// the builder decides which chips exist.
func (g *ChipGroup) LoadState(s string) {
	g.deselectAll()
	for _, f := range strings.Split(s, ",") {
		if i := atoi(f, -1); i >= 0 && i < len(g.children) {
			if c, ok := g.children[i].(*Chip); ok {
				c.Selected = true
			}
		}
	}
}

// ---- Calendar: the selected day and the month on screen ----

const dateLayout = "2006-01-02"

// SaveState implements Stateful.
func (c *Calendar) SaveState() string {
	sel := ""
	if !c.Selected.IsZero() {
		sel = c.Selected.Format(dateLayout)
	}
	return fmt.Sprintf("%s;%d;%d", sel, c.Year, int(c.Month))
}

// LoadState implements Stateful.
func (c *Calendar) LoadState(s string) {
	parts := strings.Split(s, ";")
	if len(parts) != 3 {
		return
	}
	c.Selected = time.Time{}
	if d, err := time.ParseInLocation(dateLayout, parts[0], time.Local); err == nil {
		c.Selected = d
	}
	if y, m := atoi(parts[1], 0), atoi(parts[2], 0); y > 0 && m >= 1 && m <= 12 {
		c.Year, c.Month = y, time.Month(m)
	}
}

// ---- Table: page, sort and selection (the Rows are data, not state) ----

// SaveState implements Stateful.
func (t *Table) SaveState() string {
	return fmt.Sprintf("%d;%d;%s;%d", t.Page, t.SortCol, boolState(t.SortDesc), t.Selected)
}

// LoadState implements Stateful. Values are clamped to the current data.
func (t *Table) LoadState(s string) {
	parts := strings.Split(s, ";")
	if len(parts) != 4 {
		return
	}
	t.SortCol, t.SortDesc = atoi(parts[1], -1), parts[2] == "1"
	if t.SortCol >= len(t.Columns) {
		t.SortCol = -1
	}
	t.Page = atoi(parts[0], 0)
	if t.Page >= t.PageCount() {
		t.Page = t.PageCount() - 1
	}
	if t.Page < 0 {
		t.Page = 0
	}
	t.Selected = atoi(parts[3], -1)
	if t.Selected >= len(t.Rows) {
		t.Selected = -1
	}
}
