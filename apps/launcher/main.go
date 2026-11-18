// Command launcher is the App Launcher of the ecosystem: it lists the apps
// installed on the reader, runs them and (in later versions) searches, installs
// and configures. It is the composition root of the launcher context and follows
// the app contract like any other app (see apps/hello, the blueprint).
package main

import (
	_ "embed" // go:embed
	"log"

	"go-pocketbook-base/internal/platform"
)

//go:embed app.json
var manifest []byte

func main() {
	err := platform.Run(platform.Config{Manifest: manifest, Icon: icon}, newLauncher)
	if err != nil {
		log.Fatal(err)
	}
}
