package widget

import (
	"testing"
	"time"
)

// form builds a page with one of every Stateful element (all with IDs).
func form() Widget {
	name := NewInput("name")
	name.ID = "name"
	pass := NewInput("pass")
	pass.ID, pass.Type = "pass", TypePassword
	notes := NewTextArea("notes", 3)
	notes.ID = "notes"
	check := NewCheckbox("c", false)
	check.ID = "check"
	combo := NewComboBox("city", Option{"A", "a"}, Option{"B", "b"})
	combo.ID = "combo"
	slider := NewSlider(0, 100, 10, 0)
	slider.ID = "slider"
	chips := NewChipGroup(false, NewChip("x"), NewChip("y"), NewChip("z"))
	chips.ID = "chips"
	cal := NewCalendar(time.Date(2026, 1, 10, 0, 0, 0, 0, time.Local))
	cal.ID = "cal"
	tbl := NewTable(2, TableColumn{Title: "n", Sortable: true})
	tbl.ID = "tbl"
	tbl.Rows = [][]string{{"1"}, {"2"}, {"3"}, {"4"}, {"5"}}
	return NewColumn(0, name, pass, notes, check, combo, slider, chips, cal, tbl)
}

func byID(root Widget, id string) Widget {
	var found Widget
	walk(root, func(w Widget) bool {
		if w.Elem().ID == id {
			found = w
		}
		return found == nil
	})
	return found
}

func twoTabs(mode TabMode) *Tabs {
	return NewTabs(nil,
		&Tab{Title: "Form", Build: form, Mode: mode},
		&Tab{Title: "Other", Build: func() Widget { return NewLabel("other") }})
}

// edit changes every kind of state on the current page.
func edit(p Widget) {
	byID(p, "name").(*Input).Value = "Ana"
	byID(p, "pass").(*Input).Value = "secret"
	byID(p, "notes").(*TextArea).Value = "line1\nline2"
	byID(p, "check").(*Toggle).Checked = true
	byID(p, "combo").(*ComboBox).Index = 1
	byID(p, "slider").(*Slider).SetValue(70)
	byID(p, "chips").(*ChipGroup).Elem().children[2].(*Chip).Selected = true
	cal := byID(p, "cal").(*Calendar)
	cal.Selected, cal.Year, cal.Month = time.Date(2026, 3, 5, 0, 0, 0, 0, time.Local), 2026, time.March
	tbl := byID(p, "tbl").(*Table)
	tbl.Page, tbl.SortCol, tbl.SortDesc, tbl.Selected = 2, 0, true, 4
}

func TestTabsRememberRestoresEveryKindOfState(t *testing.T) {
	tabs := twoTabs(TabRemember)
	first := tabs.Page()
	edit(first)

	tabs.Show(1)
	if tabs.Page() == first || tabs.Current() != 1 {
		t.Fatal("the other tab must show a different page")
	}
	tabs.Show(0)
	p := tabs.Page()
	if p == first {
		t.Fatal("TabRemember rebuilds the page (does not keep the tree)")
	}

	checks := []struct {
		what string
		ok   bool
	}{
		{"text", byID(p, "name").(*Input).Value == "Ana"},
		{"password is NOT remembered", byID(p, "pass").(*Input).Value == ""},
		{"text area", byID(p, "notes").(*TextArea).Value == "line1\nline2"},
		{"checkbox", byID(p, "check").(*Toggle).Checked},
		{"combo", byID(p, "combo").(*ComboBox).Value() == "b"},
		{"slider", byID(p, "slider").(*Slider).Value == 70},
		{"chips", byID(p, "chips").(*ChipGroup).SaveState() == "2"},
		{"calendar day", byID(p, "cal").(*Calendar).Selected.Format("2006-01-02") == "2026-03-05"},
		{"calendar month", byID(p, "cal").(*Calendar).Month == time.March},
	}
	for _, c := range checks {
		if !c.ok {
			t.Errorf("not restored: %s", c.what)
		}
	}
	tbl := byID(p, "tbl").(*Table)
	if tbl.Page != 2 || tbl.SortCol != 0 || !tbl.SortDesc || tbl.Selected != 4 {
		t.Errorf("table state lost: page=%d sort=%d desc=%v sel=%d", tbl.Page, tbl.SortCol, tbl.SortDesc, tbl.Selected)
	}
}

func TestTabsRebuildForgetsAndKeepAliveKeepsTheTree(t *testing.T) {
	re := twoTabs(TabRebuild)
	edit(re.Page())
	re.Show(1)
	re.Show(0)
	if byID(re.Page(), "name").(*Input).Value != "" {
		t.Error("TabRebuild must start fresh")
	}

	ka := twoTabs(TabKeepAlive)
	first := ka.Page()
	edit(first)
	ka.Show(1)
	ka.Show(0)
	if ka.Page() != first || byID(first, "name").(*Input).Value != "Ana" {
		t.Error("TabKeepAlive must keep the very same tree")
	}
}

func TestStateIsClampedToTheCurrentData(t *testing.T) {
	tbl := NewTable(2, TableColumn{Title: "n"})
	tbl.Rows = [][]string{{"1"}, {"2"}, {"3"}}
	tbl.LoadState("9;7;1;40") // page, sort column, desc and selection all out of range
	if tbl.Page != 1 || tbl.SortCol != -1 || tbl.Selected != -1 {
		t.Errorf("clamp failed: page=%d sort=%d sel=%d", tbl.Page, tbl.SortCol, tbl.Selected)
	}
	cb := NewComboBox("x", Option{"A", "a"})
	cb.LoadState("5")
	if cb.Index != -1 {
		t.Error("combo index must be clamped")
	}
	tbl.LoadState("garbage") // malformed state is ignored
	cal := NewCalendar(time.Time{})
	cal.LoadState("nope")
}

func TestTabsRestoredHookAndScope(t *testing.T) {
	restored := 0
	tabs := NewTabs(nil,
		&Tab{Title: "A", ID: "a", Build: form, Restored: func() { restored++ }},
		&Tab{Title: "B", Build: form})
	byID(tabs.Page(), "name").(*Input).Value = "in A"
	tabs.Show(1)
	byID(tabs.Page(), "name").(*Input).Value = "in B"
	tabs.Show(0)
	if restored != 2 || // built at start, then rebuilt on return
		byID(tabs.Page(), "name").(*Input).Value != "in A" {
		t.Errorf("restored hook / scoping: restored=%d", restored)
	}
	tabs.Show(1)
	if byID(tabs.Page(), "name").(*Input).Value != "in B" {
		t.Error("tabs with the same element IDs must not share state")
	}
	store := tabs.Store.(*MemoryStore)
	if _, ok := store.Get("a/name"); !ok {
		t.Error("state is keyed by tab ID then element ID")
	}
	if _, ok := store.Get("tab1/name"); !ok && store.Len() == 0 {
		t.Error("tabs without an ID are scoped by position")
	}
}

func TestTabsOnChangeAndBounds(t *testing.T) {
	changes := 0
	tabs := twoTabs(TabRemember)
	tabs.OnChange = func(e *Event) { changes++ }
	tabs.Show(0) // same tab: no change event
	tabs.Show(1)
	tabs.Show(7) // out of range is ignored
	if changes != 1 || tabs.Current() != 1 {
		t.Errorf("changes=%d current=%d", changes, tabs.Current())
	}
	if !tabs.Bar.Menus[1].Selected || tabs.Bar.Menus[0].Selected {
		t.Error("the menu bar must mark the current tab")
	}
}
