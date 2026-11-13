// Command hello is the PocketBook "Hello, Golang!" base application.
package main

import (
	"image"
	"log"
	"os"

	"go-pocketbook-base/internal/inkview"
	"go-pocketbook-base/internal/install"
	"go-pocketbook-base/internal/ui"
)

const (
	storageRoot = "/mnt/ext1"
	appTitle    = "Hello Go"
	appGroup    = "@General" // desktop group where the icon is listed
)

type app struct {
	status string // first-run message shown on screen
}

// Init turns the system panel off so the canvas is the whole screen (with the
// panel on, InkView offsets the canvas and the drawing is not centered), and
// logs the screen state before and after for diagnostics.
func (a *app) Init() {
	inkview.Logf("info before: %s", inkview.Info())
	inkview.DisablePanel()
	inkview.Logf("info after:  %s", inkview.Info())
}

func (a *app) Draw() {
	w, h := inkview.ScreenSize()
	img := ui.Render(w, h, a.status)
	inkview.Blit(img.Pix, w, h, img.Stride)
}

func (a *app) Key(code int) bool {
	if code == inkview.KeyBack {
		inkview.Exit()
		return true
	}
	return false
}

func (a *app) Pointer(x, y int) bool {
	w, h := inkview.ScreenSize()
	btn := ui.ExitButton(w, h)
	hit := image.Pt(x, y).In(btn)
	inkview.Logf("pointer up (%d,%d) screen=%dx%d exitButton=%v hit=%v", x, y, w, h, btn, hit)
	if hit {
		inkview.Exit()
		return true
	}
	return false
}

// firstRun registers the launcher icon on the desktop. PocketBook has no
// installer, so the app installs itself the first time it runs.
func firstRun() string {
	exe, err := os.Executable()
	if err != nil {
		exe = storageRoot + "/applications/hello.app"
	}
	res, err := install.Register(install.Config{
		Root:  storageRoot,
		Exe:   exe,
		Title: appTitle,
		Group: appGroup,
		Icon:  func(size int, focused bool) image.Image { return ui.RenderIcon(size, focused) },
	})
	inkview.Logf("install: %+v err=%v", res, err)
	switch {
	case err != nil:
		return "Icon setup failed: " + err.Error()
	case res.Changed:
		return "Icon registered. It shows on the home screen after the reader reloads it (e.g. when you unplug the USB cable)."
	}
	return ""
}

func main() {
	// Event log for on-device debugging; remove or comment out when done.
	inkview.DebugLog = storageRoot + "/applications/hello.log"
	wd, _ := os.Getwd()
	inkview.Logf("main start args=%q cwd=%q", os.Args, wd)

	err := inkview.Run(&app{status: firstRun()})
	inkview.Logf("Run returned err=%v", err)
	if err != nil {
		log.Fatal(err)
	}
}
