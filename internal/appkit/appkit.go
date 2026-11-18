package appkit

import (
	"embed"
	"image"
	"os"
	"path/filepath"

	"go-pocketbook-base/internal/i18n"
	"go-pocketbook-base/internal/inkview"
	"go-pocketbook-base/internal/install"
	"go-pocketbook-base/internal/version"
	"go-pocketbook-base/internal/widget"
)

//go:embed locales/*.json
var kitLocales embed.FS

// DefaultRoot is the user storage of the reader.
const DefaultRoot = "/mnt/ext1"

// fullRefreshEvery forces a full (flashing) panel refresh after this many
// partial ones, to clear the ghosting that fast e-ink updates leave behind.
const fullRefreshEvery = 10

// Config is what an app tells the kit about itself.
type Config struct {
	// Manifest is the app's app.json (embed it with go:embed).
	Manifest []byte
	// Icon draws the launcher icon; nil uses DefaultIcon(title).
	Icon func(size int, focused bool) image.Image
	// Root is the storage root (DefaultRoot when empty; tests use a temp folder).
	Root string
}

// Env is what the kit gives the app when it starts.
type Env struct {
	Width, Height int             // the screen, in device pixels
	Keyboard      widget.Keyboard // the system keyboard
	Exit          func()          // leaves the app
	Notice        string          // first-run message ("icon registered..."), already translated
	Manifest      Manifest
	Lang          string // language from the shared settings (set in the Launcher)
	Version       string // MAJOR.MINOR.BUILD of this build
	Root          string
}

// App is what an app gives the kit: the screen.
type App interface {
	// Document is the element tree to show. The kit wires its repaint.
	Document() *widget.Document
}

// KeyHandler is optionally implemented by an App to handle hardware keys the
// kit does not: return true when the key was handled.
type KeyHandler interface{ Key(code int) bool }

// Factory builds the app once the screen size is known.
type Factory func(env Env) App

// Run is the whole main() of an app: it registers the app on first run, then
// runs the InkView event loop and blocks until the app exits.
func Run(cfg Config, factory Factory) error {
	m, err := ParseManifest(cfg.Manifest)
	if err != nil {
		return err
	}
	root := cfg.Root
	if root == "" {
		root = DefaultRoot
	}
	settings := LoadSettings(root)

	// Event log for on-device debugging.
	inkview.DebugLog = filepath.Join(root, "applications", m.ID+".log")
	inkview.Logf("main start id=%s version=%s lang=%s args=%q", m.ID, version.Value, settings.Lang, os.Args)

	env := Env{
		Manifest: m, Lang: settings.Lang, Version: version.Value, Root: root,
		Notice: firstRun(cfg, m, root, settings.Lang),
	}
	err = inkview.Run(newGlue(env, factory, deviceOutput()))
	inkview.Logf("Run returned err=%v", err)
	return err
}

// kitTranslator is the kit's own messages (the first-run notice).
func kitTranslator(lang string) *i18n.Bundle {
	tr := i18n.New()
	if err := tr.AddFS(kitLocales, "locales"); err != nil {
		panic(err) // embedded files: an error is a programming mistake
	}
	tr.SetLang(lang)
	return tr
}

// firstRun registers the launcher icon on the desktop (PocketBook has no
// installer: copying the .app is the installation, the app finishes the job the
// first time it runs) and records itself for the Launcher. It returns the
// message to show, or "" when there is nothing to say.
func firstRun(cfg Config, m Manifest, root, lang string) string {
	tr := kitTranslator(lang)
	exe, err := os.Executable()
	if err != nil {
		exe = filepath.Join(root, "applications", m.ID+".app")
	}
	icon := cfg.Icon
	if icon == nil {
		icon = DefaultIcon(m.TitleIn(lang))
	}
	res, err := install.Register(install.Config{
		Root: root, Exe: exe, Title: m.TitleIn(lang), Group: m.DesktopGroup(), Icon: icon,
	})
	inkview.Logf("install: %+v err=%v", res, err)
	if err != nil {
		return tr.T("install.failed", err.Error())
	}

	rec := Installed{
		Manifest: m, Version: version.Value, Exe: exe,
		Icon: filepath.Join(root, "applications", "icons", m.ID+".bmp"),
	}
	if err := WriteInstalled(root, rec); err != nil {
		inkview.Logf("record: %v", err)
	}
	if res.Changed {
		return tr.T("install.registered")
	}
	return ""
}
