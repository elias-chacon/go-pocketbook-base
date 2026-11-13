package widget

// PointerType is the phase of a touch.
type PointerType int

// Pointer phases. Click is synthesized after Up when the finger is released
// over the same element it went down on.
const (
	PointerDown PointerType = iota
	PointerMove
	PointerUp
	PointerClick
)

// PointerEvent is delivered to widgets that implement PointerHandler.
// X and Y are in design pixels (the same space as Base.Bounds).
type PointerEvent struct {
	Type PointerType
	X, Y float64
}

// PointerHandler is implemented by interactive widgets.
type PointerHandler interface {
	HandlePointer(e *PointerEvent)
}

// Event is what JavaScript-style callbacks (OnClick, OnChange…) receive.
type Event struct {
	Target  Widget  // the element the event is about
	X, Y    float64 // touch position in design pixels (JS clientX/clientY), for click events
	stopped bool
}

// StopPropagation keeps the event from bubbling to ancestors' OnClick.
func (e *Event) StopPropagation() { e.stopped = true }

// Handler is a JavaScript-style event callback.
type Handler func(e *Event)

func fire(h Handler, target Widget) {
	if h != nil {
		h(&Event{Target: target})
	}
}
