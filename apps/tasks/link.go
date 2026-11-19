package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// linkFile remembers which account the tasks synchronize with, so that disconnecting
// does not have to remove an account the mail app uses too (an Outlook account serves both).
const linkFile = "tasks-link.json"

// link is the content of linkFile.
type link struct {
	Account string `json:"account,omitempty"` // the account chosen; "" means the first one that gives tasks
	Off     bool   `json:"off,omitempty"`     // synchronization was turned off
}

func readLink(dir string) link {
	var l link
	if data, err := os.ReadFile(filepath.Join(dir, linkFile)); err == nil {
		_ = json.Unmarshal(data, &l)
	}
	return l
}

func writeLink(dir string, l link) error {
	data, err := json.Marshal(l)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, linkFile), data, 0o600)
}
