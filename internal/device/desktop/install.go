// Package desktop registers an app on the PocketBook desktop (home screen).
//
// PocketBook has no installer: an app is just a file in /mnt/ext1/applications.
// To get a launcher icon the app must (1) provide BMP icons and (2) be listed in
// system/profiles/default/config/desktop/view.json. Register does both, so
// copying the single .app file to the device is the whole installation.
package desktop

import (
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/image/bmp"
)

// IconSize is the icon edge in pixels (the desktop accepts up to 128x128).
const IconSize = 128

// Config describes the app to register.
type Config struct {
	Root  string // storage root, e.g. /mnt/ext1
	Exe   string // absolute path of the running .app
	Title string // name shown on the desktop
	Group string // desktop group, e.g. "@General"
	// Icon renders the launcher icon; focused is the pressed/inverted variant.
	Icon func(size int, focused bool) image.Image
}

// Result reports what Register did.
type Result struct {
	Key     string // view.json key, "U_<name>"
	Changed bool   // view.json was modified
	Backup  string // backup file, when Changed
}

// Register is idempotent: it writes missing icons and adds the app to
// view.json only when it is not already listed. view.json is backed up, the
// patched content is validated as JSON, and the file is replaced atomically.
func Register(c Config) (Result, error) {
	name := strings.TrimSuffix(filepath.Base(c.Exe), ".app")
	res := Result{Key: "U_" + name}

	if err := writeIcons(c, name); err != nil {
		return res, fmt.Errorf("icons: %w", err)
	}

	view, err := findView(c.Root)
	if err != nil {
		return res, err
	}
	data, err := os.ReadFile(view)
	if err != nil {
		return res, err
	}
	text := string(data)

	var patched string
	if regexp.MustCompile(`"` + regexp.QuoteMeta(res.Key) + `"\s*:`).MatchString(text) {
		// Already registered: only repair a wrong launch path (an earlier
		// version wrote a relative one, which the desktop cannot start).
		patched, err = fixPath(text, res.Key, launchPath(c))
		if err != nil {
			return res, err
		}
		if patched == text {
			return res, nil
		}
	} else if patched, err = patch(text, res.Key, entry(c, name), c.Group); err != nil {
		return res, err
	}

	res.Backup = fmt.Sprintf("%s.bak-%s", view, time.Now().Format("20060102-150405"))
	if err := os.WriteFile(res.Backup, data, 0o644); err != nil {
		return res, fmt.Errorf("backup: %w", err)
	}
	tmp := view + ".tmp"
	if err := os.WriteFile(tmp, []byte(patched), 0o644); err != nil {
		return res, err
	}
	if err := os.Rename(tmp, view); err != nil {
		os.Remove(tmp)
		return res, err
	}
	res.Changed = true
	return res, nil
}

func findView(root string) (string, error) {
	// On 6.x firmware system/config is a link to system/profiles/default/config.
	for _, rel := range []string{
		"system/profiles/default/config/desktop/view.json",
		"system/config/desktop/view.json",
	} {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("view.json not found under %s/system", root)
}

// launchPath is the path the desktop uses to start the app. It must be
// absolute: a relative "applications/x.app" shows the icon but the app never
// starts when tapped (verified on PB741; the MobileRead wiki also uses
// absolute paths for "path").
func launchPath(c Config) string { return filepath.ToSlash(c.Exe) }

var rePath = regexp.MustCompile(`("path"\s*:\s*)"[^"]*"`)

// fixPath makes the app's "path" in its existing view.json entry equal want.
// It returns text unchanged when it already does.
func fixPath(text, key, want string) (string, error) {
	// "_comment" is an array in the same object, so decode entries leniently.
	var raw struct{ Applications map[string]json.RawMessage }
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return "", fmt.Errorf("view.json is not valid JSON: %w", err)
	}
	var cur struct{ Path string }
	if err := json.Unmarshal(raw.Applications[key], &cur); err != nil {
		return text, nil // key only in an apps array, or not an entry object: leave it alone
	}
	if cur.Path == want {
		return text, nil
	}

	// Rewrite the first "path" inside this entry (our own entries put it first).
	start := regexp.MustCompile(`"` + regexp.QuoteMeta(key) + `"\s*:\s*\{`).FindStringIndex(text)
	if start == nil {
		return "", fmt.Errorf("entry %s not found", key)
	}
	loc := rePath.FindStringSubmatchIndex(text[start[1]:])
	if loc == nil {
		return "", fmt.Errorf("no path in entry %s", key)
	}
	a, b := start[1]+loc[3], start[1]+loc[1] // replace only the quoted value after `"path":`
	out := text[:a] + `"` + want + `"` + text[b:]
	// Re-check: the new text must be valid and the entry must now point at want.
	var check struct{ Applications map[string]json.RawMessage }
	if err := json.Unmarshal([]byte(out), &check); err != nil {
		return "", fmt.Errorf("patched view.json would be invalid: %w", err)
	}
	var v struct{ Path string }
	if err := json.Unmarshal(check.Applications[key], &v); err != nil || v.Path != want {
		return "", fmt.Errorf("could not update path of %s", key)
	}
	return out, nil
}

