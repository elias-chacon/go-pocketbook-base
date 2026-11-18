// Package i18n is the translation module shared by every app built on this base.
//
// The rule of the project: apps are written in Brazilian Portuguese (pt-BR, the
// Primary language) and always fall back to English (en, the Fallback) for
// anything a language does not translate, so a missing string is never blank.
//
// Messages are flat key -> text maps, usually kept in one JSON file per language
// (locales/pt-BR.json, locales/en.json) embedded in the app with go:embed:
//
//	//go:embed locales/*.json
//	var locales embed.FS
//
//	tr := i18n.New()
//	if err := tr.AddFS(locales, "locales"); err != nil { ... }
//	tr.SetLang("en")
//	label := widget.NewLabel(tr.T("greeting", name)) // "Hello, %s"
//
// Text may contain fmt verbs (%s, %d...), filled from the arguments of T.
package i18n

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// The languages every app supports.
const (
	Primary  = "pt-BR" // what the app is written in, and the default language
	Fallback = "en"    // used for any key the current language lacks
)

// Bundle holds the messages of every language and the current language.
type Bundle struct {
	lang string
	msgs map[string]map[string]string // language -> key -> text

	// OnChange runs after SetLang changes the language (rebuild the screen here).
	OnChange func()
}

// New creates an empty Bundle whose current language is Primary.
func New() *Bundle {
	return &Bundle{lang: Primary, msgs: map[string]map[string]string{}}
}

// Normalize turns a language tag as the system or a file name writes it
// ("pt_BR", "PT-br", "pt_BR.UTF-8", "en_US@euro") into the canonical form
// ("pt-BR", "en-US"): lower-case language, upper-case region.
func Normalize(tag string) string {
	if i := strings.IndexAny(tag, ".@"); i >= 0 {
		tag = tag[:i]
	}
	parts := strings.FieldsFunc(tag, func(r rune) bool { return r == '-' || r == '_' })
	if len(parts) == 0 {
		return ""
	}
	parts[0] = strings.ToLower(parts[0])
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) == 2 {
			parts[i] = strings.ToUpper(parts[i])
		}
	}
	return strings.Join(parts, "-")
}

// base is the language without its region: "pt-BR" -> "pt".
func base(lang string) string {
	if i := strings.IndexByte(lang, '-'); i >= 0 {
		return lang[:i]
	}
	return lang
}

// Add registers (or extends) the messages of a language.
func (b *Bundle) Add(lang string, messages map[string]string) {
	lang = Normalize(lang)
	if b.msgs[lang] == nil {
		b.msgs[lang] = map[string]string{}
	}
	for k, v := range messages {
		b.msgs[lang][k] = v
	}
}

// AddJSON registers the messages of a language from a flat JSON object
// {"key": "text", ...}.
func (b *Bundle) AddJSON(lang string, data []byte) error {
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("i18n: %s: %w", lang, err)
	}
	b.Add(lang, m)
	return nil
}

// AddFS registers every <language>.json file of dir in fsys (typically an
// embed.FS): "locales/pt-BR.json" adds pt-BR.
func (b *Bundle) AddFS(fsys fs.FS, dir string) error {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return fmt.Errorf("i18n: %w", err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || path.Ext(name) != ".json" {
			continue
		}
		data, err := fs.ReadFile(fsys, path.Join(dir, name))
		if err != nil {
			return fmt.Errorf("i18n: %w", err)
		}
		if err := b.AddJSON(strings.TrimSuffix(name, ".json"), data); err != nil {
			return err
		}
	}
	return nil
}

// Langs lists the registered languages, sorted.
func (b *Bundle) Langs() []string {
	out := make([]string, 0, len(b.msgs))
	for l := range b.msgs {
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}

// Lang is the current language (always a registered one when possible).
func (b *Bundle) Lang() string { return b.resolve(b.lang) }

// SetLang selects the language (any tag: "pt_BR", "en-us"...). A language with
// no messages still works: every lookup falls back. OnChange runs when the
// effective language changed.
func (b *Bundle) SetLang(tag string) {
	tag = Normalize(tag)
	if tag == "" {
		tag = Primary
	}
	before := b.Lang()
	b.lang = tag
	if b.Lang() != before && b.OnChange != nil {
		b.OnChange()
	}
}

// resolve maps a tag to a registered language: itself, else the registered
// language with the same base ("pt" -> "pt-BR"; "en-GB" -> "en"), else itself.
func (b *Bundle) resolve(lang string) string {
	if _, ok := b.msgs[lang]; ok {
		return lang
	}
	if _, ok := b.msgs[base(lang)]; ok {
		return base(lang)
	}
	for _, l := range b.Langs() { // sorted, so the choice is stable
		if base(l) == base(lang) {
			return l
		}
	}
	return lang
}

// chain is the order in which languages are searched: the current one, then
// English, then Portuguese.
func (b *Bundle) chain() []string {
	var out []string
	for _, l := range []string{b.lang, Fallback, Primary} {
		l = b.resolve(l)
		dup := false
		for _, have := range out {
			dup = dup || have == l
		}
		if !dup {
			out = append(out, l)
		}
	}
	return out
}

// T returns the text of key in the current language, falling back to English
// and then Portuguese; a key nobody defines is returned as is (so a missing
// translation is visible, not blank). args fill the fmt verbs of the text.
func (b *Bundle) T(key string, args ...interface{}) string {
	text := key
	for _, l := range b.chain() {
		if s, ok := b.msgs[l][key]; ok {
			text = s
			break
		}
	}
	if len(args) > 0 {
		return fmt.Sprintf(text, args...)
	}
	return text
}

// Has reports whether some language in the search chain defines key.
func (b *Bundle) Has(key string) bool {
	for _, l := range b.chain() {
		if _, ok := b.msgs[l][key]; ok {
			return true
		}
	}
	return false
}

// Missing lists, sorted, the keys that Primary or Fallback define but lang does
// not: what is still untranslated. Use it in a test to keep a language complete.
func (b *Bundle) Missing(lang string) []string {
	have := b.msgs[b.resolve(Normalize(lang))]
	seen := map[string]bool{}
	var out []string
	for _, ref := range []string{Primary, Fallback} {
		for k := range b.msgs[b.resolve(ref)] {
			if _, ok := have[k]; !ok && !seen[k] {
				seen[k] = true
				out = append(out, k)
			}
		}
	}
	sort.Strings(out)
	return out
}
