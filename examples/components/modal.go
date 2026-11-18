package components

import (
	"fmt"

	"go-pocketbook-base/internal/toolkit/widget"
)

// ModalExample builds a dialog with OK and Cancel and the close, maximize and
// minimize buttons. It is shown over the page with m.Open(doc); it takes every
// touch until it is closed (the Back key closes it too, as Cancel).
func ModalExample() widget.Widget {
	nick := widget.NewInput("Nickname")
	m := widget.NewModal("Edit profile", widget.NewLabel("Choose how you appear to others."), nick)
	m.SetButtons(widget.ModalOKCancel)
	m.SetControls(widget.ModalClose | widget.ModalMaximize | widget.ModalMinimize)
	m.OnOK = func(e *widget.Event) { fmt.Println("saved:", nick.Value) }
	m.OnCancel = func(e *widget.Event) { fmt.Println("cancelled") }
	return m
}
