// Command eng is the engineering app: a menu of tests of the reader's panel and touch (colors,
// screen patterns, the update modes of InkView, a touch trail, scroll speed) that writes every
// measure to a log file. It is the composition root of internal/engineering.
package main

import (
	_ "embed" // go:embed
	"log"

	"go-pocketbook-base/internal/platform"
)

//go:embed app.json
var manifest []byte

func main() {
	if err := platform.Run(platform.Config{Manifest: manifest}, newEng); err != nil {
		log.Fatal(err)
	}
}
