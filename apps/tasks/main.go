// Command tasks is the Tasks app: a to-do list with My Day, Important, Planned and
// Overdue, kept in the shared database on the reader. It is the composition root of
// the tasks screens; the rules live in internal/tasks.
package main

import (
	_ "embed" // go:embed
	"log"

	"go-pocketbook-base/internal/platform"
)

//go:embed app.json
var manifest []byte

func main() {
	if err := platform.Run(platform.Config{Manifest: manifest}, newTasks); err != nil {
		log.Fatal(err)
	}
}
