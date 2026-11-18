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
	"os"
	"path/filepath"

	"go-pocketbook-base/internal/kernel/version"
	"go-pocketbook-base/internal/platform/application"
	"go-pocketbook-base/internal/platform/domain"
	"go-pocketbook-base/internal/platform/infrastructure/jsonfiles"
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
	err = reader.Run(application.NewRuntime(env, factory, reader.Screen{}))
	reader.Logf("Run returned err=%v", err)
	return err
}
