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
