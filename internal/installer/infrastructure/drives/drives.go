// Package drives finds the PocketBooks connected to this computer by looking for
// the reader's system folder in the places each operating system mounts drives.
package drives

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"go-pocketbook-base/internal/installer/domain"
)

// viewJSON is where the reader keeps its home screen; a drive with it is a PocketBook.
var viewJSON = []string{
	filepath.Join("system", "profiles", "default", "config", "desktop", "view.json"),
	filepath.Join("system", "config", "desktop", "view.json"),
}

// Finder is the domain.ReaderFinder of this computer.
type Finder struct {
	// Roots lists the places to look (nil: the ones of the running system).
	Roots func() []string
}

// Find implements domain.ReaderFinder.
func (f Finder) Find() []domain.Reader {
	roots := f.Roots
	if roots == nil {
		roots = DefaultRoots
	}
	var out []domain.Reader
	for _, root := range roots() {
		if IsReader(root) {
			out = append(out, domain.Reader{Root: root, Label: label(root)})
		}
	}
	return out
}

// IsReader reports whether root is the storage of a PocketBook.
func IsReader(root string) bool {
	for _, rel := range viewJSON {
		if st, err := os.Stat(filepath.Join(root, rel)); err == nil && !st.IsDir() {
			return true
		}
	}
	return false
}

func label(root string) string {
	if runtime.GOOS == "windows" {
		return root
	}
	return filepath.Base(root)
}

// DefaultRoots are the mount points of the running system.
func DefaultRoots() []string { return rootsFor(runtime.GOOS, os.Getenv("USER"), filepath.Glob) }

// rootsFor lists the candidates for an operating system; glob is injected so the
// rules are testable on any computer.
func rootsFor(goos, user string, glob func(string) ([]string, error)) []string {
	switch goos {
	case "windows":
		var out []string
		for c := 'A'; c <= 'Z'; c++ {
			out = append(out, string(c)+`:\`)
		}
		return out
	case "darwin":
		return globAll(glob, "/Volumes/*")
	default: // linux and the others
		patterns := []string{"/media/*", "/media/*/*", "/run/media/*/*", "/mnt/*"}
		if user != "" {
			patterns = append(patterns, "/media/"+user+"/*", "/run/media/"+user+"/*")
		}
		return globAll(glob, patterns...)
	}
}

func globAll(glob func(string) ([]string, error), patterns ...string) []string {
	seen := map[string]bool{}
	var out []string
	for _, p := range patterns {
		matches, _ := glob(p)
		for _, m := range matches {
			if !seen[m] {
				seen[m] = true
				out = append(out, m)
			}
		}
	}
	sort.Strings(out)
	return out
}
