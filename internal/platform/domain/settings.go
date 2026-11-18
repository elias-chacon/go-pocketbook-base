package domain

import "go-pocketbook-base/internal/localization/i18n"

// Settings are the preferences shared by every app. The Launcher edits them; the
// apps read them when they start (so the language chosen in the Launcher is the
// language of all).
type Settings struct {
	// Lang is the language tag ("pt-BR", "en"); empty means the primary language.
	Lang string `json:"lang"`
}

// DefaultSettings are the settings of a reader where nothing was chosen yet.
func DefaultSettings() Settings { return Settings{Lang: i18n.Primary} }

// Normalized returns the settings with every value in its canonical form and no
// value empty: whatever was read from disk becomes something usable.
func (s Settings) Normalized() Settings {
	out := DefaultSettings()
	if lang := i18n.Normalize(s.Lang); lang != "" {
		out.Lang = lang
	}
	return out
}
