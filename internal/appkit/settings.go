package appkit

import (
	"encoding/json"
	"os"
	"path/filepath"

	"go-pocketbook-base/internal/i18n"
)

// Settings are the preferences shared by every app, kept in
// applications/.pbgo/settings.json. The Launcher edits them; the apps read them
// when they start (so the language chosen in the Launcher is the language of all).
type Settings struct {
	// Lang is the language tag ("pt-BR", "en"); empty means the primary language.
	Lang string `json:"lang"`
}

// SettingsPath is the file under root.
func SettingsPath(root string) string {
	return filepath.Join(root, filepath.FromSlash(MetaDir), settingsFile)
}

// LoadSettings reads the settings under root; a missing or broken file gives the
// defaults (it is only preferences: never a reason not to start).
func LoadSettings(root string) Settings {
	s := Settings{Lang: i18n.Primary}
	data, err := os.ReadFile(SettingsPath(root))
	if err != nil {
		return s
	}
	var read Settings
	if json.Unmarshal(data, &read) != nil {
		return s
	}
	if read.Lang != "" {
		s.Lang = i18n.Normalize(read.Lang)
	}
	return s
}

// SaveSettings writes the settings atomically.
func SaveSettings(root string, s Settings) error {
	return writeJSON(SettingsPath(root), s)
}
