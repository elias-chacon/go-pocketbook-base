// Command hello is the blueprint of every app of this project: a component
// showcase that follows the whole contract (see internal/appkit). To make a new
// app, copy this folder to apps/<id>, change app.json and the screen, and build
// it with `.\build.ps1 -App <id>` (or `make dist APP=<id>`).
//
// The contract, in the order things happen:
//
//  1. app.json (embedded below) names the app and its texts in pt-BR and en.
//  2. appkit.Run registers the icon on the first run, records the app for the
//     Launcher, reads the shared settings (language) and starts InkView.
//  3. newHello builds the screen for the real screen size; the kit connects
//     touch, keys, the system keyboard and the partial repaint to it.
package main

import (
	_ "embed" // go:embed
	"image"
	"log"

	"go-pocketbook-base/internal/appkit"
	"go-pocketbook-base/internal/ui"
)

//go:embed app.json
var manifest []byte

func main() {
	err := appkit.Run(appkit.Config{
		Manifest: manifest,
		// An app may draw its own launcher icon; without Icon the kit draws one
		// from the title (appkit.DefaultIcon).
		Icon: func(size int, focused bool) image.Image { return ui.RenderIcon(size, focused) },
	}, newHello)
	if err != nil {
		log.Fatal(err)
	}
}