// entry builds the "applications" definition. The icons are relative to the
// user storage (as documented in the _comment of view.json, and verified);
// the launch path is absolute, see launchPath.
func entry(c Config, name string) string {
	rel := func(p string) string {
		if r, err := filepath.Rel(c.Root, p); err == nil && !strings.HasPrefix(r, "..") {
			return filepath.ToSlash(r)
		}
		return filepath.ToSlash(p)
	}
	title, _ := json.Marshal(c.Title)
	icon := func(n string) string { return rel(filepath.Join(c.Root, "applications", "icons", n)) }
	return fmt.Sprintf(`
        "U_%s": {
            "path": %q,
            "title": %s,
            "icon": { "path": %q },
            "focused_icon": { "path": %q }
        }`, name, launchPath(c), title, icon(name+".bmp"), icon(name+"_f.bmp"))
}

var reApps = regexp.MustCompile(`("applications"\s*:\s*\{)(\s*\})?`)

// patch inserts def into "applications" and key into the group's "apps" array,
// editing the text in place so the rest of the file keeps its formatting.
func patch(text, key, def, group string) (string, error) {
	m := reApps.FindStringSubmatchIndex(text)
	if m == nil {
		return "", fmt.Errorf(`"applications" object not found in view.json`)
	}
	if m[4] >= 0 { // empty object: no trailing comma
		text = text[:m[3]] + def + "\n    }" + text[m[5]:]
	} else {
		text = text[:m[3]] + def + "," + text[m[3]:]
	}

	g := regexp.MustCompile(`("title"\s*:\s*"` + regexp.QuoteMeta(group) + `"[^\]]*?"apps"\s*:\s*\[)(\s*\])?`)
	m = g.FindStringSubmatchIndex(text)
	if m == nil {
		return "", fmt.Errorf("group %q with an apps array not found in view.json", group)
	}
	item := "\n                    " + fmt.Sprintf("%q", key)
	if m[4] >= 0 {
		text = text[:m[3]] + item + "\n                ]" + text[m[5]:]
	} else {
		text = text[:m[3]] + item + "," + text[m[3]:]
	}

	var v interface{}
	if err := json.Unmarshal([]byte(text), &v); err != nil {
		return "", fmt.Errorf("patched view.json would be invalid: %w", err)
	}
	return text, nil
}

func writeIcons(c Config, name string) error {
	dir := filepath.Join(c.Root, "applications", "icons")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for suffix, focused := range map[string]bool{"": false, "_f": true} {
		p := filepath.Join(dir, name+suffix+".bmp")
		if _, err := os.Stat(p); err == nil {
			continue
		}
		f, err := os.Create(p)
		if err != nil {
			return err
		}
		err = bmp.Encode(f, c.Icon(IconSize, focused))
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return err
		}
	}
	return nil
}
