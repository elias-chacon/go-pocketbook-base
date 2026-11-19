// Command appicons draws the icon.png of the apps that draw their icon in code.
//
// Every app may ship apps/<id>/icon.png (a 128x128 picture): tools/release
// publishes it next to the app so the Launcher can show it before the app is
// installed, and the installer uses it as the app's icon on the home screen. The
// apps of this repository draw theirs with vector code (showcase.RenderIcon,
// presentation.RenderIcon); this tool turns that drawing into the file. A test
// fails when a file is out of date, so run it after changing an icon:
//
//	go run ./tools/appicons
package main

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"go-pocketbook-base/internal/launcher/presentation"
	"go-pocketbook-base/internal/platform"
	"go-pocketbook-base/internal/showcase"
)

// edge is the size of icon.png.
const edge = 128

// renderers are the apps of this repository that draw their icon in code.
var renderers = map[string]func(size int, focused bool) image.Image{
	"hello":    func(size int, focused bool) image.Image { return showcase.RenderIcon(size, focused) },
	"launcher": presentation.RenderIcon,
	"mail":     platform.DefaultIcon("E-mail"),
	"tasks":    platform.DefaultIcon("Tarefas"),
}

// iconPNG is the encoded icon of an app, or false when it has no renderer.
func iconPNG(id string) ([]byte, bool, error) {
	render, ok := renderers[id]
	if !ok {
		return nil, false, nil
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, render(edge, false)); err != nil {
		return nil, false, err
	}
	return buf.Bytes(), true, nil
}

// write puts apps/<id>/icon.png under appsDir for every app that has a renderer.
func write(appsDir string) ([]string, error) {
	var written []string
	for id := range renderers {
		data, _, err := iconPNG(id)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(appsDir, id, "icon.png"), data, 0o644); err != nil {
			return nil, err
		}
		written = append(written, id)
	}
	return written, nil
}
