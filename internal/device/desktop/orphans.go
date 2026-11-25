package desktop

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// readerRoot is where the reader mounts its storage; the paths of view.json start with it.
const readerRoot = "/mnt/ext1/"

// onDisk is the file a path of view.json names, with the reader's storage found under root (on the
// reader root is /mnt/ext1 itself; on a drive it is the drive).
func onDisk(root, p string) string {
	if rest := strings.TrimPrefix(p, readerRoot); rest != p {
		return filepath.Join(root, filepath.FromSlash(rest))
	}
	return filepath.FromSlash(p)
}

// RemoveOrphans takes out of view.json the entries of user apps (keys "U_...") whose program is no
// longer there, and deletes their icons: what an app removed by hand, or by an old version, leaves
// behind. An entry without a path is left alone, and so is every entry whose program exists. It
// returns the ids of the apps it took out (report, when not nil, is told each as its turn comes); view.json is backed up, validated and replaced atomically
// like Register does it, and nothing is written when there is nothing to take out.
func RemoveOrphans(root string, report func(id string)) ([]string, error) {
	view, err := findView(root)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(view)
	if err != nil {
		return nil, err
	}
	var parsed struct{ Applications map[string]json.RawMessage }
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}
	var gone []string
	for key, raw := range parsed.Applications {
		var e struct{ Path string }
		if !strings.HasPrefix(key, "U_") || json.Unmarshal(raw, &e) != nil || e.Path == "" {
			continue
		}
		if _, err := os.Stat(onDisk(root, e.Path)); os.IsNotExist(err) {
			gone = append(gone, strings.TrimPrefix(key, "U_"))
		}
	}
	if len(gone) == 0 {
		return nil, nil
	}
	sort.Strings(gone)

	text := string(data)
	for _, id := range gone {
		if report != nil {
			report(id)
		}
		if text, err = withoutKey(text, "U_"+id); err != nil {
			return nil, err
		}
	}
	if err := replaceView(view, data, text); err != nil {
		return nil, err
	}
	for _, id := range gone {
		for _, suffix := range []string{".bmp", "_f.bmp"} {
			os.Remove(filepath.Join(root, "applications", "icons", id+suffix))
		}
	}
	return gone, nil
}
