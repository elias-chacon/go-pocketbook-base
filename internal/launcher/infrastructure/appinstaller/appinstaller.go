// Package appinstaller is the domain.Installer of the Launcher: it downloads the
// <id>.app of a catalog listing into the reader's applications folder and
// registers it, so the app shows up on the home screen and in the Launcher.
package appinstaller

import (
	"fmt"
	"os"
	"path/filepath"

	"go-pocketbook-base/internal/launcher/domain"
	"go-pocketbook-base/internal/launcher/infrastructure/github"
)

// maxApp is the largest app accepted (they are a few megabytes).
const maxApp = 64 << 20

// RegisterFunc registers the downloaded app (its icons, home screen entry and
// record). The composition root supplies it: it is the platform's registration.
type RegisterFunc func(root string, l domain.Listing) error

// Installer implements domain.Installer.
type Installer struct {
	Client   github.Doer
	Root     string // the reader's storage, e.g. /mnt/ext1
	Register RegisterFunc
}

// Install downloads the app through a temporary file (an interrupted download
// never leaves half an app) and registers it. The size announced by the release
// must match what arrived.
func (i Installer) Install(l domain.Listing) error {
	data, err := github.Get(i.Client, l.URL, maxApp)
	if err != nil {
		return err
	}
	if l.Size > 0 && int64(len(data)) != l.Size {
		return fmt.Errorf("download incomplete: %d of %d bytes", len(data), l.Size)
	}
	dir := filepath.Join(i.Root, "applications")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	target := filepath.Join(dir, l.ID+".app")
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		os.Remove(tmp)
		return err
	}
	return i.Register(i.Root, l)
}
