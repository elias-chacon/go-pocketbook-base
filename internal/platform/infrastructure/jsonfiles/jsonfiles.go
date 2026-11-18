// Package jsonfiles implements the platform's storage ports with small JSON files
// under <root>/applications/.pbgo: one record per installed app and one file of
// shared settings. Writes are atomic (a temporary file, then a rename).
package jsonfiles

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"go-pocketbook-base/internal/platform/domain"
)

// MetaDir is where apps leave what the Launcher needs to know about them,
// relative to the storage root (a hidden folder inside applications/).
const MetaDir = "applications/.pbgo"

// settingsFile is not an app record: Apps.List skips it.
const settingsFile = "settings.json"

func metaDir(root string) string { return filepath.Join(root, filepath.FromSlash(MetaDir)) }

// RecordPath is where the record of app id lives under root.
func RecordPath(root, id string) string { return filepath.Join(metaDir(root), id+".json") }

// SettingsPath is the settings file under root.
func SettingsPath(root string) string { return filepath.Join(metaDir(root), settingsFile) }

// Apps is the domain.AppRegistry kept in JSON files.
type Apps struct{ root string }

// NewApps creates the registry of the reader whose storage is root.
func NewApps(root string) *Apps { return &Apps{root: root} }

// Save implements domain.AppRegistry.
func (a *Apps) Save(app domain.InstalledApp) error {
	if err := app.Validate(); err != nil {
		return err
	}
	return writeJSON(RecordPath(a.root, app.ID), app)
}

// List implements domain.AppRegistry: sorted by id; an app whose executable is
// gone (the .app file was deleted) is skipped, and so is anything invalid, so one
// broken file never hides the other apps.
func (a *Apps) List() []domain.InstalledApp {
	files, err := filepath.Glob(filepath.Join(metaDir(a.root), "*.json"))
	if err != nil {
		return nil
	}
	sort.Strings(files)
	var out []domain.InstalledApp
	for _, f := range files {
		if filepath.Base(f) == settingsFile {
			continue
		}
		if app, ok := readRecord(f); ok {
			out = append(out, app)
		}
	}
	return out
}

func readRecord(path string) (domain.InstalledApp, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.InstalledApp{}, false
	}
	var app domain.InstalledApp
	if json.Unmarshal(data, &app) != nil || app.Validate() != nil {
		return domain.InstalledApp{}, false
	}
	if _, err := os.Stat(app.Exe); err != nil {
		return domain.InstalledApp{}, false
	}
	return app, true
}

// Settings is the domain.SettingsStore kept in a JSON file.
type Settings struct{ root string }

// NewSettings creates the settings store of the reader whose storage is root.
func NewSettings(root string) *Settings { return &Settings{root: root} }

// Load implements domain.SettingsStore: a missing or broken file gives the
// defaults (it is only preferences: never a reason not to start).
func (s *Settings) Load() domain.Settings {
	data, err := os.ReadFile(SettingsPath(s.root))
	if err != nil {
		return domain.DefaultSettings()
	}
	var read domain.Settings
	if json.Unmarshal(data, &read) != nil {
		return domain.DefaultSettings()
	}
	return read.Normalized()
}

// Save implements domain.SettingsStore.
func (s *Settings) Save(v domain.Settings) error {
	return writeJSON(SettingsPath(s.root), v)
}

// writeJSON writes v to path atomically, creating the folder.
func writeJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
