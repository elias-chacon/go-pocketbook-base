package main

import (
	"path/filepath"

	"go-pocketbook-base/internal/accounts/infrastructure/oauthconfig"
	"go-pocketbook-base/internal/platform"
	"go-pocketbook-base/internal/showcase"
	"go-pocketbook-base/internal/toolkit/widget"
)

// hello is the app: it gives the platform its screen (platform.App) and the keys
// the platform does not handle itself (platform.KeyHandler).
type hello struct{ demo *showcase.App }

// newHello is the platform.Factory: the platform calls it once the screen size
// is known, with everything the app may need in env.
func newHello(env platform.Env) platform.App {
	return &hello{demo: showcase.New(env.Width, env.Height, showcase.Config{
		Version:  env.Version,
		Keyboard: env.Keyboard,
		Exit:     env.Exit,
		Notice:   env.Notice,
		Lang:     env.Lang,
		OAuth: func(address func() string) []widget.OAuthProvider {
			return oauthProviders(env, dataDir(platform.DefaultRoot), address)
		}, OAuthSetup: filepath.Join(dataDir(platform.DefaultRoot), oauthconfig.File),
		// The language chosen here becomes the language of every app.
		OnLangChange: func(lang string) { _ = env.Settings.Save(platform.Settings{Lang: lang}) },
	})}
}

// Document implements platform.App.
func (h *hello) Document() *widget.Document { return h.demo.Doc }

// Key implements platform.KeyHandler. The platform already used left and Menu to
// close a popup or a modal; with nothing open, left and right change the page.
func (h *hello) Key(key platform.Key) bool {
	switch {
	case key == platform.KeyPrevious:
		h.demo.MovePage(-1)
	case key == platform.KeyNext && !h.demo.Doc.HasModal():
		h.demo.MovePage(1)
	default:
		return false
	}
	return true
}
