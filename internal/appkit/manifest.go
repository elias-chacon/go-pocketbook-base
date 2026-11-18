// Package appkit is the contract every app of this project follows (the Launcher
// and the plugins in ./apps, with apps/hello as the blueprint): the device
// lifecycle (touch, keys, partial repaint, system keyboard), the first-run
// self-registration and the manifest the Launcher reads.
//
// An app is one folder, apps/<id>/, with a main.go, an app.json manifest and its
// locales, built into the single file <id>.app that is copied to the reader's
// applications folder. Nothing else has to be installed.
package appkit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"go-pocketbook-base/internal/i18n"
)

// MetaDir is where apps leave what the Launcher needs to know about them,
// relative to the storage root (a hidden folder inside applications/).
const MetaDir = "applications/.pbgo"

// settingsFile is not an app record: ReadInstalled skips it.
const settingsFile = "settings.json"

// DefaultGroup is the desktop group an app's icon is listed in.
const DefaultGroup = "@General"

var idPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

// Manifest is the app.json of an app: who it is, in every language it speaks.
type Manifest struct {
	// ID is the folder name under apps/ and the name of the file: <id>.app.
	ID string `json:"id"`
	// Title and Description are texts by language; pt-BR is required (it is the
	// primary language) and en is the fallback.
	Title       map[string]string `json:"title"`
	Description map[string]string `json:"description"`
	// Group is the desktop group of the icon (default "@General").
	Group string `json:"group,omitempty"`
}

// ParseManifest decodes and validates an app.json.
func ParseManifest(data []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("appkit: app.json: %w", err)
	}
	if err := m.Validate(); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

// Validate checks the rules of the contract.
func (m Manifest) Validate() error {
	var problems []string
	if !idPattern.MatchString(m.ID) {
		problems = append(problems, fmt.Sprintf("id %q must be lower-case letters, digits, - or _, starting with a letter (it names the file <id>.app)", m.ID))
	}
	for _, field := range []struct {
		name string
		m    map[string]string
	}{{"title", m.Title}, {"description", m.Description}} {
		if strings.TrimSpace(field.m[i18n.Primary]) == "" {
			problems = append(problems, fmt.Sprintf("%s needs a %s text (the primary language)", field.name, i18n.Primary))
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("appkit: invalid manifest: %s", strings.Join(problems, "; "))
	}
	return nil
}

// pick chooses the text of lang from a by-language map: the language itself, its
// base language, English, then Portuguese, then anything.
func pick(texts map[string]string, lang string) string {
	lang = i18n.Normalize(lang)
	if lang == "" {
		lang = i18n.Primary // no preference: the primary language
	}
	base := lang
	if i := strings.IndexByte(lang, '-'); i >= 0 {
		base = lang[:i]
	}
	for _, l := range []string{lang, base, i18n.Fallback, i18n.Primary} {
		if s := strings.TrimSpace(texts[l]); s != "" {
			return s
		}
	}
	keys := make([]string, 0, len(texts))
	for k := range texts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if s := strings.TrimSpace(texts[k]); s != "" {
			return s
		}
	}
	return ""
}

// TitleIn is the title in lang (falling back like i18n does).
func (m Manifest) TitleIn(lang string) string {
	if t := pick(m.Title, lang); t != "" {
		return t
	}
	return m.ID
}

// DescriptionIn is the description in lang.
func (m Manifest) DescriptionIn(lang string) string { return pick(m.Description, lang) }

// DesktopGroup is Group, or DefaultGroup when empty.
func (m Manifest) DesktopGroup() string {
	if m.Group == "" {
		return DefaultGroup
	}
	return m.Group
}

// Installed is the record an app writes about itself the first time it runs (and
// that the Launcher writes when it installs one): the manifest plus where it is.
type Installed struct {
	Manifest
	Version string `json:"version"`
	Exe     string `json:"exe"`  // absolute path of <id>.app
	Icon    string `json:"icon"` // absolute path of the BMP icon ("" when none)
}

// RecordPath is where the record of app id lives under root.
func RecordPath(root, id string) string {
	return filepath.Join(root, filepath.FromSlash(MetaDir), id+".json")
}

// WriteInstalled saves the record atomically (write a temporary file, then rename).
func WriteInstalled(root string, in Installed) error {
	if err := in.Validate(); err != nil {
		return err
	}
	return writeJSON(RecordPath(root, in.ID), in)
}

// ReadInstalled lists the records under root, sorted by id. A record whose
// executable is gone (the .app file was deleted) is skipped, and so is anything
// that is not valid: one broken file must not hide the other apps.
func ReadInstalled(root string) []Installed {
	dir := filepath.Join(root, filepath.FromSlash(MetaDir))
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil
	}
	sort.Strings(files)
	var out []Installed
	for _, f := range files {
		if filepath.Base(f) == settingsFile {
			continue
		}
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var in Installed
		if json.Unmarshal(data, &in) != nil || in.Validate() != nil {
			continue
		}
		if _, err := os.Stat(in.Exe); err != nil {
			continue
		}
		out = append(out, in)
	}
	return out
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
