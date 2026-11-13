package inkview

// App is implemented by the application and receives InkView events.
type App interface {
	// Init is called once when the app starts.
	Init()
	// Draw is called whenever the screen must be (re)painted.
	Draw()
	// Key handles a key press; return true if handled.
	Key(code int) bool
	// Pointer handles a tap/click release at x,y; return true if handled.
	Pointer(x, y int) bool
}
