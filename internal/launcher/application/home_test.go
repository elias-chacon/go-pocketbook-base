package application

import (
	"errors"
	"testing"

	"go-pocketbook-base/internal/launcher/domain"
	platform "go-pocketbook-base/internal/platform/domain"
)

type registry struct{ apps []platform.InstalledApp }

func (r registry) Save(platform.InstalledApp) error { return nil }
func (r registry) List() []platform.InstalledApp    { return r.apps }

type launcher struct {
	opened []domain.Tile
	err    error
}

func (l *launcher) Launch(t domain.Tile) error { l.opened = append(l.opened, t); return l.err }

func installed(id, title string) platform.InstalledApp {
	return platform.InstalledApp{
		Manifest: platform.Manifest{ID: id, Title: map[string]string{"pt-BR": title}, Description: map[string]string{"pt-BR": "d"}},
		Exe:      "/apps/" + id + ".app",
	}
}

func TestHomeListsTheInstalledAppsExceptItself(t *testing.T) {
	h := Home{
		Apps:   registry{apps: []platform.InstalledApp{installed("launcher", "Launcher"), installed("hello", "Hello"), installed("books", "Books")}},
		SelfID: "launcher",
	}
	tiles := h.Tiles("pt-BR")
	if len(tiles) != 2 || tiles[0].ID != "books" || tiles[1].ID != "hello" {
		t.Errorf("tiles = %+v", tiles)
	}
}

func TestHomeOpensAnAppThroughThePortAndReportsFailure(t *testing.T) {
	l := &launcher{}
	h := Home{Apps: registry{}, Launcher: l}
	tile := domain.Tile{ID: "hello", Exe: "/apps/hello.app"}
	if err := h.Open(tile); err != nil || len(l.opened) != 1 || l.opened[0] != tile {
		t.Fatalf("opened = %+v err=%v", l.opened, err)
	}
	l.err = errors.New("exec format error")
	if err := h.Open(tile); err == nil {
		t.Error("a failure to start must reach the caller")
	}
}
