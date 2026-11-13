// Command preview renders the demo screens to PNG files so the UI can be
// developed on a desktop without a device.
//
//	go run ./cmd/preview                 # all pages: preview-<n>.png
//	go run ./cmd/preview -page 2 -o x.png
package main

import (
	"flag"
	"fmt"
	"image/png"
	"log"
	"os"

	"go-pocketbook-base/internal/demo"
	"go-pocketbook-base/internal/version"
	"go-pocketbook-base/internal/widget"
)

func main() {
	w := flag.Int("w", 1404, "screen width (default: PB741)")
	h := flag.Int("h", 1872, "screen height (default: PB741)")
	pageIdx := flag.Int("page", -1, "page index to render; -1 renders every page")
	out := flag.String("o", "", "output file (default preview-<page>.png)")
	flag.Parse()

	app := demo.New(*w, *h, demo.Config{Version: version.Value, Keyboard: &widget.FakeKeyboard{}})
	pages := []int{*pageIdx}
	if *pageIdx < 0 {
		pages = pages[:0]
		for i := range app.Pages() {
			pages = append(pages, i)
		}
	}
	for _, i := range pages {
		app.ShowPage(i)
		name := *out
		if name == "" || len(pages) > 1 {
			name = fmt.Sprintf("preview-%d.png", i)
		}
		write(name, app)
	}
}

func write(name string, app *demo.App) {
	f, err := os.Create(name)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, app.Doc.Render()); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s", name)
}
