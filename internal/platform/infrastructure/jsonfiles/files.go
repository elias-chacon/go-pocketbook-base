package jsonfiles

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go-pocketbook-base/internal/platform/domain"
)

// Files is the domain.AppFiles of the reader's storage.
type Files struct{ root string }

var _ domain.AppFiles = Files{}

// NewFiles creates the remover of the reader whose storage is root.
func NewFiles(root string) Files { return Files{root: root} }

// ownedBy reports whether a file of the data folder is named after the app: "<id>." , "<id>-" or
// "<id>_" followed by anything.
func ownedBy(appID, name string) bool {
	if !strings.HasPrefix(name, appID) || len(name) == len(appID) {
		return false
	}
	return strings.ContainsRune(".-_", rune(name[len(appID)]))
}

// Remove implements domain.AppFiles: <id>.app, its log, its record (what makes the Launcher
// list it) and the given files of the data folder. What is not there is not a failure; every
// file is tried and the failures are reported together.
func (f Files) Remove(appID string, dataFiles []string) error {
	apps := filepath.Join(f.root, "applications")
	paths := []string{
		filepath.Join(apps, appID+".app"), filepath.Join(apps, appID+".app.new"), filepath.Join(apps, appID+".log"),
		RecordPath(f.root, appID),
	}
	for _, name := range dataFiles {
		if name == "" || strings.ContainsAny(name, `/\`) || name == ".." {
			continue // a name that would leave the folder is never touched
		}
		paths = append(paths, filepath.Join(metaDir(f.root), name))
	}
	// By convention everything named after the app (bible.db, bible.db-journal, tasks-link.json)
	// is its own, even when the record of an older version did not list it.
	if entries, err := os.ReadDir(metaDir(f.root)); err == nil {
		for _, e := range entries {
			if !e.IsDir() && ownedBy(appID, e.Name()) {
				paths = append(paths, filepath.Join(metaDir(f.root), e.Name()))
			}
		}
	}
	var problems []string
	for _, p := range paths {
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			problems = append(problems, err.Error())
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("files: %s", strings.Join(problems, "; "))
	}
	return nil
}
