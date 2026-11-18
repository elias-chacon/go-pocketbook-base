// Package domain is the heart of the Installer context: putting the apps of the
// ecosystem on a PocketBook connected to the computer, for someone who only wants
// to double-click a program. It knows nothing of operating systems, files or the
// console.
//
// Ubiquitous language:
//
//   - Reader: a PocketBook mounted on the computer (a drive, USB "PC link").
//   - Package: one app to put on the Reader: its <id>.app and its manifest.
//   - Outcome: what happened to one Package.
package domain

import "errors"

// Reader is a PocketBook mounted on this computer.
type Reader struct {
	Root  string // the root of its storage, e.g. "F:\" or "/media/me/POCKETBOOK"
	Label string // what the person recognizes it by
}

// Package is one app to install.
type Package struct {
	ID       string
	Version  string
	Manifest []byte // the app.json
	Binary   []byte // the <id>.app
}

// Outcome is the result of installing one Package.
type Outcome struct {
	ID         string
	Registered bool // the home-screen entry was added or repaired (the app is new on the reader)
	Err        error
}

// ErrNoReader is returned when no PocketBook is connected.
var ErrNoReader = errors.New("installer: no PocketBook found")

// ErrNoPackages is returned when the installer carries no app.
var ErrNoPackages = errors.New("installer: no app to install")

// ReaderFinder is the port that finds the connected readers.
type ReaderFinder interface{ Find() []Reader }

// PackageSource is the port that provides the apps to install.
type PackageSource interface{ Packages() ([]Package, error) }

// Deployer is the port that puts one Package on a Reader: the file in its
// applications folder, then the app's registration on the home screen.
type Deployer interface {
	Deploy(r Reader, p Package) (registered bool, err error)
}
