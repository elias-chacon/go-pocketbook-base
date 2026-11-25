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
	"image"
	"os"
	"path"
	"path/filepath"
	"time"

	"go-pocketbook-base/internal/kernel/version"
	"go-pocketbook-base/internal/platform/application"
	"go-pocketbook-base/internal/platform/domain"
	"go-pocketbook-base/internal/platform/infrastructure/jsonfiles"
	"go-pocketbook-base/internal/platform/infrastructure/process"
	"go-pocketbook-base/internal/platform/infrastructure/reader"
	"go-pocketbook-base/internal/platform/infrastructure/selfupdate"
	"go-pocketbook-base/internal/toolkit/widget"
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
	Intent       = domain.Intent
	// Panel, Raw and the update modes are for an app that tests the screen (the engineering app).
	Panel      = application.Panel
	PanelInfo  = application.PanelInfo
	Raw        = application.Raw
	UpdateMode = application.UpdateMode
	PaintStats = application.PaintStats
)

// The update modes of the panel, from the default one to the fast ones.
const (
	UpdatePartial = application.UpdatePartial
	UpdateDynamic = application.UpdateDynamic
	UpdateA2      = application.UpdateA2
	UpdateDU4     = application.UpdateDU4
	UpdateBW      = application.UpdateBW
	UpdateHQ      = application.UpdateHQ
	UpdateFull    = application.UpdateFull
	UpdateModes   = application.UpdateModes
)

// The keys an app may handle in its KeyHandler.
const (
	KeyOther    = application.KeyOther
	KeyBack     = application.KeyBack
	KeyPrevious = application.KeyPrevious
	KeyNext     = application.KeyNext
	KeyMenu     = application.KeyMenu
)

// httpTimeout bounds every request of Env.HTTP: a download of a few megabytes over Wi-Fi.
const httpTimeout = 60 * time.Second

// The repository whose latest release holds the published build of every app: the Launcher lists
// the apps from it and every app updates itself from it.
const (
	RepoOwner = "elias-chacon"
	RepoName  = "go-pocketbook-base"
)

// checkTimeout bounds the background look for an update (an offline reader answers at once).
const checkTimeout = 15 * time.Second

// DefaultRoot is the user storage of the reader (what RegisterOnDrive takes on the reader itself).
const DefaultRoot = application.DefaultRoot

// DeviceModel is the model the reader reports ("PocketBook 741"...; empty on the desktop).
func DeviceModel() string { return reader.DeviceModel() }

// DiskSpace is the total and the free bytes of the volume that holds path (ok false when unknown).
func DiskSpace(path string) (total, free uint64, ok bool) {
	return reader.DiskSpace(path)
}

// Logf writes a trace line to the app's log (applications/<id>.log). Nothing is
// written unless verbose mode is on: create the file applications/.pbgo/debug on
// the reader (or set PBGO_DEBUG=1) and the log traces every event and step.
func Logf(format string, args ...interface{}) { reader.Logf(format, args...) }

// Errorf writes what went wrong to the same log, always. Never pass it a
// password, a token or the text of a message.
func Errorf(format string, args ...interface{}) { reader.Errorf(format, args...) }

// verboseRequested reports whether the trace was asked for: the marker file
// applications/.pbgo/debug exists on the reader, or PBGO_DEBUG is set.
func verboseRequested(root string) bool {
	if os.Getenv("PBGO_DEBUG") != "" {
		return true
	}
	_, err := os.Stat(filepath.Join(root, "applications", ".pbgo", "debug"))
	return err == nil
}

// The errors of Env.Send.
var (
	ErrNoSuchApp  = domain.ErrNoSuchApp
	ErrNotHandled = domain.ErrNotHandled
)

// ParseManifest decodes and validates an app.json.
func ParseManifest(data []byte) (Manifest, error) { return domain.ParseManifest(data) }

// IconFromImage makes the launcher icon of an app from a picture (its icon.png).
func IconFromImage(img image.Image) domain.IconRenderer { return application.IconFromImage(img) }

