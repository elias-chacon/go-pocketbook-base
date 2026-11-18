// Package application holds the use cases of the Launcher: which tiles the home
// screen shows and what happens when one is tapped. It talks to the world only
// through the ports of the platform and launcher domains.
package application

import (
	"go-pocketbook-base/internal/launcher/domain"
	platform "go-pocketbook-base/internal/platform/domain"
)

// Home is the use case "the home screen": list the installed apps and open one.
type Home struct {
	Apps     platform.AppRegistry // where the installed apps are recorded
	Launcher domain.AppLauncher   // how an app is started
	SelfID   string               // the Launcher's own id: it does not list itself
}

// Tiles are the installed apps as tiles, in lang.
func (h Home) Tiles(lang string) []domain.Tile {
	return domain.TilesFrom(h.Apps.List(), lang, h.SelfID)
}

// Open starts the app of a tile. When the platform replaces this program with it
// the call never returns; an error means the app could not be started.
func (h Home) Open(t domain.Tile) error { return h.Launcher.Launch(t) }
