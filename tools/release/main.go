// Command release prepares what is published on a GitHub Release:
//
//   - one <id>.zip per app (its <id>.app built for the reader and its app.json) and
//     a catalog.json that lists them with manifests, sizes and SHA-256, which the
//     Launcher reads to list and install the apps (and people can unzip by hand);
//   - the installer for people (cmd/pbinstaller) for Windows, Linux and macOS,
//     each carrying the apps inside, as a zip / tar.gz;
//   - SHA256SUMS of everything.
//
// Build the apps first (.\build.ps1 -All), then:
//
//	go run ./tools/release            # writes ./release
//
// The Windows installer gets the Launcher's icon through a .syso that this tool
// writes itself (syso.go): nothing is downloaded.
package main

import (
	"fmt"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"go-pocketbook-base/internal/launcher/infrastructure/github"
	launcherui "go-pocketbook-base/internal/launcher/presentation"
)

const (
	distApps = "dist/applications"
	payload  = "cmd/pbinstaller/payload"
	outDir   = "release"
	// defaultRepo is where the release is published when GITHUB_REPOSITORY is not set.
	defaultRepo = "elias-chacon/go-pocketbook-base"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "release:", err)
		os.Exit(1)
	}
}

func run() error {
	raw, err := os.ReadFile(".version")
	if err != nil {
		return err
	}
	version := strings.TrimSpace(string(raw))

	ids, err := stage(distApps, "apps", payload, version)
	if err != nil {
		return err
	}
	fmt.Printf("version %s, apps: %s\n", version, strings.Join(ids, ", "))

	if err := os.RemoveAll(outDir); err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	var files []string
	var entries []github.IndexEntry
	for _, id := range ids { // one zip per app, for the Launcher (and people) to download
		entry, err := packageApp(distApps, "apps", outDir, id, appVersion("apps", id, version))
		if err != nil {
			return err
		}
		entries = append(entries, entry)
		files = append(files, entry.Asset)
		if entry.Icon != "" {
			files = append(files, entry.Icon)
		}
	}
	if err := writeCatalog(outDir, entries); err != nil {
		return err
	}
	files = append(files, github.CatalogAsset)

	windowsIcon()
	for _, t := range targets {
		name, err := build(t, version)
		if err != nil {
			return err
		}
		files = append(files, name)
		fmt.Println("built", name)
	}
	if err := checksums(outDir, files); err != nil {
		return err
	}
	repo := os.Getenv("GITHUB_REPOSITORY")
	if repo == "" {
		repo = defaultRepo
	}
	base := "https://github.com/" + repo + "/releases/download/v" + version
	return os.WriteFile(filepath.Join(outDir, "RELEASE_NOTES.md"), []byte(notes(base)), 0o644)
}

// windowsIcon writes the .syso that gives the Windows executable the Launcher's
// icon (see syso.go). It is a nicety: a failure only warns.
func windowsIcon() {
	data, err := sysoFor(func(size int) image.Image { return launcherui.RenderIcon(size, false) })
	if err != nil {
		fmt.Println("warning: the Windows installer gets the default icon:", err)
		return
	}
	syso := filepath.Join("cmd", "pbinstaller", "rsrc_windows_amd64.syso")
	if err := os.WriteFile(syso, data, 0o644); err != nil {
		fmt.Println("warning: the Windows installer gets the default icon:", err)
	}
}

// build compiles the installer for t and archives it in outDir; it returns the
// archive's file name.
func build(t target, version string) (string, error) {
	exePath := filepath.Join(os.TempDir(), fmt.Sprintf("pbgo-%s-%s-%s", t.OS, t.Arch, t.exe()))
	cmd := exec.Command("go", "build", "-trimpath",
		"-ldflags", "-s -w -X go-pocketbook-base/internal/kernel/version.Value="+version,
		"-o", exePath, "./cmd/pbinstaller")
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS="+t.OS, "GOARCH="+t.Arch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("%s/%s: %v\n%s", t.OS, t.Arch, err, out)
	}
	defer os.Remove(exePath)
	name := t.archive()
	return name, writeArchive(filepath.Join(outDir, name), t, t.exe(), exePath)
}
