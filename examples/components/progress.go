package components

import (
	"go-pocketbook-base/internal/toolkit/widget"
)

// ProgressExample is a progress bar; change Value and the bar follows.
func ProgressExample() widget.Widget {
	bar := widget.NewProgress(35)
	bar.Value = 60 // 0..100
	return bar
}
