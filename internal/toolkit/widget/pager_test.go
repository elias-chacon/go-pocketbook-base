package widget

import (
	"fmt"
	"strings"
	"testing"
)

func rows(n int) []Widget {
	out := make([]Widget, n)
	for i := range out {
		l := NewLabel(fmt.Sprintf("row %d", i))
		l.ID = fmt.Sprintf("row-%d", i)
		out[i] = l
	}
	return out
}

func visibleRows(d *Document) []string {
	var ids []string
	d.Walk(func(w Widget) bool {
		if id := w.Elem().ID; strings.HasPrefix(id, "row-") {
			ids = append(ids, id)
		}
		return true
	})
	return ids
}

func indicatorOf(p *Pager) string { return p.indicator.Text }

func TestAPagerShowsOnePageOfItemsAndTurnsPages(t *testing.T) {
	p := NewPager(4, rows(10)...)
	d := testDoc(p.Root)
	if p.Len() != 10 || p.PageSize() != 4 || p.Pages() != 3 || p.Page() != 0 {
		t.Fatalf("len=%d size=%d pages=%d page=%d", p.Len(), p.PageSize(), p.Pages(), p.Page())
	}
	if got := visibleRows(d); len(got) != 4 || got[0] != "row-0" || got[3] != "row-3" {
		t.Errorf("the first page: %v", got)
	}
	if indicatorOf(p) != "1 / 3" || !p.prev.Disabled || p.next.Disabled {
		t.Errorf("footer: %q prev.disabled=%v next.disabled=%v", indicatorOf(p), p.prev.Disabled, p.next.Disabled)
	}

	tap(d, p.next)
	d.Layout()
	if got := visibleRows(d); len(got) != 4 || got[0] != "row-4" || indicatorOf(p) != "2 / 3" {
		t.Errorf("the second page: %v %s", got, indicatorOf(p))
	}
	p.Next()
	d.Layout()
	if got := visibleRows(d); len(got) != 2 || got[1] != "row-9" || !p.next.Disabled || p.prev.Disabled {
		t.Errorf("the last page holds the rest: %v", got)
	}
	p.Next() // stops at the end
	if p.Page() != 2 {
		t.Error("no page after the last")
	}
	tap(d, p.prev)
	if p.Page() != 1 {
		t.Errorf("the previous button: %d", p.Page())
	}
}

func TestPagerCallsOnPageOnlyWhenThePageChanges(t *testing.T) {
	p := NewPager(2, rows(5)...)
	var c counter
	p.OnPage = c.handler()
	p.GoTo(0) // already there
	p.Prev()  // at the start
	if c.n != 0 {
		t.Errorf("no change, no call: %d", c.n)
	}
	p.GoTo(2)
	p.GoTo(99) // clamped to the same last page
	if c.n != 1 || p.Page() != 2 {
		t.Errorf("calls=%d page=%d", c.n, p.Page())
	}
	p.GoTo(-5)
	if p.Page() != 0 || c.n != 2 {
		t.Errorf("clamped to the first: %d %d", p.Page(), c.n)
	}
}

func TestReplacingTheItemsKeepsThePageWhenItStillExists(t *testing.T) {
	p := NewPager(3, rows(9)...)
	p.GoTo(2)
	p.SetItems(rows(7)) // 3 pages: still page 2
	if p.Page() != 2 || p.Pages() != 3 {
		t.Errorf("%d %d", p.Page(), p.Pages())
	}
	p.SetItems(rows(2)) // one page: back to it
	if p.Page() != 0 || indicatorOf(p) != "1 / 1" || !p.next.Disabled || !p.prev.Disabled {
		t.Errorf("%d %q", p.Page(), indicatorOf(p))
	}
	p.SetItems(nil)
	if p.Pages() != 1 || p.Len() != 0 || indicatorOf(p) != "1 / 1" {
		t.Error("an empty list is one empty page")
	}
	if NewPager(0).PageSize() != 1 {
		t.Error("a page holds at least one item")
	}
}

func TestThePagerButtonsAreBigEnoughToTouch(t *testing.T) {
	p := NewPager(3, rows(9)...)
	testDoc(p.Root)
	for name, b := range map[string]*Button{"prev": p.prev, "next": p.next} {
		if b.Bounds.W < MinTouch || b.Bounds.H < MinTouch {
			t.Errorf("%s is %vx%v", name, b.Bounds.W, b.Bounds.H)
		}
	}
}
