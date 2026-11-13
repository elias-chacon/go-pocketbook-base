// Command docgen renders one PNG per widget for the README, using the widget
// library itself so the pictures never go stale:
//
//	go run ./tools/docgen            # writes docs/components/*.png
package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"

	"go-pocketbook-base/internal/geom"
	"go-pocketbook-base/internal/widget"
)

func main() {
	out := flag.String("o", "docs/components", "output directory")
	flag.Parse()
	if err := generate(*out); err != nil {
		log.Fatal(err)
	}
}

// generate writes every example into dir.
func generate(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, ex := range examples() {
		img := render(ex)
		path := filepath.Join(dir, ex.file+".png")
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		err = png.Encode(f, img)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		log.Printf("wrote %s (%dx%d)", path, img.Bounds().Dx(), img.Bounds().Dy())
	}
	return nil
}

// render draws the example in a framed host panel and crops the screen to it.
func render(ex example) *image.Gray {
	w := ex.build()
	host := widget.NewPanel(w)
	host.Style.Width = widget.Px(ex.width)
	host.Style.Padding = geom.All(24)
	host.Style.BorderColor, host.Style.BorderWidth, host.Style.BorderRadius = widget.Mid, 2, 14
	root := widget.NewColumn(0, host)
	root.Style.AlignItems = widget.AlignStart
	root.Style.Padding = geom.All(10)

	d := widget.NewDocument(1404, 1872)
	d.SetRoot(root)
	d.Layout()
	if ex.open != nil {
		ex.open(d, w)
	}
	img := d.Render()

	b := host.Bounds
	r := image.Rect(int(b.X)-8, int(b.Y)-8, int(b.Right())+8, int(b.Bottom()+ex.extra)+8)
	return img.SubImage(r.Intersect(img.Bounds())).(*image.Gray)
}
