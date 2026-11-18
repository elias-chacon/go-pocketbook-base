package main

import (
	"image"
	"os"

	"go-pocketbook-base/internal/launcher/application"
	"go-pocketbook-base/internal/launcher/domain"
	"go-pocketbook-base/internal/launcher/infrastructure/processlauncher"
	"go-pocketbook-base/internal/launcher/presentation"
	"go-pocketbook-base/internal/platform"
	"go-pocketbook-base/internal/toolkit/gfx"
	"go-pocketbook-base/internal/toolkit/widget"
)

// iconEdge is the size of the icons the launcher asks for (they are written at 128).
const iconEdge = 256

// launcher is the app: the platform's Env wired to the launcher context.
type launcher struct{ home *presentation.Home }

// newLauncher is the platform.Factory.
func newLauncher(env platform.Env) platform.App {
	self, err := os.Executable()
	if err != nil {
		self = "" // an app started by us then cannot find its way back: it just ends
	}
	return &launcher{home: presentation.NewHome(presentation.Config{
		Width: env.Width, Height: env.Height, Keyboard: env.Keyboard,
		Tr:   presentation.NewTranslator(env.Lang),
		Grid: domain.DefaultGrid,
		Home: application.Home{
			Apps:     env.Apps,
			Launcher: processlauncher.Launcher{Self: self},
			SelfID:   env.Manifest.ID,
		},
		IconFor: iconFor,
		Notice:  env.Notice,
	})}
}

// Document implements platform.App.
func (l *launcher) Document() *widget.Document { return l.home.Doc }

// Key implements platform.KeyHandler. The platform already used left and Menu to
// close a popup or a modal; with nothing open, left and right change the page.
func (l *launcher) Key(key platform.Key) bool {
	switch key {
	case platform.KeyPrevious:
		l.home.PrevPage()
	case platform.KeyNext:
		l.home.NextPage()
	default:
		return false
	}
	return true
}

// iconFor loads the BMP icon an app wrote when it first ran, or draws the
// default one (the initial of the title) when it is missing or unreadable.
func iconFor(t domain.Tile) image.Image {
	if t.IconPath != "" {
		if img, err := gfx.LoadImage(t.IconPath, iconEdge, iconEdge); err == nil {
			return img
		}
	}
	return platform.DefaultIcon(t.Title)(128, false)
}
