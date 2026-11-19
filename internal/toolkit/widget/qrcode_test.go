package widget

import (
	"strings"
	"testing"
)

func TestQRCodeIsASquareBitmapOfTheAskedSize(t *testing.T) {
	im, err := NewQRCode("https://example.org/auth?x=1", 320)
	if err != nil {
		t.Fatal(err)
	}
	b := im.Source.Bounds()
	if b.Dx() != b.Dy() || b.Dx() < 100 {
		t.Fatalf("bounds = %v", b)
	}
	if im.Style.Width.V != 320 || im.Style.Height.V != 320 {
		t.Fatalf("size = %v x %v", im.Style.Width, im.Style.Height)
	}
}

func TestQRCodeRefusesATextThatDoesNotFit(t *testing.T) {
	if _, err := NewQRCode(strings.Repeat("x", 5000), 200); err == nil {
		t.Fatal("5000 characters do not fit in a QR code")
	}
}
