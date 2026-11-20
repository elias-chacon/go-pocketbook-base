package widget

import (
	"fmt"
	"testing"
)

func verses(n int) (texts, numbers []string) {
	for i := 1; i <= n; i++ {
		texts = append(texts, fmt.Sprintf("Texto do versículo %d. %s", i, longText(12)))
		numbers = append(numbers, fmt.Sprint(i))
	}
	return texts, numbers
}

func TestAParagraphPagerKnowsWhichParagraphsAreOnThePage(t *testing.T) {
	texts, numbers := verses(40)
	pp := NewParagraphPager(texts, numbers)
	pp.Lines = 10
	d := testDoc(pp)
	if pp.Pages() < 3 {
		t.Fatalf("40 long paragraphs in 10 lines need several pages: %d", pp.Pages())
	}
	first, last, ok := pp.Visible()
	if !ok || first != 0 || last < 1 {
		t.Fatalf("page 1 shows %d..%d (%v)", first, last, ok)
	}

	var c counter
	pp.OnPage = c.handler()
	_, next := pp.pagerRects()
	tapAt(d, next.Center())
	f2, l2, _ := pp.Visible()
	if pp.Page != 1 || c.n != 1 || f2 < first || l2 <= last {
		t.Errorf("after next: page=%d calls=%d visible=%d..%d (before %d..%d)", pp.Page, c.n, f2, l2, first, last)
	}
	prev, _ := pp.pagerRects()
	tapAt(d, prev.Center())
	tapAt(d, prev.Center())
	if pp.Page != 0 {
		t.Errorf("back at the start: %d", pp.Page)
	}
	tapAt(d, geomPt(pp.Bounds.X+20, pp.Bounds.Y+20))
	if pp.Page != 0 {
		t.Error("only the footer turns pages")
	}
	if img := d.Render(); img == nil {
		t.Fatal("it paints")
	}
}

func TestAParagraphPagerGoesToAParagraphAndStartsOverWithNewContent(t *testing.T) {
	texts, numbers := verses(40)
	pp := NewParagraphPager(texts, numbers)
	pp.Lines = 10
	testDoc(pp)
	pp.GoToParagraph(30)
	first, last, _ := pp.Visible()
	if first > 30 || last < 30 && first != 30 {
		t.Errorf("paragraph 30 must be reachable from the page shown: %d..%d", first, last)
	}
	pp.GoTo(999)
	if pp.Page != pp.Pages()-1 {
		t.Errorf("the page is clamped: %d of %d", pp.Page, pp.Pages())
	}
	pp.SetParagraphs([]string{"one"}, nil)
	if pp.Page != 0 || pp.Pages() != 1 {
		t.Errorf("new content: page=%d pages=%d", pp.Page, pp.Pages())
	}
	if f, l, ok := pp.Visible(); !ok || f != 0 || l != 0 {
		t.Errorf("visible = %d..%d %v", f, l, ok)
	}
	pp.SetParagraphs(nil, nil)
	if _, _, ok := pp.Visible(); ok {
		t.Error("nothing to show, nothing visible")
	}
}
