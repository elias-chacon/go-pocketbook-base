package domain

import "go-pocketbook-base/internal/kernel/fold"

// Fold lowers the case and removes the accents, so "Êxodo" and "exodo" meet.
func Fold(s string) string { return fold.Text(s) }

// Words splits a search into folded words (letters and digits only).
func Words(query string) []string { return fold.Words(query) }

// MatchesAll reports whether the text has every word (the words are already folded).
func MatchesAll(text string, words []string) bool { return fold.MatchesAll(text, words) }
