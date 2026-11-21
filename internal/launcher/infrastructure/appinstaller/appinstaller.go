// Package appinstaller is the domain.Installer of the Launcher: it downloads the
// zip of a catalog listing (<id>.app and app.json), checks it, puts the <id>.app
// in the reader's applications folder and registers the app, so it shows up on
// the home screen and in the Launcher.
//
// Installing by hand is the same thing: download <id>.zip from the release and
// extract the <id>.app into the reader's applications folder.
package appinstaller

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go-pocketbook-base/internal/launcher/domain"
	"go-pocketbook-base/internal/launcher/infrastructure/github"
	platform "go-pocketbook-base/internal/platform/domain"
)

// maxApp is the largest app (and zip) accepted (they are a few megabytes).
const maxApp = 64 << 20

// RegisterFunc registers the installed app (its icons, home screen entry and
// record), using l.Manifest. The composition root supplies it: it is the
// platform's registration.
type RegisterFunc func(root string, l domain.Listing) error

// Installer implements domain.Installer.
type Installer struct {
	Client   github.Doer
	Root     string // the reader's storage, e.g. /mnt/ext1
	Register RegisterFunc
}

// Install downloads the zip, checks its size and SHA-256 against the catalog,
// extracts <id>.app through a temporary file (an interrupted install never leaves
// half an app) and registers it with the manifest that came in the zip.
func (i Installer) Install(l domain.Listing) error {
	data, err := github.Get(i.Client, l.URL, maxApp)
	if err != nil {
		return err
	}
	if err := verify(data, l); err != nil {
		return err
	}
	binary, manifest, err := unpack(data, l.ID)
	if err != nil {
		return err
	}
	l.Manifest = manifest

	dir := filepath.Join(i.Root, "applications")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	target := filepath.Join(dir, l.ID+".app")
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, binary, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		os.Remove(tmp)
		return err
	}
	return i.Register(i.Root, l)
}

// verify compares what arrived with what the catalog announced.
func verify(data []byte, l domain.Listing) error {
	if l.Size > 0 && int64(len(data)) != l.Size {
		return fmt.Errorf("download incomplete: %d of %d bytes", len(data), l.Size)
	}
	if l.SHA256 != "" {
		sum := sha256.Sum256(data)
		if !strings.EqualFold(hex.EncodeToString(sum[:]), l.SHA256) {
			return fmt.Errorf("download corrupted: SHA-256 does not match the catalog")
		}
	}
	return nil
}

// unpack reads <id>.app and app.json from the zip. Only these two names are
// looked at, and nothing is written from the zip's own paths.
func unpack(data []byte, id string) (binary, manifest []byte, err error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, nil, fmt.Errorf("not a valid zip: %w", err)
	}
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
		switch {
		case isEntry(f.Name, id+".app"):
			binary, err = readEntry(f)
		case isEntry(f.Name, "app.json"):
			manifest, err = readEntry(f)
		}
		if err != nil {
			return nil, nil, err
		}
	}
	if binary == nil {
		return nil, nil, fmt.Errorf("the zip has no %s.app (it has: %s)", id, strings.Join(names, ", "))
	}
	m, err := platform.ParseManifest(manifest)
	if err != nil {
		return nil, nil, fmt.Errorf("the zip's app.json is not valid: %w", err)
	}
	if m.ID != id {
		return nil, nil, fmt.Errorf("the zip's app.json is for %q, not %q", m.ID, id)
	}
	return binary, manifest, nil
}

// isEntry reports whether the zip entry name is the file want, wherever the zip kept it: a leading
// "./" or folders, and Windows separators, do not make it another file. Nothing is written from the
// zip's own paths: only the name decides which entry is read.
func isEntry(name, want string) bool {
	name = strings.ReplaceAll(name, "\\", "/")
	return strings.EqualFold(name[strings.LastIndex(name, "/")+1:], want)
}

// readEntry reads a zip entry, refusing one that expands beyond maxApp.
func readEntry(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, maxApp+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxApp {
		return nil, fmt.Errorf("%s is larger than %d bytes", f.Name, maxApp)
	}
	return data, nil
}
