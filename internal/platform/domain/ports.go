package domain

import "image"

// The ports: what the domain needs from the outside world, stated in the
// domain's own words. The infrastructure layer implements them; the application
// layer uses them. Nothing here knows how they are done.

// AppRegistry is the collection of apps known to be installed on this reader.
type AppRegistry interface {
	// Save records (or updates) an installed app.
	Save(app InstalledApp) error
	// List returns the installed apps, sorted by id. An app whose file is gone is
	// not listed, and one broken record never hides the others.
	List() []InstalledApp
}

// SettingsStore keeps the shared Settings.
type SettingsStore interface {
	// Load never fails: a missing or broken store gives the defaults.
	Load() Settings
	Save(s Settings) error
}

// IconRenderer draws an app's launcher icon (size x size). focused is the
// inverted variant shown while the icon is tapped.
type IconRenderer func(size int, focused bool) image.Image

// Registration is what it takes to put an app on the reader's home screen.
type Registration struct {
	Exe   string       // absolute path of <id>.app
	Title string       // name shown under the icon, in the current language
	Group string       // desktop group
	Icon  IconRenderer // draws the icons the registrar must write
}

// DesktopRegistrar puts an app on the reader's home screen. It is idempotent:
// registering an app that is already there changes nothing.
type DesktopRegistrar interface {
	// Register reports whether the home screen configuration was modified.
	Register(r Registration) (changed bool, err error)
}

// DesktopRemover takes an app off the reader's home screen: its entry and its icons.
type DesktopRemover interface {
	// Unregister reports whether the home screen configuration was modified.
	Unregister(appID string) (changed bool, err error)
}

// AppFiles deletes the files of an app: its program, its record, its log and the given files of
// the apps' data folder (a file that does not exist is not a failure).
type AppFiles interface {
	Remove(appID string, dataFiles []string) error
}
