// Package readerfs is the domain.Deployer that writes to a PocketBook mounted as a
// drive: the app goes to its applications folder, then Register adds the app to
// the home screen (icons, view.json, the record the Launcher reads).
package readerfs

import (
	"os"
	"path/filepath"

	"go-pocketbook-base/internal/installer/domain"
)

// RegisterFunc registers the app of a package on the reader at root. The
// composition root supplies it (it knows each app's icon); it reports whether
// the home screen entry changed.
type RegisterFunc func(root string, p domain.Package) (changed bool, err error)

// Deployer implements domain.Deployer.
type Deployer struct{ Register RegisterFunc }

// Deploy writes <root>/applications/<id>.app (through a temporary file, so an
// interrupted copy never leaves half an app) and registers it.
func (d Deployer) Deploy(r domain.Reader, p domain.Package) (bool, error) {
	dir := filepath.Join(r.Root, "applications")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, err
	}
	target := filepath.Join(dir, p.ID+".app")
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, p.Binary, 0o755); err != nil {
		return false, err
	}
	if err := os.Rename(tmp, target); err != nil {
		os.Remove(tmp)
		return false, err
	}
	return d.Register(r.Root, p)
}
