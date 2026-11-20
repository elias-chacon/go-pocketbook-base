// Package fold compares texts the way a person searches: without the case and without the accents,
// so "Êxodo" and "exodo" meet. It is part of the shared kernel (it depends on nothing).
package fold

import (
	"strings"
	"unicode"
)

// accents maps the accented letters of the languages of the apps to the plain ones.
var accents = map[rune]rune{
	'à': 'a', 'á': 'a', 'â': 'a', 'ã': 'a', 'ä': 'a', 'å': 'a', 'ā': 'a',
	'è': 'e', 'é': 'e', 'ê': 'e', 'ë': 'e', 'ē': 'e',
	'ì': 'i', 'í': 'i', 'î': 'i', 'ï': 'i', 'ī': 'i',
	'ò': 'o', 'ó': 'o', 'ô': 'o', 'õ': 'o', 'ö': 'o', 'ō': 'o',
	'ù': 'u', 'ú': 'u', 'û': 'u', 'ü': 'u', 'ū': 'u',
	'ç': 'c', 'ñ': 'n', 'ý': 'y', 'ÿ': 'y',
}

// Text lowers the case and removes the accents.
func Text(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		r = unicode.ToLower(r)
		if p, ok := accents[r]; ok {
			r = p
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Words splits a search into folded words (letters and digits only).
func Words(query string) []string {
	return strings.FieldsFunc(Text(query), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
}

// MatchesAll reports whether the text has every word (the words are already folded). No words
// match nothing.
func MatchesAll(text string, words []string) bool {
	if len(words) == 0 {
		return false
	}
	folded := Text(text)
	for _, w := range words {
		if !strings.Contains(folded, w) {
			return false
		}
	}
	return true
}
