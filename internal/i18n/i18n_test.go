package i18n

import (
	"reflect"
	"testing"
	"testing/fstest"
)

func newBundle() *Bundle {
	b := New()
	b.Add("pt-BR", map[string]string{"hello": "Olá, %s", "only.pt": "só em português", "ok": "OK"})
	b.Add("en", map[string]string{"hello": "Hello, %s", "only.en": "only in English", "ok": "OK"})
	b.Add("es", map[string]string{"hello": "Hola, %s"})
	return b
}

func TestPortugueseIsTheDefaultLanguage(t *testing.T) {
	b := newBundle()
	if b.Lang() != Primary || b.T("hello", "Ana") != "Olá, Ana" {
		t.Errorf("default = %q %q", b.Lang(), b.T("hello", "Ana"))
	}
}

func TestMissingKeysFallBackToEnglishThenPortugueseThenTheKey(t *testing.T) {
	b := newBundle()
	b.SetLang("es")
	if got := b.T("hello", "Ana"); got != "Hola, Ana" {
		t.Errorf("own language first: %q", got)
	}
	if got := b.T("only.en"); got != "only in English" {
		t.Errorf("a key Spanish lacks must come from English: %q", got)
	}
	if got := b.T("only.pt"); got != "só em português" {
		t.Errorf("a key English also lacks must come from Portuguese: %q", got)
	}
	if got := b.T("nobody.has.this"); got != "nobody.has.this" {
		t.Errorf("an unknown key is returned as is: %q", got)
	}

	b.SetLang("en")
	if got := b.T("only.pt"); got != "só em português" {
		t.Errorf("English lacking a key falls back to Portuguese: %q", got)
	}
}

func TestUnknownLanguageStillWorksThroughTheFallback(t *testing.T) {
	b := newBundle()
	b.SetLang("fr-FR") // nothing registered for French
	if got := b.T("hello", "Ana"); got != "Hello, Ana" {
		t.Errorf("unknown language must fall back to English: %q", got)
	}
}

func TestTagsAreNormalizedAndRegionsResolve(t *testing.T) {
	for in, want := range map[string]string{
		"pt_BR": "pt-BR", "PT-br": "pt-BR", "pt_BR.UTF-8": "pt-BR", "en_US@euro": "en-US", "EN": "en", "": "", "zh_hant_tw": "zh-hant-TW",
	} {
		if got := Normalize(in); got != want {
			t.Errorf("Normalize(%q) = %q, want %q", in, got, want)
		}
	}
	b := newBundle()
	b.SetLang("pt") // no "pt" messages: the registered pt-BR is used
	if b.Lang() != "pt-BR" || b.T("hello", "A") != "Olá, A" {
		t.Errorf("pt must resolve to pt-BR: %q", b.Lang())
	}
	b.SetLang("en-GB") // only "en" is registered
	if b.Lang() != "en" || b.T("hello", "A") != "Hello, A" {
		t.Errorf("en-GB must resolve to en: %q", b.Lang())
	}
	b.SetLang("") // empty goes back to the default
	if b.Lang() != Primary {
		t.Errorf("empty tag = %q", b.Lang())
	}
}

func TestOnChangeRunsOnlyWhenTheLanguageChanges(t *testing.T) {
	b := newBundle()
	calls := 0
	b.OnChange = func() { calls++ }
	b.SetLang("pt-BR") // already there
	b.SetLang("pt_br") // the same language, written differently
	if calls != 0 {
		t.Fatalf("no change, but OnChange ran %d times", calls)
	}
	b.SetLang("en")
	b.SetLang("en")
	if calls != 1 {
		t.Errorf("OnChange ran %d times, want 1", calls)
	}
}

func TestHasLangsAndMissing(t *testing.T) {
	b := newBundle()
	if !b.Has("hello") || b.Has("zzz") {
		t.Error("Has")
	}
	if got := b.Langs(); !reflect.DeepEqual(got, []string{"en", "es", "pt-BR"}) {
		t.Errorf("Langs = %v", got)
	}
	if got := b.Missing("es"); !reflect.DeepEqual(got, []string{"ok", "only.en", "only.pt"}) {
		t.Errorf("Missing(es) = %v", got)
	}
	if got := b.Missing("pt-BR"); !reflect.DeepEqual(got, []string{"only.en"}) {
		t.Errorf("Missing(pt-BR) = %v", got)
	}
	if got := b.Missing("xx"); len(got) != 4 {
		t.Errorf("an unregistered language misses everything: %v", got)
	}
}

func TestLoadingFromJSONAndFiles(t *testing.T) {
	b := New()
	if err := b.AddJSON("pt-BR", []byte(`{"a": "um"}`)); err != nil {
		t.Fatal(err)
	}
	if err := b.AddJSON("pt-BR", []byte(`{"b": "dois"}`)); err != nil { // extends, does not replace
		t.Fatal(err)
	}
	if b.T("a") != "um" || b.T("b") != "dois" {
		t.Errorf("AddJSON must merge: %q %q", b.T("a"), b.T("b"))
	}
	if err := b.AddJSON("en", []byte(`{not json`)); err == nil {
		t.Error("invalid JSON must be an error")
	}

	fsys := fstest.MapFS{
		"locales/pt-BR.json": {Data: []byte(`{"title": "Início"}`)},
		"locales/en.json":    {Data: []byte(`{"title": "Home"}`)},
		"locales/notes.txt":  {Data: []byte("ignored")},
		"locales/sub/x.json": {Data: []byte(`{"ignored": "x"}`)},
	}
	f := New()
	if err := f.AddFS(fsys, "locales"); err != nil {
		t.Fatal(err)
	}
	if got := f.Langs(); !reflect.DeepEqual(got, []string{"en", "pt-BR"}) {
		t.Errorf("files -> languages: %v", got)
	}
	f.SetLang("en")
	if f.T("title") != "Home" || f.Has("ignored") {
		t.Errorf("title=%q", f.T("title"))
	}
	if err := f.AddFS(fsys, "nope"); err == nil {
		t.Error("a missing folder must be an error")
	}
	bad := fstest.MapFS{"l/en.json": {Data: []byte(`[1]`)}}
	if err := New().AddFS(bad, "l"); err == nil {
		t.Error("a malformed file must be an error")
	}
}

func TestFormatVerbsAreOnlyAppliedWithArguments(t *testing.T) {
	b := New()
	b.Add("pt-BR", map[string]string{"pct": "100% pronto", "n": "%d itens"})
	if got := b.T("pct"); got != "100% pronto" {
		t.Errorf("a text with a literal %% and no arguments must stay intact: %q", got)
	}
	if got := b.T("n", 3); got != "3 itens" {
		t.Errorf("T(n, 3) = %q", got)
	}
}
