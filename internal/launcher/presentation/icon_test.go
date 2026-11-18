package presentation

import (
	"image"
	"testing"
)

func TestTheLauncherDrawsItsOwnIcon(t *testing.T) {
	normal, focused := RenderIcon(128, false).(*image.Gray), RenderIcon(128, true).(*image.Gray)
	if normal.GrayAt(2, 2).Y != 255 || focused.GrayAt(2, 2).Y != 0 {
		t.Error("the focused icon is the inverted one")
	}
	dark := 0
	for _, p := range normal.Pix {
		if p < 128 {
			dark++
		}
	}
	if dark == 0 {
		t.Error("the icon draws something")
	}
}
