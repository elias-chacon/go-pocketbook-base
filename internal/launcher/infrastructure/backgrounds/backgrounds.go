// Package backgrounds lists the pictures the person put in the backgrounds
// folder of the reader (applications/backgrounds) to choose a Launcher background.
package backgrounds

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// extensions are the picture formats the toolkit decodes.
var extensions = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true}

// Dir is the backgrounds folder of the reader whose storage is root.
func Dir(root string) string { return filepath.Join(root, "applications", "backgrounds") }

// Names lists the pictures of dir, sorted (a missing folder is an empty list).
func Names(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && extensions[strings.ToLower(filepath.Ext(e.Name()))] && !strings.HasPrefix(e.Name(), ".") {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}
