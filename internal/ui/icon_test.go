package ui

import (
	"bytes"
	"testing"
)

func TestRenderIcon(t *testing.T) {
	normal, focused := RenderIcon(128, false), RenderIcon(128, true)
	if b := normal.Bounds(); b.Dx() != 128 || b.Dy() != 128 {
		t.Fatalf("size = %v", b)
	}
	if normal.GrayAt(2, 2).Y != 255 || focused.GrayAt(2, 2).Y != 0 {
		t.Error("the focused icon is the normal one with inverted colors")
	}
	if bytes.Equal(normal.Pix, focused.Pix) {
		t.Error("focused and normal icons must differ")
	}
}
