package widget

import (
	"fmt"
	"strings"
	"testing"

	"go-pocketbook-base/internal/kernel/geom"
)

func longText(words int) string {
	var b strings.Builder
	for i := 0; i < words; i++ {
		fmt.Fprintf(&b, "palavra%d ", i)
	}
	return b.String()
}

func TestATextPagerSplitsALongTextIntoPages(t *testing.T) {
	tp := NewTextPager(longText(400))
	tp.Lines = 8
	d := testDoc(tp)
	if tp.Pages() < 3 {
		t.Fatalf("400 words in 8 lines need several pages: %d", tp.Pages())
	}
	h := tp.Bounds.H
	if h < lineHeight(tp.textStyle().Size)*8 {
		t.Errorf("the height holds the lines and the footer: %v", h)
	}
	if testing.Short() {
		return
	}
	if img := d.Render(); img == nil {
		t.Fatal("it paints")
	}

	var c counter
	tp.OnPage = c.handler()
	_, next := tp.pagerRects()
	tapAt(d, next.Center())
	if tp.Page != 1 || c.n != 1 {
		t.Errorf("the next button: page=%d calls=%d", tp.Page, c.n)
	}
	prev, _ := tp.pagerRects()
	tapAt(d, prev.Center())
	tapAt(d, prev.Center()) // at the start: nothing
	if tp.Page != 0 {
		t.Errorf("the previous button: %d", tp.Page)
	}
	tapAt(d, geomPt(tp.Bounds.X+20, tp.Bounds.Y+20)) // a touch on the text turns nothing
	if tp.Page != 0 {
		t.Error("only the footer turns pages")
	}
}

func TestATextPagerStopsAtTheEndsAndStartsOverWithNewText(t *testing.T) {
	tp := NewTextPager(longText(300))
	tp.Lines = 6
	testDoc(tp)
	last := tp.Pages() - 1
	tp.GoTo(999)
	if tp.Page != last {
		t.Errorf("clamped to the last page: %d of %d", tp.Page, last)
	}
	tp.Next()
	if tp.Page != last {
		t.Error("no page after the last")
	}
	tp.GoTo(-4)
	if tp.Page != 0 {
		t.Error("clamped to the first")
	}
	tp.GoTo(2)
	tp.SetText("curto")
	if tp.Page != 0 || tp.Pages() != 1 {
		t.Errorf("a new text starts on the first page: %d %d", tp.Page, tp.Pages())
	}
	tp.SetText("")
	if tp.Pages() != 1 {
		t.Error("an empty text is one page")
	}
}

func TestATextPagerFollowsTheWidthAndHonorsParagraphs(t *testing.T) {
	tp := NewTextPager("primeiro parágrafo\n\nsegundo parágrafo")
	tp.Lines = 10
	testDoc(tp)
	lines := tp.wrap(tp.ContentRect().W)
	if len(lines) != 3 || lines[1] != "" {
		t.Errorf("a blank line keeps the paragraphs apart: %q", lines)
	}

	wide := NewTextPager(longText(60))
	narrow := NewTextPager(longText(60))
	wide.Lines, narrow.Lines = 5, 5
	testDoc(wide)
	narrow.Style.Width = Px(500)
	testDoc(narrow)
	if narrow.Pages() <= wide.Pages() {
		t.Errorf("a narrower box needs more pages: %d vs %d", narrow.Pages(), wide.Pages())
	}

	cached := narrow.wrap(narrow.ContentRect().W)
	again := narrow.wrap(narrow.ContentRect().W)
	if len(cached) != len(again) || &cached[0] != &again[0] {
		t.Error("the wrapped text is kept while nothing changes")
	}
	narrow.Style.FontSize = 40
	if len(narrow.wrap(narrow.ContentRect().W)) <= len(cached) {
		t.Error("a bigger font wraps into more lines")
	}
}

func TestATextPagerCanBeDisabledAndDefaultsTo20Lines(t *testing.T) {
	tp := NewTextPager("x")
	tp.Disabled = true
	d := testDoc(tp)
	if tp.lines() != textPagerLines {
		t.Errorf("default lines: %d", tp.lines())
	}
	if d.Render() == nil {
		t.Error("a disabled viewer still paints")
	}
	if got := tp.Measure(1000); got.W != 1000 || got.H <= geom.Max(0, textPagerFooter) {
		t.Errorf("Measure: %+v", got)
	}
}
