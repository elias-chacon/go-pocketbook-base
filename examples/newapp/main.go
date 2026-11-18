// examples/newapp/main.go
// Command newapp is a starting point for a new PocketBook application:
// copy this folder to cmd/<name>, change the screen and run `make dist APP=<name>`.
package main

import (
	"log"

	"go-pocketbook-base/examples"
	"go-pocketbook-base/internal/inkview"
	"go-pocketbook-base/internal/widget"
)

type app struct {
	doc     *widget.Document
	pending bool // a repaint is already queued
}

// Init runs once, when the app starts.
func (a *app) Init() {
	inkview.DisablePanel() // the app gets the whole screen
	w, h := inkview.ScreenSize()
	a.doc = examples.NewPagedApp(w, h, deviceKeyboard{}) // <- your screen goes here
	a.doc.OnInvalidate = a.requestRepaint
}

// requestRepaint queues at most one repaint, however many changes happen.
func (a *app) requestRepaint() {
	if !a.pending {
		a.pending = true
		inkview.Repaint()
	}
}

// Draw renders the document and shows it on the e-ink screen.
func (a *app) Draw() {
	a.pending = false
	img := a.doc.Render()
	b := img.Bounds()
	inkview.Blit(img.Pix, b.Dx(), b.Dy(), img.Stride)
}

// Key handles hardware keys. The PB741 has no Back key, so left and Menu close
// a popup, then a modal; Home is handled by the system (it closes the app).
func (a *app) Key(code int) bool {
	switch {
	case inkview.IsBackKey(code) && a.doc.HandleBack():
	case code == inkview.KeyBack:
		inkview.Exit() // devices that do have a Back key
	default:
		return false
	}
	return true
}

// Pointer forwards touches (device pixels) to the document.
func (a *app) Pointer(kind inkview.PointerKind, x, y int) bool {
	if kind == inkview.PointerMove && a.pending {
		return true // the screen is still catching up: skip intermediate positions
	}
	phase := map[inkview.PointerKind]widget.PointerType{
		inkview.PointerDown: widget.PointerDown,
		inkview.PointerMove: widget.PointerMove,
		inkview.PointerUp:   widget.PointerUp,
	}[kind]
	a.doc.HandlePointer(phase, float64(x), float64(y))
	return true
}

// deviceKeyboard opens the reader's on-screen keyboard.
type deviceKeyboard struct{}

func (deviceKeyboard) Request(r widget.KeyboardRequest) {
	flags := inkview.KeyboardNormal
	if r.Kind == widget.KeyboardNumeric {
		flags = inkview.KeyboardNumeric
	}
	inkview.OpenKeyboard(r.Title, r.Initial, r.MaxLen, flags, r.Done)
}

func main() {
	if err := inkview.Run(&app{}); err != nil {
		log.Fatal(err)
	}
}
