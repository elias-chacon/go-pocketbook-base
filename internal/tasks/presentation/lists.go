package presentation

import "go-pocketbook-base/internal/toolkit/widget"

// listsMenu is the drop-down to pick the list that is shown (and receives new
// tasks), to start a new list and to delete the one shown.
func (s *Screen) listsMenu() *widget.Dropdown {
	lists, err := s.cfg.Checklist.Lists()
	if err != nil {
		s.fail(err)
	}
	s.lists = lists
	label := s.t("tasks.lists")
	items := []widget.MenuItem{{Label: s.t("tasks.list.all"), OnSelect: func() { s.ShowList("") }}}
	for _, l := range lists {
		l := l
		if l.ID == s.listID {
			label = l.Name
		}
		items = append(items, widget.MenuItem{Label: l.Name, OnSelect: func() { s.ShowList(l.ID) }})
	}
	items = append(items, widget.MenuItem{Separator: true},
		widget.MenuItem{Label: s.t("tasks.list.new"), OnSelect: s.StartNewList},
		widget.MenuItem{Label: s.t("tasks.list.delete"), Disabled: s.listID == "" || len(lists) <= 1, OnSelect: s.DeleteShownList})
	d := widget.NewDropdown(label, items...)
	d.ID = "tasks-lists"
	return d
}

// ShowList shows the tasks of one list ("" for every list).
func (s *Screen) ShowList(id string) {
	s.listID, s.page, s.naming = id, 0, false
	s.showList()
}

// StartNewList makes the add field ask for a list name.
func (s *Screen) StartNewList() {
	s.naming = true
	s.showList()
}

func (s *Screen) createList(name string) {
	l, err := s.cfg.Checklist.CreateList(name)
	if err != nil {
		s.naming = false
		s.fail(err)
		return
	}
	s.naming, s.listID, s.page, s.notice = false, l.ID, 0, ""
	s.showList()
}

// DeleteShownList deletes the list that is shown, with its tasks.
func (s *Screen) DeleteShownList() {
	if s.listID == "" {
		return
	}
	if err := s.cfg.Checklist.DeleteList(s.listID); err != nil {
		s.fail(err)
		return
	}
	s.listID, s.page = "", 0
	s.showList()
}
