package harness

import (
	"os"
	"path/filepath"
	"testing"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/widget"
)

func TestSavePNGWritesTheScreen(t *testing.T) {
	doc, _ := NewDoc(widget.NewButton("Hello"))
	path := filepath.Join(t.TempDir(), "hello.png")
	if err := SavePNG(doc, path); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(path); err != nil || fi.Size() < 1000 {
		t.Errorf("png: %v size %v", err, fi)
	}
	if err := SavePNG(doc, filepath.Join(t.TempDir(), "missing", "x.png")); err == nil {
		t.Error("a path that cannot be created must be an error")
	}
}

func TestTheTestKitHelpers(t *testing.T) {
	clicks := 0
	btn := widget.NewButton("Tap")
	btn.OnClick = func(*widget.Event) { clicks++ }
	doc, kb := NewDoc(btn)
	if kb.Reply != "7" || doc.Keyboard != widget.Keyboard(kb) {
		t.Error("NewDoc gives the document a fake keyboard that answers 7")
	}
	if Inked(doc) == 0 {
		t.Error("a laid-out button draws something")
	}
	if got := Interactive(doc); len(got) != 1 || got[0] != widget.Widget(btn) {
		t.Errorf("Interactive = %v", got)
	}
	Tap(doc, btn)
	TapAt(doc, btn.Bounds.Center())
	if clicks != 2 {
		t.Errorf("Tap and TapAt touch the element: %d", clicks)
	}
	TapAt(doc, geom.Point{X: 5, Y: 1800}) // an empty corner: nothing happens
}
