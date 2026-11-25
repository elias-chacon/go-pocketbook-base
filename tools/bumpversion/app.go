package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"go-pocketbook-base/internal/kernel/version"
)

// notePrefix starts the second line of an app .version: the fingerprint of the sources the version
// was given to.
const notePrefix = "sources "

// pkg is the part of `go list -json` this tool reads.
type pkg struct {
	Dir        string
	Standard   bool
	Module     *struct{ Main bool }
	GoFiles    []string
	CgoFiles   []string
	CFiles     []string
	HFiles     []string
	SFiles     []string
	EmbedFiles []string
}

// files are the source files of a package of this module (not the standard library, which belongs to
// the build image; not the dependencies, which go.sum covers).
func (p pkg) files() []string {
	if p.Standard || p.Module == nil || !p.Module.Main {
		return nil
	}
	var out []string
	for _, group := range [][]string{p.GoFiles, p.CgoFiles, p.CFiles, p.HFiles, p.SFiles, p.EmbedFiles} {
		for _, f := range group {
			out = append(out, filepath.Join(p.Dir, f))
		}
	}
	return out
}

// listPackages runs `go list -deps -json` for the app the way the build sees it (linux/arm, cgo), so
// the file set does not depend on the machine that asks.
func listPackages(root, app string) ([]pkg, error) {
	cmd := exec.Command("go", "list", "-deps", "-json", "./apps/"+app)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=arm", "CGO_ENABLED=1")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list %s: %v\n%s", app, err, stderr.String())
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	var pkgs []pkg
	for {
		var p pkg
		if err := dec.Decode(&p); err == io.EOF {
			return pkgs, nil
		} else if err != nil {
			return nil, err
		}
		pkgs = append(pkgs, p)
	}
}

// fingerprint hashes what makes the app binary: the files of every package of the module it uses,
// plus go.mod and go.sum. Line endings do not count (Windows checks files out as CRLF).
func fingerprint(root string, files []string) (string, error) {
	rel := map[string]bool{"go.mod": true, "go.sum": true}
	for _, f := range files {
		r, err := filepath.Rel(root, f)
		if err != nil {
			return "", err
		}
		rel[filepath.ToSlash(r)] = true
	}
	names := make([]string, 0, len(rel))
	for n := range rel {
		names = append(names, n)
	}
	sort.Strings(names)
	h := sha256.New()
	for _, n := range names {
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(n)))
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "%s\n%s\n", n, bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")))
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// sourcesOf is the fingerprint of an app, from its real packages.
func sourcesOf(root, app string) (string, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	pkgs, err := listPackages(abs, app)
	if err != nil {
		return "", err
	}
	var files []string
	for _, p := range pkgs {
		files = append(files, p.files()...)
	}
	return fingerprint(abs, files)
}

// recorded reads the fingerprint a .version file was stamped with ("" when it has none).
func recorded(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r", ""), "\n")
	for _, line := range lines[1:] {
		if strings.HasPrefix(line, notePrefix) {
			return strings.TrimPrefix(line, notePrefix)
		}
	}
	return ""
}

// stamp writes version and fingerprint to path.
func stamp(path string, v version.Version, sources string) error {
	return os.WriteFile(path, []byte(v.String()+"\n"+notePrefix+sources+"\n"), 0o644)
}

// bumpApp moves the version of apps/<id>/.version. "auto" bumps BUILD only when the sources of the
// app changed since the version was given (an app that did not change keeps its version, and with it
// the update check has nothing to offer); "check" only reports it and writes nothing.
func bumpApp(root, app, kind string, sources func(root, app string) (string, error)) (v version.Version, changed bool, err error) {
	path := filepath.Join(root, "apps", app, ".version")
	cur, err := version.ReadFile(path)
	if err != nil {
		return cur, false, err
	}
	now, err := sources(root, app)
	if err != nil {
		return cur, false, err
	}
	changed = now != recorded(path)
	switch kind {
	case "check":
		return cur, changed, nil
	case "auto":
		if !changed {
			return cur, false, nil
		}
		kind = string(version.Build)
	}
	next, err := cur.Bump(version.Kind(kind))
	if err != nil {
		return cur, changed, err
	}
	return next, true, stamp(path, next, now)
}
