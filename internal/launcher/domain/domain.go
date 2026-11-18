// Package domain is the heart of the Launcher context: what the home screen of
// the ecosystem shows and how it is paged, with no knowledge of the screen, of
// files or of how an app is started.
//
// Ubiquitous language:
//
//   - Tile: what the home screen shows for an installed app (icon and title).
//   - Grid: the columns and rows of tiles on one page.
//   - Page: the part of the tiles that fits one Grid.
//
// The Launcher depends on the platform context only for its vocabulary (an
// InstalledApp); how apps are found and started reaches this layer through ports.
package domain

import (
	"sort"
	"strings"

	platform "go-pocketbook-base/internal/platform/domain"
)

// Tile is what the home screen shows for an installed app, in one language.
type Tile struct {
	ID          string
	Title       string
	Description string
	Version     string
	Exe         string // absolute path of <id>.app
	IconPath    string // absolute path of the BMP icon ("" when none)
}

// TilesFrom makes the tiles of the installed apps in lang, sorted by title
// (ignoring case and accents: "Álbum" comes before "Zebra"). The app with id self (the Launcher) is not listed.
func TilesFrom(apps []platform.InstalledApp, lang, self string) []Tile {
	tiles := make([]Tile, 0, len(apps))
	for _, a := range apps {
		if a.ID == self {
			continue
		}
		tiles = append(tiles, Tile{
			ID: a.ID, Title: a.TitleIn(lang), Description: a.DescriptionIn(lang),
			Version: a.Version, Exe: a.Exe, IconPath: a.Icon,
		})
	}
	sort.SliceStable(tiles, func(i, j int) bool {
		return sortKey(tiles[i].Title) < sortKey(tiles[j].Title)
	})
	return tiles
}

// unaccent folds the accented letters of Portuguese (and the common Latin ones).
var unaccent = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a", "é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i", "ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u", "ç", "c", "ñ", "n",
)

// sortKey is the title as it is compared: lower-case and without accents.
func sortKey(title string) string { return unaccent.Replace(strings.ToLower(title)) }

// Bounds of a Grid: a page must stay usable on a 7.8" screen.
const (
	MinSide = 1
	MaxSide = 8
)

// Grid is the layout of one page of tiles.
type Grid struct{ Columns, Rows int }

// DefaultGrid is the grid of a Launcher whose layout was never changed.
var DefaultGrid = Grid{Columns: 4, Rows: 5}

// Normalized limits each side to [MinSide, MaxSide].
func (g Grid) Normalized() Grid {
	clamp := func(v, def int) int {
		if v == 0 {
			return def
		}
		if v < MinSide {
			return MinSide
		}
		if v > MaxSide {
			return MaxSide
		}
		return v
	}
	return Grid{Columns: clamp(g.Columns, DefaultGrid.Columns), Rows: clamp(g.Rows, DefaultGrid.Rows)}
}

// PerPage is how many tiles one page holds.
func (g Grid) PerPage() int {
	n := g.Normalized()
	return n.Columns * n.Rows
}

// Pages is how many pages n tiles need (at least one: an empty home is a page).
func (g Grid) Pages(n int) int {
	per := g.PerPage()
	if n <= 0 {
		return 1
	}
	return (n + per - 1) / per
}

// ClampPage keeps page inside [0, pages-1].
func ClampPage(page, pages int) int {
	switch {
	case page >= pages:
		return pages - 1
	case page < 0:
		return 0
	}
	return page
}

// Page returns the tiles of the given page (clamped), in order.
func (g Grid) Page(tiles []Tile, page int) []Tile {
	per := g.PerPage()
	page = ClampPage(page, g.Pages(len(tiles)))
	from := page * per
	if from >= len(tiles) {
		return nil
	}
	to := from + per
	if to > len(tiles) {
		to = len(tiles)
	}
	return tiles[from:to]
}

// AppLauncher is the port that starts an app. Starting may replace the running
// program (the reader has one screen), in which case Launch never returns; it
// returns an error when the app could not be started.
type AppLauncher interface {
	Launch(t Tile) error
}
