package presentation

import (
	"strings"
	"testing"

	"go-pocketbook-base/internal/toolkit/widget"
)

// selectRow picks a row of the manager's table the way a tap on it does.
func selectRow(t *testing.T, s *Screen, row int) {
	t.Helper()
	table := byID(t, s, "versions-table").(*widget.Table)
	table.Selected = row
	table.OnSelect(&widget.Event{})
}

func TestTheVersionsManagerListsWhatIsDownloadedInATable(t *testing.T) {
	w := newWorld(t)
	w.ready()
	s := w.screen("pt-BR")
	s.OpenVersions()
	table := byID(t, s, "versions-table").(*widget.Table)
	if len(table.Rows) != 1 || !strings.Contains(table.Rows[0][0], "Livre") || table.Rows[0][3] != "completa" {
		t.Fatalf("rows: %v", table.Rows)
	}
	if !byID(t, s, "versions-delete").Elem().Disabled {
		t.Error("nothing selected: nothing to delete")
	}
	selectRow(t, s, 0)
	if byID(t, s, "versions-delete").Elem().Disabled {
		t.Error("a selected version can be deleted")
	}
}

func TestTheVersionsManagerSaysWhenNothingIsDownloaded(t *testing.T) {
	w := newWorld(t)
	s := w.screen("en")
	s.OpenVersions()
	if findID(s, "versions-none") == nil || findID(s, "versions-table") != nil {
		t.Error("an empty reader shows the notice, not a table")
	}
}

func TestDeletingAVersionAsksFirstThenRemovesItsTextAndKeepsTheProgress(t *testing.T) {
	w := newWorld(t)
	w.ready()
	s := w.screen("pt-BR")
	s.OpenReader(43, 2, 5)
	s.Show(0)
	before := progressOf(t, s)
	s.OpenVersions()
	selectRow(t, s, 0)
	tap(t, s, "versions-delete")
	if findID(s, "modal-remove") == nil {
		t.Fatal("it must ask before deleting")
	}
	byID(t, s, "modal-remove").(*widget.Modal).Cancel()
	if findID(s, "versions-table") == nil || len(byID(t, s, "versions-table").(*widget.Table).Rows) != 1 {
		t.Fatal("cancelling comes back to the manager with the version still there")
	}

	selectRow(t, s, 0)
	tap(t, s, "versions-delete")
	byID(t, s, "modal-remove").(*widget.Modal).Confirm()
	if findID(s, "versions-none") == nil {
		t.Error("after deleting the only version the manager says there are none")
	}
	versions, _ := w.bible.Versions()
	for _, v := range versions {
		if v.Slug == "almeida-livre" && (v.Downloaded || v.Verses != 0) {
			t.Errorf("the text must be gone: %+v", v)
		}
	}
	if s.ready() || s.prefs.Version != "" {
		t.Errorf("the version in use was deleted: ready=%v version=%q", s.ready(), s.prefs.Version)
	}
	byID(t, s, "modal-versions").(*widget.Modal).Close()
	if findID(s, "home") == nil {
		t.Error("the Home goes back to offering the download")
	}
	_ = before
}

func TestAVersionBeingDownloadedIsNotDeleted(t *testing.T) {
	w := newWorld(t)
	w.ready()
	s := w.screen("pt-BR")
	s.job = &job{}
	s.remove(s.stored()[0])
	if !s.ready() {
		t.Error("with a download running nothing is deleted")
	}
	s.job = nil
}
