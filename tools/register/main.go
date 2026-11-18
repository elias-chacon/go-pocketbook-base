// Command register installs the apps on a reader mounted as a drive: it writes
// the icons and the home-screen entry (view.json) and records the app for the
// Launcher, so after copying the .app files and unplugging the cable the icons
// are already there. scripts/install.ps1 runs it.
//
//	go run ./tools/register -drive F: [-app launcher | -all]
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go-pocketbook-base/internal/launcher/presentation"
	"go-pocketbook-base/internal/platform"
	"go-pocketbook-base/internal/platform/domain"
	"go-pocketbook-base/internal/showcase"
	"image"
)

// icons are the apps that draw their own icon; any other app gets the default one.
var icons = map[string]domain.IconRenderer{
	"hello":    func(size int, focused bool) image.Image { return showcase.RenderIcon(size, focused) },
	"launcher": presentation.RenderIcon,
}

func main() {
	drive := flag.String("drive", "", "the reader's drive, e.g. F:")
	app := flag.String("app", "hello", "the app (a folder of ./apps)")
	all := flag.Bool("all", false, "every app of ./apps")
	flag.Parse()
	if *drive == "" {
		fmt.Fprintln(os.Stderr, "usage: register -drive F: [-app id | -all]")
		os.Exit(2)
	}
	if err := run(*drive, *app, *all); err != nil {
		fmt.Fprintln(os.Stderr, "register:", err)
		os.Exit(1)
	}
}

func run(drive, app string, all bool) error {
	root := strings.TrimRight(drive, `\/`) + `\`
	version, _ := os.ReadFile(".version")
	ids := []string{app}
	if all {
		entries, err := os.ReadDir("apps")
		if err != nil {
			return err
		}
		ids = nil
		for _, e := range entries {
			if e.IsDir() {
				ids = append(ids, e.Name())
			}
		}
	}
	for _, id := range ids {
		manifest, err := os.ReadFile(filepath.Join("apps", id, "app.json"))
		if err != nil {
			return err
		}
		changed, err := platform.RegisterOnDrive(root, manifest, icons[id], strings.TrimSpace(string(version)))
		if err != nil {
			return fmt.Errorf("%s: %w", id, err)
		}
		fmt.Printf("registered %s (view.json changed: %v)\n", id, changed)
	}
	return nil
}
