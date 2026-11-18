// Package application holds the use cases of the platform context: running an
// app (the lifecycle between the reader's events and the app's document) and
// registering it the first time. It orchestrates the domain and the UI toolkit
// and reaches the reader and the disk only through ports, so everything here is
// testable on the PC.
package application

import (
	"image"

	"go-pocketbook-base/internal/platform/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

// DefaultRoot is the user storage of the reader.
const DefaultRoot = "/mnt/ext1"

// Config is what an app tells the platform about itself.
type Config struct {
	// Manifest is the app's app.json (embed it with go:embed).
	Manifest []byte
	// Icon draws the launcher icon; nil uses DefaultIcon(title).
	Icon domain.IconRenderer
	// Root is the storage root (DefaultRoot when empty; tests use a temp folder).
	Root string
}

// Env is what the platform gives the app when it starts.
type Env struct {
	Width, Height int             // the screen, in device pixels
	Keyboard      widget.Keyboard // the system keyboard
	Exit          func()          // leaves the app
	Notice        string          // first-run message ("icon registered..."), already translated
	Manifest      domain.Manifest
	Lang          string // language from the shared settings (set in the Launcher)
	Version       string // MAJOR.MINOR.BUILD of this build
	// Settings is the store of the preferences shared by every app: save the
	// language here when the user changes it.
	Settings domain.SettingsStore
	// Apps lists the apps installed on this reader.
	Apps domain.AppRegistry
	// Flush paints the pending changes right now. An app that is about to block
	// (a download) calls it first so the screen says why; normally the platform
	// repaints by itself once the handler returns.
	Flush func()
}

// App is what an app gives the platform: the screen.
type App interface {
	// Document is the element tree to show. The platform wires its repaint.
	Document() *widget.Document
}

// KeyHandler is optionally implemented by an App to handle hardware keys the
// platform does not: return true when the key was handled.
type KeyHandler interface{ Key(k Key) bool }

// Factory builds the app once the screen size is known.
type Factory func(env Env) App

// Key is a hardware key, named by what it does and not by its InkView code.
type Key int

// The keys the platform knows. The PB741 has Home (the system closes the app, it
// never reaches the app), left, right and Menu, and no Back key.
const (
	KeyOther    Key = iota // any other key
	KeyBack                // a real Back key (not on the PB741)
	KeyPrevious            // previous page: the left key
	KeyNext                // next page: the right key
	KeyMenu
)

// IsBack reports whether the key acts as "Back": the PB741 has no Back key, so
// the left and Menu keys stand in for it.
func (k Key) IsBack() bool { return k == KeyBack || k == KeyPrevious || k == KeyMenu }

// Screen is the reader's panel, as the runtime needs it. The infrastructure
// implements it with InkView; tests with a recorder.
type Screen interface {
	// Show draws the whole image and refreshes the whole panel.
	Show(img *image.Gray)
	// ShowRect draws only r (a part of img) and refreshes just that area.
	ShowRect(img *image.Gray, r image.Rectangle)
	// RequestRepaint asks for a call to Runtime.Draw.
	RequestRepaint()
	// Close ends the app.
	Close()
}
