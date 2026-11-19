// Command pbinstaller is the installer of the ecosystem for the person who only
// wants the apps on the PocketBook: double-click it with the reader connected by
// USB and it copies the apps (the Launcher and the others) to the reader and puts
// their icons on the home screen. It runs on Windows, Linux and macOS, and
// carries the apps inside its own executable: the release builds one per system
// (see tools/release).
//
//	pbinstaller                 # finds the reader by itself
//	pbinstaller -drive F:\      # the reader's drive, when it is not found
//
// It is the composition root of the installer context.
package main

import (
	"embed"
	"flag"
	"image"
	"io/fs"
	"os"

	"go-pocketbook-base/internal/installer/application"
	"go-pocketbook-base/internal/installer/domain"
	"go-pocketbook-base/internal/installer/infrastructure/drives"
	"go-pocketbook-base/internal/installer/infrastructure/embedded"
	"go-pocketbook-base/internal/installer/infrastructure/locale"
	"go-pocketbook-base/internal/installer/infrastructure/readerfs"
	"go-pocketbook-base/internal/installer/presentation"
	"go-pocketbook-base/internal/kernel/version"
	"go-pocketbook-base/internal/platform"
)

// payload is what tools/release stages: <id>.app, <id>.json (the manifest), the
// optional <id>.png (the icon) and VERSION.
//
//go:embed payload
var payload embed.FS

// iconOf is the icon an app ships (its icon.png, carried in the payload); an app
// without one, or with one that cannot be read, gets the default icon (nil).
func iconOf(p domain.Package) func(size int, focused bool) image.Image {
	if len(p.Icon) == 0 {
		return nil
	}
	icon, err := platform.IconFromPNG(p.Icon)
	if err != nil {
		return nil
	}
	return icon
}

func main() {
	drive := flag.String("drive", "", "the reader's drive or folder, when it is not found by itself")
	noPause := flag.Bool("no-pause", false, "do not wait for Enter at the end")
	flag.Parse()

	files, err := fs.Sub(payload, "payload")
	if err != nil {
		panic(err) // the folder is embedded: this is a build mistake
	}
	console := presentation.Console{
		In: os.Stdin, Out: os.Stdout, NoPause: *noPause, Version: version.Value,
		Tr: presentation.NewTranslator(language()),
		Install: application.Install{
			Finder: drives.Finder{},
			Source: embedded.Source{FS: files},
			Deployer: readerfs.Deployer{Register: func(root string, p domain.Package) (bool, error) {
				return platform.RegisterOnDrive(root, p.Manifest, iconOf(p), p.Version)
			}},
		},
	}
	os.Exit(console.Run(*drive))
}

// language is the computer's, or Portuguese (the project's primary language).
func language() string {
	if tag := locale.Detect(os.Getenv); tag != "" {
		return tag
	}
	return "pt-BR"
}
