package fold

import "testing"

func TestTextIgnoresCaseAndAccents(t *testing.T) {
	for in, want := range map[string]string{"Êxodo": "exodo", "JOÃO": "joao", "Gênesis": "genesis", "Ñandú": "nandu", "plain": "plain", "": ""} {
		if got := Text(in); got != want {
			t.Errorf("Text(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWordsKeepOnlyLettersAndDigits(t *testing.T) {
	got := Words("  Amor, e FÉ! 3:16 ")
	want := []string{"amor", "e", "fe", "3", "16"}
	if len(got) != len(want) {
		t.Fatalf("%v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("word %d: %q, want %q", i, got[i], want[i])
		}
	}
}

func TestMatchesAllNeedsEveryWordAndNoWordsMatchNothing(t *testing.T) {
	if !MatchesAll("Português (Brasil)", Words("PORTUGUES bras")) {
		t.Error("every word is in the text")
	}
	if MatchesAll("Português (Brasil)", Words("portugal")) {
		t.Error("a word that is not there")
	}
	if MatchesAll("anything", nil) {
		t.Error("an empty search matches nothing")
	}
}
