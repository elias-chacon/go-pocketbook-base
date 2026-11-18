package appkit

import (
	"image"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go-pocketbook-base/internal/inkview"
	"go-pocketbook-base/internal/widget"
)

const helloManifest = `{
  "id": "hello",
  "title": {"pt-BR": "Olá Go", "en": "Hello Go"},
  "description": {"pt-BR": "Vitrine", "en": "Showcase"}
}`

func TestParseManifest(t *testing.T) {
	m, err := ParseManifest([]byte(helloManifest))
	if err != nil || m.ID != "hello" || m.DesktopGroup() != DefaultGroup {
		t.Fatalf("%+v %v", m, err)
	}
	m.Group = "@Games"
	if m.DesktopGroup() != "@Games" {
		t.Error("an explicit group wins")
	}
}

func TestManifestRulesOfTheContract(t *testing.T) {
	bad := map[string]string{
		"not json":                 `{`,
		"no id":                    `{"title":{"pt-BR":"A"},"description":{"pt-BR":"B"}}`,
		"upper-case id":            `{"id":"Hello","title":{"pt-BR":"A"},"description":{"pt-BR":"B"}}`,
		"id that is not a name":    `{"id":"a/b","title":{"pt-BR":"A"},"description":{"pt-BR":"B"}}`,
		"title only in English":    `{"id":"x","title":{"en":"A"},"description":{"pt-BR":"B"}}`,
		"empty description":        `{"id":"x","title":{"pt-BR":"A"},"description":{"pt-BR":"  "}}`,
		"id starting with a digit": `{"id":"1x","title":{"pt-BR":"A"},"description":{"pt-BR":"B"}}`,
	}
	for name, data := range bad {
		if _, err := ParseManifest([]byte(data)); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
	err := Manifest{ID: "A b"}.Validate()
	if err == nil || !strings.Contains(err.Error(), "primary language") {
		t.Errorf("all problems are reported together: %v", err)
	}
}

func TestTitlesFallBackLikeI18n(t *testing.T) {
	m, _ := ParseManifest([]byte(helloManifest))
	for lang, want := range map[string]string{"pt-BR": "Olá Go", "pt_BR": "Olá Go", "en": "Hello Go", "en-GB": "Hello Go", "fr": "Hello Go"} {
		if got := m.TitleIn(lang); got != want {
			t.Errorf("TitleIn(%q) = %q, want %q", lang, got, want)
		}
	}
	if m.DescriptionIn("en") != "Showcase" || m.DescriptionIn("") != "Vitrine" {
		t.Errorf("descriptions: %q %q", m.DescriptionIn("en"), m.DescriptionIn(""))
	}
	only := Manifest{ID: "x", Title: map[string]string{"es": "Hola"}}
	if only.TitleIn("pt-BR") != "Hola" {
		t.Error("with no pt/en text, any text beats nothing")
	}
	if (Manifest{ID: "bare"}).TitleIn("en") != "bare" {
		t.Error("with no title at all, the id is shown")
	}
}

func TestInstalledRecordsRoundTripAndSkipWhatIsBroken(t *testing.T) {
	root := t.TempDir()
	apps := filepath.Join(root, "applications")
	if err := os.MkdirAll(apps, 0o755); err != nil {
		t.Fatal(err)
	}
	exe := func(name string) string {
		p := filepath.Join(apps, name+".app")
		if err := os.WriteFile(p, []byte("elf"), 0o755); err != nil {
			t.Fatal(err)
		}
		return p
	}
	m, _ := ParseManifest([]byte(helloManifest))
	other := Manifest{ID: "alpha", Title: map[string]string{"pt-BR": "A"}, Description: map[string]string{"pt-BR": "a"}}

	for _, in := range []Installed{
		{Manifest: m, Version: "1.2.3", Exe: exe("hello"), Icon: "/x/hello.bmp"},
		{Manifest: other, Version: "0.1.0", Exe: exe("alpha")},
		{Manifest: Manifest{ID: "gone", Title: m.Title, Description: m.Description}, Exe: filepath.Join(apps, "gone.app")}, // .app deleted
	} {
		if err := WriteInstalled(root, in); err != nil {
			t.Fatal(err)
		}
	}
	// junk that must not hide the good ones
	meta := filepath.Join(root, filepath.FromSlash(MetaDir))
	os.WriteFile(filepath.Join(meta, "broken.json"), []byte("{nope"), 0o644)
	os.WriteFile(filepath.Join(meta, "invalid.json"), []byte(`{"id":"UPPER"}`), 0o644)
	os.WriteFile(filepath.Join(meta, "notes.txt"), []byte("x"), 0o644)
	if err := SaveSettings(root, Settings{Lang: "en"}); err != nil { // not an app record
		t.Fatal(err)
	}

	got := ReadInstalled(root)
	var ids []string
	for _, in := range got {
		ids = append(ids, in.ID)
	}
	if !reflect.DeepEqual(ids, []string{"alpha", "hello"}) {
		t.Fatalf("installed = %v (sorted, only the valid ones with an .app)", ids)
	}
	if got[1].Version != "1.2.3" || got[1].Icon != "/x/hello.bmp" || got[1].TitleIn("en") != "Hello Go" {
		t.Errorf("record = %+v", got[1])
	}
	if err := WriteInstalled(root, Installed{Manifest: Manifest{ID: "BAD"}}); err == nil {
		t.Error("an invalid manifest must not be written")
	}
	if len(ReadInstalled(filepath.Join(root, "nowhere"))) != 0 {
		t.Error("no folder, no apps")
	}
}

func TestSettingsDefaultsAndPersistence(t *testing.T) {
	root := t.TempDir()
	if s := LoadSettings(root); s.Lang != "pt-BR" {
		t.Errorf("default language = %q", s.Lang)
	}
	if err := SaveSettings(root, Settings{Lang: "en_US"}); err != nil {
		t.Fatal(err)
	}
	if s := LoadSettings(root); s.Lang != "en-US" {
		t.Errorf("a saved tag is normalized: %q", s.Lang)
	}
	os.WriteFile(SettingsPath(root), []byte("{broken"), 0o644)
	if s := LoadSettings(root); s.Lang != "pt-BR" {
		t.Errorf("a broken file gives the defaults: %q", s.Lang)
	}
	os.WriteFile(SettingsPath(root), []byte(`{"lang": ""}`), 0o644)
	if s := LoadSettings(root); s.Lang != "pt-BR" {
		t.Errorf("an empty language gives the default: %q", s.Lang)
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

// ---- the glue ----

// fakeApp is a one-button app; keys goes to its KeyHandler.
type fakeApp struct {
	doc   *widget.Document
	keys  []int
	taken bool
}

func (f *fakeApp) Document() *widget.Document { return f.doc }
func (f *fakeApp) Key(code int) bool          { f.keys = append(f.keys, code); return f.taken }

type recorder struct {
	blits, repaints, exits int
	rects                  []image.Rectangle
}

func (r *recorder) output() output {
	return output{
		Blit:     func(*image.Gray) { r.blits++ },
		BlitRect: func(_ *image.Gray, rect image.Rectangle) { r.rects = append(r.rects, rect) },
		Repaint:  func() { r.repaints++ },
		Exit:     func() { r.exits++ },
	}
}

func newTestGlue(t *testing.T) (*glue, *fakeApp, *widget.Button, *recorder) {
	t.Helper()
	btn := widget.NewButton("Tap me")
	doc := widget.NewDocument(1404, 1872)
	doc.SetRoot(widget.NewColumn(0, btn))
	app := &fakeApp{doc: doc}
	rec := &recorder{}
	g := newGlue(Env{Width: 1404, Height: 1872}, func(Env) App { return app }, rec.output())
	g.start()
	return g, app, btn, rec
}

func TestGlueRepaintsOnlyWhatChangedAndRefreshesFullyEveryTenUpdates(t *testing.T) {
	g, _, btn, rec := newTestGlue(t)
	g.Draw() // the system asks for the first draw: everything
	if rec.blits != 1 {
		t.Fatalf("first draw must blit the whole screen: %d", rec.blits)
	}

	for i := 0; i < fullRefreshEvery; i++ {
		btn.Text = "changed " + string(rune('a'+i))
		btn.Invalidate() // our own repaint request: partial
		if rec.repaints == 0 {
			t.Fatal("a change must request a repaint")
		}
		g.Draw()
	}
	if len(rec.rects) != fullRefreshEvery || rec.blits != 1 {
		t.Fatalf("expected %d partial blits and no extra full one yet: rects=%d blits=%d", fullRefreshEvery, len(rec.rects), rec.blits)
	}
	btn.Text = "one more"
	btn.Invalidate()
	g.Draw()
	if rec.blits != 2 {
		t.Errorf("after %d partial updates the next one is a full refresh: blits=%d", fullRefreshEvery, rec.blits)
	}

	before := rec.blits
	g.Draw() // not requested by us (pending is false): the system covered the screen
	if rec.blits != before+1 {
		t.Error("a draw the system asked for must repaint everything")
	}
}

func TestGlueCoalescesRepaintRequestsAndSkipsMovesWhileBusy(t *testing.T) {
	g, _, btn, rec := newTestGlue(t)
	g.Draw()
	rec.repaints = 0
	btn.Text = "a"
	btn.Invalidate()
	btn.Text = "b"
	btn.Invalidate()
	if rec.repaints != 1 {
		t.Errorf("two changes, one queued repaint: %d", rec.repaints)
	}
	if !g.Pointer(inkview.PointerMove, 5, 5) {
		t.Error("a move while the screen catches up is swallowed")
	}
	clicks := 0
	btn.OnClick = func(*widget.Event) { clicks++ }
	g.pending = false
	c := btn.Bounds.Center()
	g.Pointer(inkview.PointerDown, int(c.X), int(c.Y))
	g.Pointer(inkview.PointerUp, int(c.X), int(c.Y))
	if clicks != 1 {
		t.Errorf("touch must reach the document: %d clicks", clicks)
	}
}

func TestGlueKeys(t *testing.T) {
	g, app, _, rec := newTestGlue(t)
	g.Draw()

	// with a modal open, left and Menu cancel it before the app sees them
	m := widget.NewModal("Hi", widget.NewLabel("x"))
	m.Open(g.doc)
	if !g.Key(inkview.KeyPrev) || g.doc.HasModal() || len(app.keys) != 0 {
		t.Fatalf("left must close the modal and not reach the app: keys=%v", app.keys)
	}
	m.Open(g.doc)
	if !g.Key(inkview.KeyMenu) || g.doc.HasModal() {
		t.Error("Menu must close the modal too")
	}

	// nothing open: the app decides
	if g.Key(inkview.KeyNext) || !reflect.DeepEqual(app.keys, []int{inkview.KeyNext}) {
		t.Errorf("an unhandled key goes to the app and is reported unhandled: %v", app.keys)
	}
	app.taken = true
	if !g.Key(inkview.KeyPrev) {
		t.Error("a key the app takes is handled")
	}

	// a real Back key leaves, but only when nothing took it
	app.taken = false
	if !g.Key(inkview.KeyBack) || rec.exits != 1 {
		t.Errorf("Back with nothing open leaves: exits=%d", rec.exits)
	}
}

func TestGlueWithAnAppThatHandlesNoKeys(t *testing.T) {
	doc := widget.NewDocument(1404, 1872)
	g := newGlue(Env{}, func(Env) App { return plainApp{doc} }, (&recorder{}).output())
	g.start()
	if g.Key(inkview.KeyNext) {
		t.Error("no KeyHandler: an unknown key is not handled")
	}
}

type plainApp struct{ doc *widget.Document }

func (p plainApp) Document() *widget.Document { return p.doc }

// ---- first run ----

// A minimal desktop config in the shape install expects.
const viewJSON = `{
    "applications": {
    },
    "groups": [
        { "title": "@General", "apps": [
        ] }
    ]
}`

func TestFirstRunRegistersTheAppAndLeavesARecordForTheLauncher(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "system", "profiles", "default", "config", "desktop")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "view.json"), []byte(viewJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	m, _ := ParseManifest([]byte(helloManifest))

	notice := firstRun(Config{}, m, root, "pt-BR")
	if !strings.Contains(notice, "Ícone registrado") {
		t.Errorf("first run says what happened, in Portuguese: %q", notice)
	}
	view, _ := os.ReadFile(filepath.Join(dir, "view.json"))
	if !strings.Contains(string(view), "U_") || !strings.Contains(string(view), "Olá Go") {
		t.Errorf("the desktop entry uses the translated title: %s", view)
	}
	if _, err := os.Stat(filepath.Join(root, "applications", "icons")); err != nil {
		t.Errorf("icons folder: %v", err)
	}

	// The record points at this test binary (os.Executable), so it is "installed".
	got := ReadInstalled(root)
	if len(got) != 1 || got[0].ID != "hello" || got[0].Version == "" || filepath.Base(got[0].Icon) != "hello.bmp" {
		t.Fatalf("record = %+v", got)
	}

	if again := firstRun(Config{}, m, root, "en"); again != "" {
		t.Errorf("a second run has nothing to say: %q", again)
	}
}

func TestFirstRunReportsFailuresInTheChosenLanguage(t *testing.T) {
	m, _ := ParseManifest([]byte(helloManifest))
	notice := firstRun(Config{}, m, t.TempDir(), "en") // no view.json under this root
	if !strings.HasPrefix(notice, "Could not register the icon:") {
		t.Errorf("notice = %q", notice)
	}
}

func TestRunRejectsABrokenManifestBeforeTouchingTheDevice(t *testing.T) {
	if err := Run(Config{Manifest: []byte(`{"id": "BAD"}`)}, nil); err == nil {
		t.Error("an invalid manifest must stop the app at start")
	}
	// A valid one reaches the stubbed InkView, which reports it is not on a device.
	err := Run(Config{Manifest: []byte(helloManifest), Root: t.TempDir()}, func(Env) App { return nil })
	if err != inkview.ErrNotOnDevice {
		t.Errorf("off-device Run = %v", err)
	}
}
