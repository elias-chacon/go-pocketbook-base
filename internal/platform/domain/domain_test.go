package domain

import (
	"strings"
	"testing"
)

const helloManifest = `{
  "id": "hello",
  "title": {"pt-BR": "Olá Go", "en": "Hello Go"},
  "description": {"pt-BR": "Vitrine", "en": "Showcase"}
}`

func TestParseManifest(t *testing.T) {
	m, err := ParseManifest([]byte(helloManifest))
	if err != nil || m.ID != "hello" || m.DesktopGroup() != DefaultGroup {
		t.Fatalf("%+v %v", m, err)
	}
	m.Group = "@Games"
	if m.DesktopGroup() != "@Games" {
		t.Error("an explicit group wins")
	}
}

func TestManifestRulesOfTheContract(t *testing.T) {
	bad := map[string]string{
		"not json":                 `{`,
		"no id":                    `{"title":{"pt-BR":"A"},"description":{"pt-BR":"B"}}`,
		"upper-case id":            `{"id":"Hello","title":{"pt-BR":"A"},"description":{"pt-BR":"B"}}`,
		"id that is not a name":    `{"id":"a/b","title":{"pt-BR":"A"},"description":{"pt-BR":"B"}}`,
		"title only in English":    `{"id":"x","title":{"en":"A"},"description":{"pt-BR":"B"}}`,
		"empty description":        `{"id":"x","title":{"pt-BR":"A"},"description":{"pt-BR":"  "}}`,
		"id starting with a digit": `{"id":"1x","title":{"pt-BR":"A"},"description":{"pt-BR":"B"}}`,
	}
	for name, data := range bad {
		if _, err := ParseManifest([]byte(data)); err == nil {
			t.Errorf("%s must be rejected", name)
		}
	}
	err := Manifest{ID: "A b"}.Validate()
	if err == nil || !strings.Contains(err.Error(), "primary language") {
		t.Errorf("all problems are reported together: %v", err)
	}
}

func TestTitlesFallBackLikeI18n(t *testing.T) {
	m, _ := ParseManifest([]byte(helloManifest))
	for lang, want := range map[string]string{"pt-BR": "Olá Go", "pt_BR": "Olá Go", "en": "Hello Go", "en-GB": "Hello Go", "fr": "Hello Go"} {
		if got := m.TitleIn(lang); got != want {
			t.Errorf("TitleIn(%q) = %q, want %q", lang, got, want)
		}
	}
	if m.DescriptionIn("en") != "Showcase" || m.DescriptionIn("") != "Vitrine" {
		t.Errorf("descriptions: %q %q", m.DescriptionIn("en"), m.DescriptionIn(""))
	}
	only := Manifest{ID: "x", Title: map[string]string{"es": "Hola"}}
	if only.TitleIn("pt-BR") != "Hola" {
		t.Error("with no pt/en text, any text beats nothing")
	}
	if (Manifest{ID: "bare"}).TitleIn("en") != "bare" {
		t.Error("with no title at all, the id is shown")
	}
}

func TestSettingsAreNormalizedAndNeverEmpty(t *testing.T) {
	if DefaultSettings().Lang != "pt-BR" {
		t.Errorf("default = %q", DefaultSettings().Lang)
	}
	for in, want := range map[string]string{"": "pt-BR", "en_US": "en-US", "PT-br": "pt-BR", "en": "en"} {
		if got := (Settings{Lang: in}).Normalized().Lang; got != want {
			t.Errorf("Normalized(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsReleaseIsOnlyMajorMinorBuild(t *testing.T) {
	if IsRelease("dev") || !IsRelease("1.2.3") || !IsRelease("v1.2.3") {
		t.Error("only MAJOR.MINOR.BUILD is a release")
	}
}
