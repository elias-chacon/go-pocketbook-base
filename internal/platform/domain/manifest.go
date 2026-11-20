// Package domain is the heart of the platform context: what an app of this
// project IS, with no knowledge of files, the reader or the screen.
//
// Ubiquitous language:
//
//   - App: a program of the ecosystem, one folder under ./apps and one <id>.app file.
//   - Manifest: the app's identity (id, titles, description), written by its author.
//   - InstalledApp: an app that has run on this reader, with its version and files.
//   - Settings: preferences shared by every app (today the language).
//
// The rules live here (what a valid manifest is, which text to show for a
// language); how things are stored or shown is in the other layers, which reach
// this one only through the ports in ports.go.
package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"go-pocketbook-base/internal/localization/i18n"
)

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
	// Intents are the actions this app handles when another app opens it (see
	// Intent), each "<app>.<verb>".
	Intents []string `json:"intents,omitempty"`
	// Data are the files of the app in applications/.pbgo (its own database, its settings): they
	// are deleted when the app is removed. Plain file names, never paths.
	Data []string `json:"data,omitempty"`
	// Shares are the files in applications/.pbgo that this app uses together with others (the
	// accounts): they are deleted only when no other installed app shares them.
	Shares []string `json:"shares,omitempty"`
}

// ParseManifest decodes and validates an app.json.
func ParseManifest(data []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, fmt.Errorf("platform: app.json: %w", err)
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
	for _, name := range append(append([]string{}, m.Data...), m.Shares...) {
		if !plainFileName(name) {
			problems = append(problems, fmt.Sprintf("data file %q must be a plain file name", name))
		}
	}
	for _, action := range m.Intents {
		if !ValidAction(action) {
			problems = append(problems, fmt.Sprintf("intent %q must look like app.verb", action))
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("platform: invalid manifest: %s", strings.Join(problems, "; "))
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

// Handles reports whether the app declared that it handles an intent action.
func (m Manifest) Handles(action string) bool {
	for _, a := range m.Intents {
		if a == action {
			return true
		}
	}
	return false
}

// DesktopGroup is Group, or DefaultGroup when empty.
func (m Manifest) DesktopGroup() string {
	if m.Group == "" {
		return DefaultGroup
	}
	return m.Group
}

// InstalledApp is an app that has run on this reader (or that the Launcher
// installed): the manifest plus where it is and which version.
type InstalledApp struct {
	Manifest
	Version string `json:"version"`
	Exe     string `json:"exe"`  // absolute path of <id>.app
	Icon    string `json:"icon"` // absolute path of the BMP icon ("" when none)
}

// plainFileName reports whether name is a file name and not a path (it never leaves the folder it is in).
func plainFileName(name string) bool {
	return name != "" && name != "." && name != ".." && !strings.ContainsAny(name, "/\\")
}
