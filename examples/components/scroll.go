package components

import (
	"fmt"

	"go-pocketbook-base/internal/toolkit/widget"
)

// ScrollExample is a panel of a fixed height with more content than fits: a window with a scroll
// bar. The content moves by dragging a finger over it, by dragging the thumb of the bar or by
// tapping the bar above or below the thumb.
func ScrollExample() widget.Widget {
	var cards []widget.Widget
	for i := 1; i <= 6; i++ {
		cards = append(cards, widget.NewCard(fmt.Sprintf("Card %d", i), "Content taller than the window"))
	}
	more := widget.NewButton("A button in the content")
	more.OnClick = func(e *widget.Event) { fmt.Println("tapped") }
	cards = append(cards, more)

	scroll := widget.NewScroll(cards...)
	scroll.Style.Height = widget.Px(560) // a definite height: the window
	scroll.OnScroll = func(e *widget.Event) { fmt.Println("offset", scroll.Offset) }
	return scroll
}
