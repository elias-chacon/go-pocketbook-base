package main

import (
	"go-pocketbook-base/internal/appkit"
	"go-pocketbook-base/internal/demo"
	"go-pocketbook-base/internal/inkview"
	"go-pocketbook-base/internal/widget"
)

// hello is the app: it gives the kit its screen (appkit.App) and the keys the
// kit does not handle itself (appkit.KeyHandler).
type hello struct{ demo *demo.App }

// newHello is the appkit.Factory: the kit calls it once the screen size is
// known, with everything the app may need in env.
func newHello(env appkit.Env) appkit.App {
	return &hello{demo: demo.New(env.Width, env.Height, demo.Config{
		Version:  env.Version,
		Keyboard: env.Keyboard,
		Exit:     env.Exit,
		Notice:   env.Notice,
		Lang:     env.Lang,
		// The language chosen here becomes the language of every app.
		OnLangChange: func(lang string) { _ = appkit.SaveSettings(env.Root, appkit.Settings{Lang: lang}) },
	})}
}

// Document implements appkit.App.
func (h *hello) Document() *widget.Document { return h.demo.Doc }

// Key implements appkit.KeyHandler. The kit already used left and Menu to close a
// popup or a modal; with nothing open, left and right change the page.
func (h *hello) Key(code int) bool {
	switch {
	case code == inkview.KeyPrev:
		h.demo.MovePage(-1)
	case code == inkview.KeyNext && !h.demo.Doc.HasModal():
		h.demo.MovePage(1)
	default:
		return false
	}
	return true
}
