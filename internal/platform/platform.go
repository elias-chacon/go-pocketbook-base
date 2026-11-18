// Package platform is the contract every app of this project follows (the
// Launcher and the plugins in ./apps, with apps/hello as the blueprint) and the
// one package an app imports for it.
//
// It is the context's composition root: it wires the layers together (the
// domain's ports implemented by the infrastructure, driven by the application's
// use cases) and re-exports what an app needs under one roof. The layers
// themselves:
//
//	domain          what an app is: Manifest, InstalledApp, Settings, the ports
//	application     running an app (Runtime) and registering it (FirstRun)
//	infrastructure  JSON files (jsonfiles) and the e-reader itself (reader)
//
// An app is one folder, apps/<id>/, with a main.go, an app.json manifest and its
// locales, built into the single file <id>.app that is copied to the reader's
// applications folder. Nothing else has to be installed.
package platform

import (
	"errors"
	"os"
	"path"
	"path/filepath"

	"go-pocketbook-base/internal/kernel/version"
	"go-pocketbook-base/internal/platform/application"
	"go-pocketbook-base/internal/platform/domain"
	"go-pocketbook-base/internal/platform/infrastructure/jsonfiles"
	"go-pocketbook-base/internal/platform/infrastructure/process"
	"go-pocketbook-base/internal/platform/infrastructure/reader"
)

// What an app uses, under one name each.
type (
	Config       = application.Config
	Env          = application.Env
	App          = application.App
	KeyHandler   = application.KeyHandler
	Factory      = application.Factory
	Key          = application.Key
	Manifest     = domain.Manifest
	InstalledApp = domain.InstalledApp
	Settings     = domain.Settings
)

// The keys an app may handle in its KeyHandler.
const (
	KeyOther    = application.KeyOther
	KeyBack     = application.KeyBack
	KeyPrevious = application.KeyPrevious
	KeyNext     = application.KeyNext
	KeyMenu     = application.KeyMenu
)

// DefaultRoot is the user storage of the reader (what RegisterOnDrive takes on the reader itself).
const DefaultRoot = application.DefaultRoot

// ParseManifest decodes and validates an app.json.
func ParseManifest(data []byte) (Manifest, error) { return domain.ParseManifest(data) }

// DefaultIcon is the launcher icon of an app that has none of its own.
func DefaultIcon(title string) domain.IconRenderer { return application.DefaultIcon(title) }

// Run is the whole main() of an app: it registers the app on first run, then
// runs the InkView event loop and blocks until the app exits.
func Run(cfg Config, factory Factory) error {
	m, err := domain.ParseManifest(cfg.Manifest)
	if err != nil {
		return err
	}
	root := cfg.Root
	if root == "" {
		root = application.DefaultRoot
	}
	settings, apps := jsonfiles.NewSettings(root), jsonfiles.NewApps(root)
	lang := settings.Load().Lang

	// Event log for on-device debugging.
	reader.SetLogFile(filepath.Join(root, "applications", m.ID+".log"))
	reader.Logf("main start id=%s version=%s lang=%s args=%q", m.ID, version.Value, lang, os.Args)

	exe, err := os.Executable()
	if err != nil {
		exe = filepath.Join(root, "applications", m.ID+".app")
	}
	notice := application.FirstRun{
		Registrar: reader.DesktopRegistrar{Root: root}, Apps: apps, Logf: reader.Logf,
	}.Execute(application.FirstRunInput{
		Manifest: m, Lang: lang, Version: version.Value, Exe: exe,
		IconPath: reader.IconPath(root, m.ID), Icon: cfg.Icon,
	})

	env := Env{
		Manifest: m, Lang: lang, Version: version.Value, Notice: notice,
		Settings: settings, Apps: apps,
	}
	screen := &reader.Screen{}
	// InkView ends the process when the app closes, so going back to the Launcher
	// has to happen as the app asks to exit, not after the event loop.
	screen.BeforeClose = func() {
		returnToLauncher(exe, true, os.Getenv(domain.LauncherEnv), process.Exec)
	}
	err = reader.Run(application.NewRuntime(env, factory, screen))
	reader.Logf("Run returned err=%v", err)
	return err
}

// returnToLauncher starts the Launcher in place of this app when the Launcher
// started it and the app asked to exit (see domain.ReturnTarget). A failure is
// only logged: the app simply ends, as it would from the home screen.
func returnToLauncher(self string, askedToExit bool, launcher string, exec func(path string, env []string) error) {
	target := domain.ReturnTarget(launcher, self, askedToExit)
	if target == "" {
		return
	}
	reader.Logf("returning to the launcher: %s", target)
	if err := exec(target, os.Environ()); err != nil {
		reader.Logf("could not return to the launcher: %v", err)
	}
}

// RegisterOnDrive installs an app on a reader mounted as a drive (USB), from the
// PC: it does what the app does the first time it runs (icons, the home-screen
// entry in view.json, the record for the Launcher), so the icon is there as soon
// as the cable is unplugged, without opening the app first. drive is the drive
// root (e.g. "F:\"); the app itself must already be copied to its applications
// folder. A nil icon draws the default one. It returns whether view.json changed.
func RegisterOnDrive(drive string, manifest []byte, icon domain.IconRenderer, version string) (bool, error) {
	m, err := domain.ParseManifest(manifest)
	if err != nil {
		return false, err
	}
	// The paths recorded for the reader are the reader's, not the PC's.
	exe := path.Join(application.DefaultRoot, "applications", m.ID+".app")
	changed := false
	notice := application.FirstRun{
		Registrar: registrarFunc(func(r domain.Registration) (bool, error) {
			c, err := reader.DesktopRegistrar{Root: drive}.Register(r)
			changed = c
			return c, err
		}),
		Apps: jsonfiles.NewApps(drive),
	}.Execute(application.FirstRunInput{
		Manifest: m, Lang: jsonfiles.NewSettings(drive).Load().Lang, Version: version, Exe: exe,
		IconPath: path.Join(application.DefaultRoot, "applications", "icons", m.ID+".bmp"), Icon: icon,
	})
	if notice != "" && !changed {
		return false, errors.New(notice)
	}
	return changed, nil
}

type registrarFunc func(domain.Registration) (bool, error)

func (f registrarFunc) Register(r domain.Registration) (bool, error) { return f(r) }
