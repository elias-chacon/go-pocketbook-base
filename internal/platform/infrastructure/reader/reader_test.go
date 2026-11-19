package reader

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go-pocketbook-base/internal/device/inkview"
	"go-pocketbook-base/internal/platform/application"
	"go-pocketbook-base/internal/platform/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

// The adapter implements the application's and the domain's ports.
var (
	_ application.Screen      = (*Screen)(nil)
	_ widget.Keyboard         = Keyboard{}
	_ domain.DesktopRegistrar = DesktopRegistrar{}
	_ inkview.App             = (*Adapter)(nil)
)

func TestInkViewKeysAreNamedByWhatTheyDo(t *testing.T) {
	for code, want := range map[int]application.Key{
		inkview.KeyBack: application.KeyBack, inkview.KeyPrev: application.KeyPrevious,
		inkview.KeyNext: application.KeyNext, inkview.KeyMenu: application.KeyMenu,
		inkview.KeyOK: application.KeyOther, 0x41: application.KeyOther,
	} {
		if got := KeyOf(code); got != want {
			t.Errorf("KeyOf(%#x) = %v, want %v", code, got, want)
		}
	}
	for kind, want := range map[inkview.PointerKind]widget.PointerType{
		inkview.PointerDown: widget.PointerDown, inkview.PointerMove: widget.PointerMove, inkview.PointerUp: widget.PointerUp,
	} {
		if got := PointerOf(kind); got != want {
			t.Errorf("PointerOf(%v) = %v, want %v", kind, got, want)
		}
	}
}

// recorder is a Screen for the runtime behind the adapter.
type recorder struct{ shows, closes int }

func (r *recorder) Show(*image.Gray)                      { r.shows++ }
func (r *recorder) ShowRect(*image.Gray, image.Rectangle) {}
func (r *recorder) RequestRepaint()                       {}
func (r *recorder) Close()                                { r.closes++ }

type app struct{ doc *widget.Document }

func (a app) Document() *widget.Document { return a.doc }

func TestTheAdapterDrivesTheRuntimeWithTranslatedEvents(t *testing.T) {
	btn := widget.NewButton("Tap")
	clicks := 0
	btn.OnClick = func(*widget.Event) { clicks++ }
	doc := widget.NewDocument(1404, 1872)
	doc.SetRoot(widget.NewColumn(0, btn))
	rec := &recorder{}
	rt := application.NewRuntime(application.Env{}, func(application.Env) application.App { return app{doc} }, rec)
	rt.Start(1404, 1872, Keyboard{})
	a := NewAdapter(rt)

	a.Draw()
	if rec.shows != 1 {
		t.Errorf("Draw must reach the screen: %d", rec.shows)
	}
	c := btn.Bounds.Center()
	a.Pointer(inkview.PointerDown, int(c.X), int(c.Y))
	a.Pointer(inkview.PointerUp, int(c.X), int(c.Y))
	if clicks != 1 {
		t.Errorf("a touch must reach the element: %d", clicks)
	}
	if !a.Key(inkview.KeyBack) || rec.closes != 1 {
		t.Errorf("a Back key with nothing open leaves: closes=%d", rec.closes)
	}
	if a.Key(inkview.KeyOK) {
		t.Error("an unknown key is not handled")
	}
}

func TestTheDeviceOutputsAreHarmlessOffDevice(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 10, 10))
	screen := &Screen{}
	screen.Show(img)
	screen.ShowRect(img, image.Rect(0, 0, 5, 5))
	screen.RequestRepaint()
	if screen.ClosedByApp() {
		t.Error("nobody closed it yet")
	}
	screen.Close()
	if !screen.ClosedByApp() {
		t.Error("Close is the app asking to leave")
	}
	Keyboard{}.Request(widget.KeyboardRequest{Kind: widget.KeyboardNumeric})
	Keyboard{}.Request(widget.KeyboardRequest{})
	SetLogFile(filepath.Join(t.TempDir(), "x.log"))
	Logf("hello %d", 1)
	SetLogFile("")
	if err := Run(application.NewRuntime(application.Env{}, nil, &Screen{})); err != inkview.ErrNotOnDevice {
		t.Errorf("off-device Run = %v", err)
	}
	if IconPath("/mnt/ext1", "hello") != filepath.Join("/mnt/ext1", "applications", "icons", "hello.bmp") {
		t.Errorf("icon path = %q", IconPath("/mnt/ext1", "hello"))
	}
}

// A minimal desktop config in the shape the desktop package expects.
const viewJSON = `{
    "applications": {
    },
    "groups": [
        { "title": "@General", "apps": [
        ] }
    ]
}`

func TestTheRegistrarPutsTheAppOnTheHomeScreen(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "system", "profiles", "default", "config", "desktop")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "view.json"), []byte(viewJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	reg := DesktopRegistrar{Root: root}
	r := domain.Registration{
		Exe: filepath.Join(root, "applications", "hello.app"), Title: "Olá Go", Group: domain.DefaultGroup,
		Icon: func(size int, focused bool) image.Image { return image.NewGray(image.Rect(0, 0, size, size)) },
	}

	changed, err := reg.Register(r)
	if err != nil || !changed {
		t.Fatalf("first registration: changed=%v err=%v", changed, err)
	}
	view, _ := os.ReadFile(filepath.Join(dir, "view.json"))
	if !strings.Contains(string(view), "U_hello") || !strings.Contains(string(view), "Olá Go") {
		t.Errorf("view.json = %s", view)
	}
	if _, err := os.Stat(IconPath(root, "hello")); err != nil {
		t.Errorf("the icon is where IconPath says: %v", err)
	}
	if changed, err := reg.Register(r); err != nil || changed {
		t.Errorf("registering again is idempotent: changed=%v err=%v", changed, err)
	}
	if _, err := (DesktopRegistrar{Root: t.TempDir()}).Register(r); err == nil {
		t.Error("no view.json under the root is an error")
	}
}

func TestHTTPClientIsTheReadersNetworkClientWithTheGivenTimeout(t *testing.T) {
	c := HTTPClient(7 * time.Second)
	if c == nil || c.Timeout != 7*time.Second {
		t.Errorf("%+v", c)
	}
}