// IconFromPNG is IconFromImage for the bytes of an icon.png.
func IconFromPNG(data []byte) (domain.IconRenderer, error) { return application.IconFromPNG(data) }

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
	reader.ResetLog()
	reader.SetVerbose(verboseRequested(root))
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

	intent, err := domain.DecodeIntent(os.Getenv(domain.IntentEnv))
	if err != nil {
		reader.Errorf("%v", err) // a broken intent is ignored: the app starts as if opened from the home screen
		intent = nil
	}
	env := Env{
		Manifest: m, Lang: lang, Version: version.Value, Notice: notice, Root: root,
		Settings: settings, Apps: apps, HTTP: reader.HTTPClient(httpTimeout), Intent: intent,
		Send: application.Intents{
			Apps: apps, SelfID: m.ID, SelfExe: exe,
			Exec: func(path, encoded string) error {
				return process.Exec(path, process.WithEnv(domain.IntentEnv, encoded))
			},
		}.Send,
	}
	reply := ""
	if intent != nil {
		reply = intent.ReplyTo
	}
	client := env.HTTP
	quick := *client
	quick.Timeout = checkTimeout
	updater := application.Updater{
		ID: m.ID, Current: version.Value,
		Source:  selfupdate.Source{Client: &quick, Owner: RepoOwner, Repo: RepoName},
		Install: selfupdate.Installer{Client: client, Exe: exe},
		Installed: func() (string, error) {
			data, err := os.ReadFile(exe)
			return selfupdate.Sum(data), err
		},
		State: jsonfiles.NewUpdateState(root, m.ID),
		// The new build starts the way this one was started (same intent, same way back).
		Restart: func() error { return process.Exec(exe, os.Environ()) },
	}
	screen := &reader.Screen{}
	// InkView ends the process when the app closes, so going back to the Launcher
	// has to happen as the app asks to exit, not after the event loop.
	screen.BeforeClose = func() {
		returnToLauncher(exe, true, os.Getenv(domain.LauncherEnv), reply, process.Exec)
	}
	err = reader.Run(application.NewRuntime(env, factory, screen).WithUpdates(updater))
	reader.Logf("Run returned err=%v", err)
	return err
}

// returnToLauncher goes back when the app asks to exit: to the app that opened it
// with an intent (reply), or else to the Launcher when the Launcher started it (see
// domain.ReplyTarget). The intent is taken out of the environment, so the app it
// returns to does not take it for a request of its own. A failure is only logged:
// the app simply ends, as it would from the home screen.
func returnToLauncher(self string, askedToExit bool, launcher, reply string, exec func(path string, env []string) error) {
	target := domain.ReplyTarget(reply, launcher, self, askedToExit)
	if target == "" {
		return
	}
	reader.Logf("returning to: %s", target)
	if err := exec(target, process.WithoutEnv(domain.IntentEnv)); err != nil {
		reader.Errorf("could not go back: %v", err)
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

// Chrome is the standard top bar every app puts above its screens: the app name with
// About and Exit, in the language of the reader (see widget.AppChrome).
func Chrome(env Env) widget.AppChrome { return application.Chrome(env) }

// PruneHomeScreen takes out of the home screen of the reader whose storage is root the entries of
// apps whose program is no longer there (an app removed by hand leaves its icon behind). It returns
// the ids it took out. The home screen shows the change when the cable is next unplugged.
func PruneHomeScreen(root string) ([]string, error) {
	return reader.DesktopRegistrar{Root: root}.RemoveOrphans()
}

// Uninstall removes an app and everything it left on the reader (see application.Uninstall):
// the home screen entry and icons, the program, its record and log, the data files its manifest
// lists and what eraseData drops from databases (nil: nothing).
func Uninstall(root, appID string, eraseData func(appID string) error) error {
	return application.Uninstall{
		Apps: jsonfiles.NewApps(root), Desktop: reader.DesktopRegistrar{Root: root},
		Files: jsonfiles.NewFiles(root), EraseData: eraseData,
	}.Run(appID)
}
