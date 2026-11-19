package components

import (
	"fmt"

	"go-pocketbook-base/internal/toolkit/widget"
)

// PagerExample is a list of rows shown four at a time, with previous and next buttons.
func PagerExample() widget.Widget {
	var rows []widget.Widget
	for i := 1; i <= 10; i++ {
		row := widget.NewLabel(fmt.Sprintf("Message %d", i))
		row.OnClick = func(e *widget.Event) { fmt.Println("opened", row.Text) }
		rows = append(rows, row)
	}
	pager := widget.NewPager(4, rows...)
	pager.OnPage = func(e *widget.Event) { fmt.Println("now on page", pager.Page()+1, "of", pager.Pages()) }
	return pager.Root // add this to the document
}

// TextPagerExample is a long text read by turning pages.
func TextPagerExample() widget.Widget {
	article := widget.NewTextPager("A long text is read by turning pages, not by scrolling. " +
		"Each page holds a fixed number of lines, so nothing moves while you read.")
	article.Lines = 12 // lines per page (20 by default)
	article.OnPage = func(e *widget.Event) { fmt.Println("page", article.Page+1) }
	return article
}
