// Command calendar is the Calendar app: a month with the days that have events, the
// agenda of the chosen day and an editor, kept in the shared database on the reader.
// It is the composition root of the calendar screens; the rules live in
// internal/calendar.
package main

import (
	_ "embed" // go:embed
	"log"

	"go-pocketbook-base/internal/platform"
)

//go:embed app.json
var manifest []byte

func main() {
	if err := platform.Run(platform.Config{Manifest: manifest}, newCalendar); err != nil {
		log.Fatal(err)
	}
}
