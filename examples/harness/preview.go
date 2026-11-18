// examples/harness/preview.go
package harness

import (
	"image/png"
	"os"

	"go-pocketbook-base/internal/toolkit/widget"
)

// SavePNG renders the document and writes it to a PNG file: the quickest way
// to see a screen without a device.
func SavePNG(doc *widget.Document, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(f, doc.Render()); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
