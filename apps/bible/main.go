// Command bible is the Bible app: the reading progress and the verse of the day, the books,
// chapters and verses of any version the Midvash API offers (downloaded once into the reader's
// own database, then read offline) and the settings of the text. It is the composition root of the
// Bible screens; the rules live in internal/bible.
package main

import (
	_ "embed" // go:embed
	"log"

	"go-pocketbook-base/internal/platform"
)

//go:embed app.json
var manifest []byte

func main() {
	if err := platform.Run(platform.Config{Manifest: manifest}, newBible); err != nil {
		log.Fatal(err)
	}
}
