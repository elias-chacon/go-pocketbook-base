package desktop

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// blanks are the characters between the items of view.json.
const blanks = " \t\r\n"

// Unregister is the opposite of Register: it takes the app out of view.json (its entry and its
// place in the group's apps array) and deletes its icons. It is idempotent: an app that is not
// there changes nothing. Like Register, view.json is backed up, the patched text is validated as
// JSON and the file is replaced atomically. It reports whether view.json changed.
func Unregister(root, name string) (bool, error) {
	key := "U_" + name
	var firstErr error
	for _, suffix := range []string{".bmp", "_f.bmp"} {
		err := os.Remove(filepath.Join(root, "applications", "icons", name+suffix))
		if err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = fmt.Errorf("icons: %w", err)
		}
	}
	view, err := findView(root)
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(view)
	if err != nil {
		return false, err
	}
	text := string(data)
	patched, err := withoutKey(text, key)
	if err != nil {
		return false, err
	}
	if patched == text {
		return false, firstErr
	}
	backup := fmt.Sprintf("%s.bak-%s", view, time.Now().Format("20060102-150405"))
	if err := os.WriteFile(backup, data, 0o644); err != nil {
		return false, fmt.Errorf("backup: %w", err)
	}
	tmp := view + ".tmp"
	if err := os.WriteFile(tmp, []byte(patched), 0o644); err != nil {
		return false, err
	}
	if err := os.Rename(tmp, view); err != nil {
		os.Remove(tmp)
		return false, err
	}
	return true, firstErr
}

// withoutKey removes the entry "key": {...} of the applications and every "key" of the apps
// arrays, keeping the commas right and the rest of the text as it was.
func withoutKey(text, key string) (string, error) {
	quoted := regexp.QuoteMeta(`"` + key + `"`)
	if loc := regexp.MustCompile(quoted + `\s*:\s*\{`).FindStringIndex(text); loc != nil {
		end, ok := closingBrace(text, loc[1]-1)
		if !ok {
			return "", fmt.Errorf("view.json: entry %s is not closed", key)
		}
		text = cutItem(text, loc[0], end+1)
	}
	for {
		loc := regexp.MustCompile(quoted + `\s*([,\]])`).FindStringSubmatchIndex(text)
		if loc == nil {
			break
		}
		text = cutItem(text, loc[0], loc[0]+len(`"`+key+`"`))
	}
	var v interface{}
	if err := json.Unmarshal([]byte(text), &v); err != nil {
		return "", fmt.Errorf("patched view.json would be invalid: %w", err)
	}
	return text, nil
}

// closingBrace finds the } that closes the { at text[open], skipping strings.
func closingBrace(text string, open int) (int, bool) {
	depth, inString := 0, false
	for i := open; i < len(text); i++ {
		switch c := text[i]; {
		case inString:
			if c == '\\' {
				i++
			} else if c == '"' {
				inString = false
			}
		case c == '"':
			inString = true
		case c == '{':
			depth++
		case c == '}':
			depth--
			if depth == 0 {
				return i, true
			}
		}
	}
	return 0, false
}

// cutItem removes text[from:to], an item of an object or array, with the comma that separated
// it from its neighbor (the one after it, or the one before it when it was the last).
func cutItem(text string, from, to int) string {
	after := to
	for after < len(text) && strings.ContainsRune(blanks, rune(text[after])) {
		after++
	}
	if after < len(text) && text[after] == ',' {
		return text[:from] + strings.TrimLeft(text[after+1:], blanks) // the next item takes its place
	}
	before := from
	for before > 0 && strings.ContainsRune(blanks, rune(text[before-1])) {
		before--
	}
	if before > 0 && text[before-1] == ',' {
		before--
	}
	return text[:before] + text[to:]
}
