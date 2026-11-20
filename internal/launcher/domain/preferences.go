package domain

import (
	"sort"
	"strings"
)

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
	// HiddenApps are the ids of the apps kept off the home screen (installed or not).
	HiddenApps []string `json:"hidden,omitempty"`
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
	p.HiddenApps = cleanIDs(p.HiddenApps)
	return p
}

// cleanIDs drops blanks and repeats and sorts the ids.
func cleanIDs(ids []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

// WithHidden returns the preferences with app id hidden (or shown again).
func (p Preferences) WithHidden(id string, hidden bool) Preferences {
	var ids []string
	for _, h := range p.HiddenApps {
		if h != id {
			ids = append(ids, h)
		}
	}
	if hidden {
		ids = append(ids, id)
	}
	p.HiddenApps = cleanIDs(ids)
	return p
}

// PreferencesStore is the port that keeps the preferences.
type PreferencesStore interface {
	// Load returns the saved preferences (the defaults when there are none).
	Load() Preferences
	Save(p Preferences) error
}
