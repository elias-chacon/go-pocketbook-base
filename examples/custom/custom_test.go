package custom

import (
	"testing"

	"go-pocketbook-base/examples/harness"
	"go-pocketbook-base/internal/toolkit/widget"
)

func TestCustomWidgets(t *testing.T) {
	badge := NewBadge(7)
	counter := NewTapCounter()
	changes := 0
	counter.OnChange = func(e *widget.Event) { changes++ }
	doc, _ := harness.NewDoc(widget.NewColumn(20, badge, counter))
	if badge.Bounds.W <= 0 || badge.Bounds.H <= 0 || harness.Inked(doc) == 0 {
		t.Fatal("the badge must be laid out and drawn")
	}
	if doc.TargetAt(counter.Bounds.Center()) != widget.Widget(counter) {
		t.Fatal("the counter must receive its own touches")
	}
	harness.Tap(doc, counter)
	harness.Tap(doc, counter)
	if counter.Count != 2 || changes != 2 || counter.label() != "Taps: 2" {
		t.Errorf("Count=%d changes=%d label=%q", counter.Count, changes, counter.label())
	}
	if doc.TargetAt(badge.Bounds.Center()) == widget.Widget(badge) {
		t.Error("a badge is display-only and must not capture touches")
	}
}
