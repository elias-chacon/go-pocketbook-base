# 10. Testing

[← Custom components](09-custom-components.md) · [Index](README.md) · [Next: Troubleshooting →](11-troubleshooting.md)

Because the UI is pure Go, you can test **screens, layout and touch** on your PC with the normal `go test`,
without a reader. This is the main reason to build the interface this way.

## The three tools

| Tool | What it gives you |
|---|---|
| `widget.FakeKeyboard` | a keyboard that answers immediately: `Reply` is the text typed, `Cancel` simulates cancelling, `Last` is the last request |
| `doc.HandlePointer(...)` | simulates a finger (down, move, up) at a point |
| `doc.Render()` / `harness.SavePNG` | draws the screen so you can inspect pixels or look at a PNG |

## Helpers

Most tests start with these small helpers (they live in `examples/harness/testkit.go`):

```go
// examples/harness/testkit.go#NewDoc
// NewDoc puts content in a Screen with a fake keyboard (it answers "7") and lays
// it out: the usual start of a test.
func NewDoc(content widget.Widget) (*widget.Document, *widget.FakeKeyboard) {
	kb := &widget.FakeKeyboard{Reply: "7"}
	doc := Screen(1404, 1872, kb, content)
	doc.Layout()
	return doc, kb
}
```

```go
// examples/harness/testkit.go#TapAt
// TapAt touches the screen at p (design pixels): finger down, then up.
func TapAt(doc *widget.Document, p geom.Point) {
	doc.HandlePointer(widget.PointerDown, p.X*doc.Scale, p.Y*doc.Scale)
	doc.HandlePointer(widget.PointerUp, p.X*doc.Scale, p.Y*doc.Scale)
}
```

```go
// examples/harness/testkit.go#Tap
// Tap touches the center of w.
func Tap(doc *widget.Document, w widget.Widget) { TapAt(doc, w.Elem().Bounds.Center()) }
```

```go
// examples/harness/testkit.go#Interactive
// Interactive returns the visible elements that react to touch.
func Interactive(doc *widget.Document) []widget.Widget {
	var out []widget.Widget
	doc.Walk(func(w widget.Widget) bool {
		if _, ok := w.(widget.PointerHandler); ok && w.Elem().Visible() {
			out = append(out, w)
		}
		return true
	})
	return out
}
```

- `newDoc` builds a document with a fake keyboard and lays it out, so every element already has its `Bounds`.
- `tapAt` is a complete touch (down and up) at a point in **design pixels**.
- `tap` touches the **center** of an element — the same place a finger would.
- `interactive` lists the elements that react to touch.

## Test a button

```go
// examples/recipes/recipes_test.go#TestHello
func TestHello(t *testing.T) {
	closed := false
	doc, _ := harness.NewDoc(Hello(func() { closed = true }))
	for _, w := range harness.Interactive(doc) {
		harness.Tap(doc, w)
	}
	if !closed {
		t.Error("the button must call onClose")
	}
}
```

## Test a form with the keyboard

Set `kb.Reply` to what the "user" types, then tap the field:

```go
// examples/recipes/recipes_test.go#TestLoginFormValidatesBeforeCallingOnLogin
func TestLoginFormValidatesBeforeCallingOnLogin(t *testing.T) {
	var user string
	var remember bool
	f := NewLoginForm(func(u string, r bool) { user, remember = u, r })
	doc, kb := harness.NewDoc(f.Root)
	submit := harness.Interactive(doc)[len(harness.Interactive(doc))-1] // the last element is the button

	harness.Tap(doc, submit)
	if f.Message.Hidden || f.Message.Text != "Type your user name" || user != "" {
		t.Fatalf("empty user: message=%q hidden=%v", f.Message.Text, f.Message.Hidden)
	}

	kb.Reply = "ana"
	harness.Tap(doc, f.User)
	kb.Reply = "123"
	harness.Tap(doc, f.Password)
	harness.Tap(doc, submit)
	if f.Message.Text != "The password needs at least 4 characters" {
		t.Fatalf("short password: %q", f.Message.Text)
	}

	kb.Reply = "1234"
	harness.Tap(doc, f.Password)
	harness.Tap(doc, f.Remember) // untick "Remember me"
	harness.Tap(doc, submit)
	if user != "ana" || remember || !f.Message.Hidden {
		t.Errorf("login: user=%q remember=%v hidden=%v", user, remember, f.Message.Hidden)
	}
}
```

To simulate the user cancelling the keyboard, set `kb.Cancel = true`: the field keeps its value.

## Test that the touch lands on the element

The project's rule is that **a touch at the center of every interactive element must reach that element**
(nothing covers it, nothing is cut off). The demo test applies it to every page:

```go
for _, w := range interactive(doc) {
	if w.Elem().Disabled {
		continue // a disabled element correctly ignores touch
	}
	if got := doc.TargetAt(w.Elem().Bounds.Center()); got != w {
		t.Errorf("a touch at the center of %T reaches %T", w, got)
	}
}
```

`doc.TargetAt(point)` returns the element that would receive a touch (popups first).

## Test layout

After `doc.Layout()` (or `newDoc`) each element has `Bounds` (`X`, `Y`, `W`, `H`, in design pixels):

```go
doc, _ := newDoc(widget.NewRow(12, a, b))
if b.Bounds.X <= a.Bounds.Right() {
	t.Error("b must be to the right of a")
}
```

## Test what is drawn

```go
img := doc.Render() // *image.Gray

blackPixels := 0
for _, p := range img.Pix {
	if p < 128 {
		blackPixels++
	}
}
if blackPixels == 0 {
	t.Error("nothing was drawn")
}
```

For something you can see, `harness.SavePNG(doc, "screen.png")` and open the file.

## Popups

Combo-boxes and menus open a popup. Tap the element, then use `doc.HasPopup()`; tap again outside (or call
`doc.ClosePopups()`) to close it:

```go
tap(doc, combo)
if !doc.HasPopup() {
	t.Fatal("the list must open")
}
doc.ClosePopups()
```

## Documentation samples

`go test ./tools/docsync` fails when a page of `docs/` no longer matches the code in `examples/`. Fix it with
`go run ./tools/docsync`.

## Coverage and quality

```sh
go test -cover ./...
.\scripts\sonar.ps1        # coverage + static analysis in the local SonarQube
```

Code that only runs on the reader (`cmd/`, `internal/device/inkview`) is excluded from coverage; keep your logic in
packages that run on the PC, and keep the reader glue thin.

Next: [Troubleshooting →](11-troubleshooting.md)

## Testing on an emulated reader (ARM)

Everything that does not touch the screen can be tested on the reader's own processor without the reader: `.\scripts\arm-test.ps1`
compiles the tests of every pure-Go package (`internal/**`) for **ARMv7 with the SDK's Go 1.17** and runs them under `qemu-arm` in an Alpine
container (no privileges, no changes to Docker). The container has **no zoneinfo and no system libraries**, like the reader, so a test that
forgot to carry what it needs fails there: that is how the time zone database was found missing in the calendar domain.

| It checks | It cannot check |
|---|---|
| the code builds and runs on ARM with the SDK's Go (SQLite, the AES-GCM vault, MIME, iCalendar, daylight saving, rendering) | InkView (it needs `libinkview` and the screen) |
| the behaviour without zoneinfo or a C library | the FAT filesystem of the reader's storage (it needs a privileged mount) |
| | real speed and memory: the emulation is slow and says nothing about the e-ink panel |

All 44 packages pass this way. A test of the whole app on the device is still the only way to check the screen, the touch, Wi-Fi and the FAT volume.
