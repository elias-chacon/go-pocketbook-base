package application

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"reflect"
	"strings"
	"testing"

	"go-pocketbook-base/internal/platform/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

const helloManifest = `{
  "id": "hello",
  "title": {"pt-BR": "Olá Go", "en": "Hello Go"},
  "description": {"pt-BR": "Vitrine", "en": "Showcase"}
}`

func manifest(t *testing.T) domain.Manifest {
	t.Helper()
	m, err := domain.ParseManifest([]byte(helloManifest))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// ---- the runtime ----

// fakeApp is a one-button app; keys goes to its KeyHandler.
type fakeApp struct {
	doc   *widget.Document
	keys  []Key
	taken bool
}

func (f *fakeApp) Document() *widget.Document { return f.doc }
func (f *fakeApp) Key(k Key) bool             { f.keys = append(f.keys, k); return f.taken }

// recorder is a Screen that remembers what it was asked.
type recorder struct {
	shows, repaints, closes int
	rects                   []image.Rectangle
}

func (r *recorder) Show(*image.Gray)                          { r.shows++ }
func (r *recorder) ShowRect(_ *image.Gray, a image.Rectangle) { r.rects = append(r.rects, a) }
func (r *recorder) RequestRepaint()                           { r.repaints++ }
func (r *recorder) Close()                                    { r.closes++ }

func newRuntime(t *testing.T) (*Runtime, *fakeApp, *widget.Button, *recorder) {
	t.Helper()
	btn := widget.NewButton("Tap me")
	doc := widget.NewDocument(1404, 1872)
	doc.SetRoot(widget.NewColumn(0, btn))
	app := &fakeApp{doc: doc}
	rec := &recorder{}
	rt := NewRuntime(Env{}, func(Env) App { return app }, rec)
	rt.Start(1404, 1872, &widget.FakeKeyboard{})
	return rt, app, btn, rec
}

func TestStartGivesTheAppTheScreenAndWiresItsExit(t *testing.T) {
	var got Env
	rec := &recorder{}
	doc := widget.NewDocument(10, 10)
	kb := &widget.FakeKeyboard{}
	rt := NewRuntime(Env{Lang: "en"}, func(e Env) App { got = e; return &fakeApp{doc: doc} }, rec)
	rt.Start(758, 1024, kb)
	if got.Width != 758 || got.Height != 1024 || got.Keyboard != widget.Keyboard(kb) || got.Lang != "en" {
		t.Errorf("env = %+v", got)
	}
	got.Exit()
	if rec.closes != 1 {
		t.Error("Env.Exit closes the screen")
	}
}

func TestRuntimeRepaintsOnlyWhatChangedAndRefreshesFullyEveryTenUpdates(t *testing.T) {
	rt, _, btn, rec := newRuntime(t)
	rt.Draw() // the system asks for the first draw: everything
	if rec.shows != 1 {
		t.Fatalf("first draw must show the whole screen: %d", rec.shows)
	}

	for i := 0; i < fullRefreshEvery; i++ {
		btn.Text = "changed " + string(rune('a'+i))
		btn.Invalidate() // our own repaint request: partial
		if rec.repaints == 0 {
			t.Fatal("a change must request a repaint")
		}
		rt.Draw()
	}
	if len(rec.rects) != fullRefreshEvery || rec.shows != 1 {
		t.Fatalf("expected %d partial updates and no extra full one yet: rects=%d shows=%d", fullRefreshEvery, len(rec.rects), rec.shows)
	}
	btn.Text = "one more"
	btn.Invalidate()
	rt.Draw()
	if rec.shows != 2 {
		t.Errorf("after %d partial updates the next one is a full refresh: shows=%d", fullRefreshEvery, rec.shows)
	}

	before := rec.shows
	rt.Draw() // not requested by us (pending is false): the system covered the screen
	if rec.shows != before+1 {
		t.Error("a draw the system asked for must repaint everything")
	}
}

func TestRuntimeCoalescesRepaintRequestsAndSkipsMovesWhileBusy(t *testing.T) {
	rt, _, btn, rec := newRuntime(t)
	rt.Draw()
	rec.repaints = 0
	btn.Text = "a"
	btn.Invalidate()
	btn.Text = "b"
	btn.Invalidate()
	if rec.repaints != 1 {
		t.Errorf("two changes, one queued repaint: %d", rec.repaints)
	}
	if !rt.Pointer(widget.PointerMove, 5, 5) {
		t.Error("a move while the screen catches up is swallowed")
	}
	clicks := 0
	btn.OnClick = func(*widget.Event) { clicks++ }
	rt.pending = false
	c := btn.Bounds.Center()
	rt.Pointer(widget.PointerDown, int(c.X), int(c.Y))
	rt.Pointer(widget.PointerUp, int(c.X), int(c.Y))
	if clicks != 1 {
		t.Errorf("touch must reach the document: %d clicks", clicks)
	}
}

func TestRuntimeKeys(t *testing.T) {
	rt, app, _, rec := newRuntime(t)
	rt.Draw()

	// with a modal open, left and Menu cancel it before the app sees them
	m := widget.NewModal("Hi", widget.NewLabel("x"))
	m.Open(rt.doc)
	if !rt.Key(KeyPrevious) || rt.doc.HasModal() || len(app.keys) != 0 {
		t.Fatalf("left must close the modal and not reach the app: keys=%v", app.keys)
	}
	m.Open(rt.doc)
	if !rt.Key(KeyMenu) || rt.doc.HasModal() {
		t.Error("Menu must close the modal too")
	}

	// nothing open: the app decides
	if rt.Key(KeyNext) || !reflect.DeepEqual(app.keys, []Key{KeyNext}) {
		t.Errorf("an unhandled key goes to the app and is reported unhandled: %v", app.keys)
	}
	app.taken = true
	if !rt.Key(KeyPrevious) {
		t.Error("a key the app takes is handled")
	}

	// a real Back key leaves, but only when nothing took it
	app.taken = false
	if !rt.Key(KeyBack) || rec.closes != 1 {
		t.Errorf("Back with nothing open leaves: closes=%d", rec.closes)
	}
}

type plainApp struct{ doc *widget.Document }

func (p plainApp) Document() *widget.Document { return p.doc }

func TestRuntimeWithAnAppThatHandlesNoKeys(t *testing.T) {
	rt := NewRuntime(Env{}, func(Env) App { return plainApp{widget.NewDocument(1404, 1872)} }, &recorder{})
	rt.Start(1404, 1872, nil)
	if rt.Key(KeyNext) || rt.Key(KeyOther) {
		t.Error("no KeyHandler: other keys are not handled")
	}
}

func TestWhichKeysActAsBack(t *testing.T) {
	for k, want := range map[Key]bool{KeyBack: true, KeyPrevious: true, KeyMenu: true, KeyNext: false, KeyOther: false} {
		if k.IsBack() != want {
			t.Errorf("Key(%d).IsBack() = %v", k, k.IsBack())
		}
	}
}

// ---- first run ----

type fakeRegistrar struct {
	got     domain.Registration
	changed bool
	err     error
}

func (f *fakeRegistrar) Register(r domain.Registration) (bool, error) {
	f.got = r
	return f.changed, f.err
}

type fakeRegistry struct {
	saved   []domain.InstalledApp
	saveErr error
}

func (f *fakeRegistry) Save(a domain.InstalledApp) error {
	f.saved = append(f.saved, a)
	return f.saveErr
}
func (f *fakeRegistry) List() []domain.InstalledApp { return f.saved }

func input(t *testing.T, lang string) FirstRunInput {
	return FirstRunInput{Manifest: manifest(t), Lang: lang, Version: "1.2.3", Exe: "/mnt/ext1/applications/hello.app", IconPath: "/i/hello.bmp"}
}

func TestFirstRunRegistersTheAppAndRecordsIt(t *testing.T) {
	reg, apps := &fakeRegistrar{changed: true}, &fakeRegistry{}
	var logged []string
	uc := FirstRun{Registrar: reg, Apps: apps, Logf: func(f string, a ...interface{}) { logged = append(logged, f) }}

	notice := uc.Execute(input(t, "pt-BR"))
	if !strings.Contains(notice, "Ícone registrado") {
		t.Errorf("says what happened, in Portuguese: %q", notice)
	}
	if reg.got.Title != "Olá Go" || reg.got.Group != domain.DefaultGroup || reg.got.Exe == "" || reg.got.Icon == nil {
		t.Errorf("registration = %+v (translated title, default group, default icon)", reg.got)
	}
	if len(apps.saved) != 1 || apps.saved[0].Version != "1.2.3" || apps.saved[0].Icon != "/i/hello.bmp" {
		t.Errorf("record = %+v", apps.saved)
	}
	if len(logged) == 0 {
		t.Error("what happened is logged")
	}

	reg.changed = false // a second run
	if again := uc.Execute(input(t, "en")); again != "" {
		t.Errorf("a second run has nothing to say: %q", again)
	}
	if reg.got.Title != "Hello Go" {
		t.Errorf("the title follows the language: %q", reg.got.Title)
	}
}

func TestFirstRunReportsFailuresInTheChosenLanguageAndSurvivesARecordError(t *testing.T) {
	failing := FirstRun{Registrar: &fakeRegistrar{err: errors.New("no view.json")}, Apps: &fakeRegistry{}}
	if got := failing.Execute(input(t, "en")); got != "Could not register the icon: no view.json" {
		t.Errorf("notice = %q", got)
	}
	if got := failing.Execute(input(t, "pt-BR")); !strings.HasPrefix(got, "Falha ao registrar o ícone:") {
		t.Errorf("notice = %q", got)
	}

	// not being listed must not stop the app (and a nil Logf is fine)
	ok := FirstRun{Registrar: &fakeRegistrar{changed: true}, Apps: &fakeRegistry{saveErr: errors.New("disk full")}}
	if got := ok.Execute(input(t, "pt-BR")); !strings.Contains(got, "registrado") {
		t.Errorf("a record error is not an install error: %q", got)
	}
}

func TestFirstRunUsesTheAppsOwnIconWhenItHasOne(t *testing.T) {
	reg := &fakeRegistrar{}
	called := false
	in := input(t, "pt-BR")
	in.Icon = func(size int, focused bool) image.Image {
		called = true
		return image.NewGray(image.Rect(0, 0, size, size))
	}
	FirstRun{Registrar: reg, Apps: &fakeRegistry{}}.Execute(in)
	reg.got.Icon(8, false)
	if !called {
		t.Error("the app's icon renderer must be the one registered")
	}
}

func TestDefaultIconHasTheInitialAndAnInvertedVariant(t *testing.T) {
	draw := DefaultIcon("Livros")
	normal, focused := draw(128, false).(*image.Gray), draw(128, true).(*image.Gray)
	if normal.Bounds().Dx() != 128 || normal.GrayAt(2, 2).Y != 255 || focused.GrayAt(2, 2).Y != 0 {
		t.Errorf("corner: normal=%d focused=%d", normal.GrayAt(2, 2).Y, focused.GrayAt(2, 2).Y)
	}
	dark := 0
	for _, p := range normal.Pix {
		if p < 128 {
			dark++
		}
	}
	if dark == 0 {
		t.Error("the icon draws something")
	}
	if initial("  livros") != "L" || initial("") != "?" || initial("é") != "É" {
		t.Errorf("initials: %q %q %q", initial("  livros"), initial(""), initial("é"))
	}
}

func pngOf(t *testing.T, img image.Image) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestIconFromImageScalesToGrayAndInvertsTheFocusedOne(t *testing.T) {
	src := image.NewGray(image.Rect(0, 0, 64, 64)) // black
	icon := IconFromImage(src)

	normal, focused := icon(128, false).(*image.Gray), icon(128, true).(*image.Gray)
	if normal.Bounds().Dx() != 128 || focused.Bounds().Dy() != 128 {
		t.Fatalf("size %v", normal.Bounds())
	}
	if normal.GrayAt(64, 64).Y > 5 || focused.GrayAt(64, 64).Y < 250 {
		t.Errorf("black on the normal one, white on the focused one: %d %d", normal.GrayAt(64, 64).Y, focused.GrayAt(64, 64).Y)
	}
}

func TestATransparentPartOfAnIconShowsWhite(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 32, 32)) // fully transparent
	if v := IconFromImage(src)(64, false).(*image.Gray).GrayAt(10, 10).Y; v != 255 {
		t.Errorf("transparent = white, got %d", v)
	}
	// FitContain leaves margins when the picture is not square
	wide := image.NewGray(image.Rect(0, 0, 64, 16))
	if v := IconFromImage(wide)(64, false).(*image.Gray).GrayAt(32, 2).Y; v != 255 {
		t.Errorf("the margin is white, got %d", v)
	}
}

func TestIconFromPNGDecodesOrReportsAnError(t *testing.T) {
	icon, err := IconFromPNG(pngOf(t, image.NewGray(image.Rect(0, 0, 128, 128))))
	if err != nil || icon(128, false) == nil {
		t.Fatalf("%v", err)
	}
	if _, err := IconFromPNG([]byte("not an image")); err == nil {
		t.Error("garbage is an error")
	}
	// a huge picture is shrunk while decoding, not kept whole
	big, err := IconFromPNG(pngOf(t, image.NewGray(image.Rect(0, 0, 2000, 2000))))
	if err != nil || big(32, false).Bounds().Dx() != 32 {
		t.Errorf("%v", err)
	}
}
