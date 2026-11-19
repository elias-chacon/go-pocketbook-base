package main

import (
	"bytes"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// samePicture compares two PNGs by their pixels: the bytes depend on the Go
// version that compressed them, the picture does not.
func samePicture(t *testing.T, a, b []byte) bool {
	t.Helper()
	ia, err := png.Decode(bytes.NewReader(a))
	if err != nil {
		t.Fatal(err)
	}
	ib, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if ia.Bounds() != ib.Bounds() {
		return false
	}
	for y := ia.Bounds().Min.Y; y < ia.Bounds().Max.Y; y++ {
		for x := ia.Bounds().Min.X; x < ia.Bounds().Max.X; x++ {
			ra, ga, ba, aa := ia.At(x, y).RGBA()
			rb, gb, bb, ab := ib.At(x, y).RGBA()
			if ra != rb || ga != gb || ba != bb || aa != ab {
				return false
			}
		}
	}
	return true
}

func TestTheCommittedIconsAreTheOnesTheCodeDraws(t *testing.T) {
	for id := range renderers {
		want, ok, err := iconPNG(id)
		if err != nil || !ok {
			t.Fatalf("%s: %v %v", id, ok, err)
		}
		got, err := os.ReadFile(filepath.Join("..", "..", "apps", id, "icon.png"))
		if err != nil {
			t.Fatalf("apps/%s/icon.png is missing: run go run ./tools/appicons (%v)", id, err)
		}
		if !samePicture(t, got, want) {
			t.Errorf("apps/%s/icon.png is out of date: run go run ./tools/appicons", id)
		}
	}
}

func TestAnAppWithoutARendererHasNoGeneratedIcon(t *testing.T) {
	if _, ok, err := iconPNG("someone-elses-app"); ok || err != nil {
		t.Errorf("%v %v", ok, err)
	}
}

func TestWriteCreatesOneIconPerRenderer(t *testing.T) {
	dir := t.TempDir()
	for id := range renderers {
		os.MkdirAll(filepath.Join(dir, id), 0o755)
	}
	written, err := write(dir)
	if err != nil || len(written) != len(renderers) {
		t.Fatalf("%v %v", written, err)
	}
	for _, id := range written {
		if st, err := os.Stat(filepath.Join(dir, id, "icon.png")); err != nil || st.Size() == 0 {
			t.Errorf("%s: %v", id, err)
		}
	}
	if _, err := write(filepath.Join(dir, "missing")); err == nil {
		t.Error("an apps folder that does not exist")
	}
}
