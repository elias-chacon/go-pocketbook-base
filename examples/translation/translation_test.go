package translation

import (
	"reflect"
	"testing"

	"go-pocketbook-base/examples/harness"
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/widget"
)

func TestTranslatedScreenFollowsTheLanguageAndFallsBackToEnglish(t *testing.T) {
	tr := NewTranslator()
	screen := GreetingScreen(tr, "Ana")
	texts := func() []string {
		var out []string
		for _, k := range screen.Elem().Children() {
			if l, ok := k.(*widget.Label); ok {
				out = append(out, l.Text)
			}
		}
		return out
	}
	want := []string{"Olá!", "Bem-vinda(o), Ana.", "Você tem 3 itens", "See you soon"} // "farewell" is English-only
	if got := texts(); !reflect.DeepEqual(got, want) {
		t.Fatalf("pt-BR = %q, want %q", got, want)
	}

	doc, _ := harness.NewDoc(screen)
	harness.Tap(doc, screen.Elem().Children()[4]) // open the language list
	list := doc.TargetAt(geom.Point{X: screen.Elem().Children()[4].Elem().Bounds.X + 20, Y: screen.Elem().Children()[4].Elem().Bounds.Bottom() + 6 + 6 + 76 + 76/2})
	if list == nil {
		t.Fatal("the language list must open")
	}
	harness.TapAt(doc, geom.Point{X: screen.Elem().Children()[4].Elem().Bounds.X + 20, Y: screen.Elem().Children()[4].Elem().Bounds.Bottom() + 6 + 6 + 76 + 76/2}) // "English"
	want = []string{"Hello!", "Welcome, Ana.", "You have 3 items", "See you soon"}
	if got := texts(); !reflect.DeepEqual(got, want) || tr.Lang() != "en" {
		t.Errorf("en = %q (lang %q), want %q", got, tr.Lang(), want)
	}
	if missing := tr.Missing("pt-BR"); !reflect.DeepEqual(missing, []string{"farewell"}) {
		t.Errorf("Missing(pt-BR) = %v", missing)
	}
}
