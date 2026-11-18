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

// Key codes (subset of inkview.h). On the PB741 the left key is KeyPrev and the
// right key KeyNext; Home never reaches the app (the system closes it).
const (
	KeyBack = 0x1b
	KeyOK   = 0x0a
	KeyMenu = 0x17
	KeyPrev = 0x18
	KeyNext = 0x19
)

// IsBackKey reports whether code should act as "Back". The PB741 has no Back
// key, only Home, previous page (left), next page (right) and Menu, so the left
// and Menu keys stand in for it.
func IsBackKey(code int) bool { return code == KeyBack || code == KeyPrev || code == KeyMenu }
