// Package prefsfile keeps the Launcher's preferences in a JSON file
// (applications/.pbgo/launcher.json): the domain.PreferencesStore of the reader.
package prefsfile

import (
	"encoding/json"
	"os"
	"path/filepath"

	"go-pocketbook-base/internal/launcher/domain"
)

// Store implements domain.PreferencesStore.
type Store struct{ Path string }

// New returns the store of the reader whose storage is root.
func New(root string) Store {
	return Store{Path: filepath.Join(root, "applications", ".pbgo", "launcher.json")}
}

// Load returns the saved preferences; a missing or broken file gives the defaults.
func (s Store) Load() domain.Preferences {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		return domain.DefaultPreferences()
	}
	var p domain.Preferences
	if json.Unmarshal(data, &p) != nil {
		return domain.DefaultPreferences()
	}
	return p.Normalized()
}

// Save writes the preferences (normalized) through a temporary file.
func (s Store) Save(p domain.Preferences) error {
	data, err := json.MarshalIndent(p.Normalized(), "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.Path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
