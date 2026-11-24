// Package application holds the use cases of the platform context: running an
// app (the lifecycle between the reader's events and the app's document) and
// registering it the first time. It orchestrates the domain and the UI toolkit
// and reaches the reader and the disk only through ports, so everything here is
// testable on the PC.
package application

import (
	"net/http"
	"time"

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
	// Root is the storage root of the reader (DefaultRoot; a temporary folder in tests).
	Root string
	// Settings is the store of the preferences shared by every app: save the
	// language here when the user changes it.
	Settings domain.SettingsStore
	// Apps lists the apps installed on this reader.
	Apps domain.AppRegistry
	// Flush paints the pending changes right now. An app that is about to block
	// (a download) calls it first so the screen says why; normally the platform
	// repaints by itself once the handler returns.
	Flush func()
	// Intent is the request this app was opened with by another app (nil when it
	// was opened from the home screen or the Launcher). Read it when starting.
	Intent *domain.Intent
	// Send opens another app with an action and some data (see Intents); on the
	// reader it does not return when it works, because the other app takes over.
	Send func(appID, action string, data map[string]string) error
	// After runs f on the event loop once d has passed, never in the middle of a touch or a
	// repaint, so f may change the widgets. An app does a long job (a download) in small steps,
	// each one scheduled by the one before, and the reader keeps answering touches in between.
	After func(d time.Duration, f func())
	// Panel is the reader's display beyond the repaint the platform does by itself: the update
	// modes, color and the panel's own details. Only the engineering app uses it. Nil in tests.
	Panel Panel
	// Watch registers f to be called after every repaint the platform does, with what it cost
	// (step by step). Only the engineering app uses it.
	Watch func(f func(PaintStats))
	// HTTP is the client to use for the network: it works on the reader's Linux
	// (no resolv.conf, no IPv6), which the standard one does not. Nil in tests.
	HTTP *http.Client
}

// PaintStats is what one repaint cost: the document's own steps (layout, snapshot, diff and paint)
// and the time to hand the areas to the panel.
type PaintStats struct {
	widget.UpdateStats
	Show  time.Duration // Screen.Show / ShowRect: the blit and the update call
	Areas int           // how many areas went to the panel
}

// App is what an app gives the platform: the screen.
type App interface {
	// Document is the element tree to show. The platform wires its repaint.
	Document() *widget.Document
}

// KeyHandler is optionally implemented by an App to handle hardware keys the
// platform does not: return true when the key was handled.
type KeyHandler interface{ Key(k Key) bool }

// Raw is optionally implemented by an App that sometimes draws the panel itself (the screen tests
// of the engineering app): when Raw returns true it painted, and the platform skips its own paint.
type Raw interface{ Raw(p Panel) bool }

// UpdateMode is how a part of the panel is refreshed after it was drawn.
type UpdateMode int

// The update modes, from the default one to the fast ones that trade quality for speed.
const (
	UpdatePartial UpdateMode = iota
	UpdateDynamic
	UpdateA2
	UpdateDU4
	UpdateBW
	UpdateHQ
	UpdateFull
)

// UpdateModes is how many update modes there are.
const UpdateModes = int(UpdateFull) + 1

// PanelInfo is what the reader says about its panel.
type PanelInfo struct {
	Model    string // "PocketBook 741"
	Depth    int    // bits per pixel of the framebuffer: 8 gray, 24 color
	Waveform string // the waveform file
	Canvas   string // the canvas, as InkView describes it
}

// Panel is the reader's display, as an app that tests the screen needs it. The infrastructure
// implements it with InkView.
type Panel interface {
	Info() PanelInfo
	// DrawGray draws the part r of a grayscale image and refreshes it with the mode.
	DrawGray(img *image.Gray, r image.Rectangle, m UpdateMode)
	// DrawColor draws the part r of a color image and refreshes it with the mode.
	DrawColor(img *image.RGBA, r image.Rectangle, m UpdateMode)
	// SetMode is the mode the platform's own repaints use from now on.
	SetMode(m UpdateMode)
	// Wait blocks until the panel finished the updates asked for.
	Wait()
	// Settle leaves the fast (A2) state with a clean update and waits for the panel.
	Settle()
}

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
	// After runs f on the event loop once d has passed (see Env.After).
	After(d time.Duration, f func())
	// Close ends the app.
	Close()
}
