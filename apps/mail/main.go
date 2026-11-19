// Command mail is the Mail app: it reads the mail of an account kept by the Accounts
// context (any IMAP server), through a cache in the shared database, a page at a
// time. It is the composition root of the mail screens; the rules live in
// internal/mail.
package main

import (
	_ "embed" // go:embed
	"log"

	"go-pocketbook-base/internal/platform"
)

//go:embed app.json
var manifest []byte

func main() {
	if err := platform.Run(platform.Config{Manifest: manifest}, newMail); err != nil {
		log.Fatal(err)
	}
}
