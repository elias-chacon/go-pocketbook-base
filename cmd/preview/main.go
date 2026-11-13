// Command preview renders the app screen to a PNG so the UI can be developed
// on a desktop without a device.
package main

import (
	"flag"
	"image/png"
	"log"
	"os"

	"go-pocketbook-base/internal/ui"
)

func main() {
	w := flag.Int("w", 1264, "screen width (default: PB741 Era; PocketBook 6\" = 758)")
	h := flag.Int("h", 1680, "screen height (default: PB741 Era; PocketBook 6\" = 1024)")
	out := flag.String("o", "preview.png", "output file")
	status := flag.String("status", "", "status line to show")
	flag.Parse()

	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, ui.Render(*w, *h, *status)); err != nil {
		log.Fatal(err)
	}
	log.Printf("wrote %s (%dx%d)", *out, *w, *h)
}
