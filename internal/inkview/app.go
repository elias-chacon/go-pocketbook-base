package inkview

// PointerKind is the phase of a touch.
type PointerKind int

// Pointer phases.
const (
	PointerDown PointerKind = iota
	PointerMove
	PointerUp
)

// Keyboard flags for OpenKeyboard (subset of KBD_* in inkview.h).
const (
	KeyboardNormal  = 0
	KeyboardNumeric = 4
)

// App is implemented by the application and receives InkView events.
type App interface {
	// Init is called once when the app starts.
	Init()
	// Draw is called whenever the screen must be (re)painted.
	Draw()
	// Key handles a key press; return true if handled.
	Key(code int) bool
	// Pointer handles a touch phase at x,y (device pixels); return true if handled.
	Pointer(kind PointerKind, x, y int) bool
}
