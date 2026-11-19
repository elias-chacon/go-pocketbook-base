package domain

import "strings"

// The bounds and the default of the size of an app icon, in design pixels.
const (
	MinIconSize     = 96
	MaxIconSize     = 240
	DefaultIconSize = 150
)

// Preferences are the Launcher's own settings (the language is shared by every
// app and lives in the platform's Settings).
type Preferences struct {
	Grid     Grid   `json:"grid"`
	IconSize int    `json:"iconSize"`   // edge of an app icon, in design pixels
	Image    string `json:"background"` // file name in the backgrounds folder ("" = none)
}

// DefaultPreferences are the preferences of a Launcher where nothing was chosen.
func DefaultPreferences() Preferences {
	return Preferences{Grid: DefaultGrid, IconSize: DefaultIconSize}
}

// Normalized returns the preferences with every value inside its bounds: whatever
// was read from disk becomes something the screen can show.
func (p Preferences) Normalized() Preferences {
	p.Grid = p.Grid.Normalized()
	switch {
	case p.IconSize == 0:
		p.IconSize = DefaultIconSize
	case p.IconSize < MinIconSize:
		p.IconSize = MinIconSize
	case p.IconSize > MaxIconSize:
		p.IconSize = MaxIconSize
	}
	p.Image = strings.TrimSpace(p.Image)
	if strings.ContainsAny(p.Image, "/\\") || p.Image == ".." { // a file name, never a path
		p.Image = ""
	}
	return p
}

// PreferencesStore is the port that keeps the preferences.
type PreferencesStore interface {
	// Load returns the saved preferences (the defaults when there are none).
	Load() Preferences
	Save(p Preferences) error
}
