package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateWritesEveryExample(t *testing.T) {
	dir := t.TempDir()
	if err := generate(dir); err != nil {
		t.Fatal(err)
	}
	for _, ex := range examples() {
		fi, err := os.Stat(filepath.Join(dir, ex.file+".png"))
		if err != nil || fi.Size() < 500 {
			t.Errorf("%s: %v (size %v)", ex.file, err, fi)
		}
	}
}

func TestItoa(t *testing.T) {
	for in, want := range map[int]string{0: "0", 7: "7", 10: "10", 12: "12", 305: "305"} {
		if got := itoa(in); got != want {
			t.Errorf("itoa(%d) = %q", in, got)
		}
	}
}
