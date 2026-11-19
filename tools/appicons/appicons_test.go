package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

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
		if !bytes.Equal(got, want) {
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
