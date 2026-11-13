package widget

// KeyboardKind selects the on-screen keyboard layout.
type KeyboardKind int

// Keyboard kinds.
const (
	KeyboardText KeyboardKind = iota
	KeyboardNumeric
)

// KeyboardRequest asks the system for text input.
type KeyboardRequest struct {
	Title   string
	Initial string
	Kind    KeyboardKind
	MaxLen  int
	// Done receives the typed text; ok is false when the user cancelled.
	Done func(text string, ok bool)
}

// Keyboard abstracts the platform's text entry (InkView's keyboard on the
// device, a fake in tests and the desktop preview).
type Keyboard interface {
	Request(KeyboardRequest)
}

// FakeKeyboard answers every request immediately with Reply (or leaves the
// text unchanged when Cancel is set). It records the last request.
type FakeKeyboard struct {
	Reply  string
	Cancel bool
	Last   KeyboardRequest
}

// Request implements Keyboard.
func (k *FakeKeyboard) Request(r KeyboardRequest) {
	k.Last = r
	if k.Cancel {
		r.Done(r.Initial, false)
		return
	}
	r.Done(k.Reply, true)
}
