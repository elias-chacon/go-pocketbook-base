// Package embedded is the domain.PackageSource of an installer that carries its
// apps inside its own executable: a folder (an embed.FS) with <id>.app and its
// <id>.json manifest side by side (and an optional <id>.png icon), plus an optional VERSION file.
package embedded

import (
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"go-pocketbook-base/internal/installer/domain"
)

// Source reads the packages from FS (its root is the folder of the files).
type Source struct{ FS fs.FS }

// Packages implements domain.PackageSource, sorted by id.
func (s Source) Packages() ([]domain.Package, error) {
	entries, err := fs.ReadDir(s.FS, ".")
	if err != nil {
		return nil, err
	}
	version := ""
	if v, err := fs.ReadFile(s.FS, "VERSION"); err == nil {
		version = strings.TrimSpace(string(v))
	}
	var out []domain.Package
	for _, e := range entries {
		id := strings.TrimSuffix(e.Name(), ".app")
		if e.IsDir() || id == e.Name() {
			continue
		}
		binary, err := fs.ReadFile(s.FS, e.Name())
		if err != nil {
			return nil, err
		}
		manifest, err := fs.ReadFile(s.FS, id+".json")
		if err != nil {
			return nil, fmt.Errorf("%s has no manifest: %w", e.Name(), err)
		}
		icon, _ := fs.ReadFile(s.FS, id+".png") // optional
		out = append(out, domain.Package{ID: id, Version: version, Manifest: manifest, Binary: binary, Icon: icon})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}
