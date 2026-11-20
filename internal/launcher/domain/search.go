package domain

import (
	"errors"
	"sort"
	"strings"

	platform "go-pocketbook-base/internal/platform/domain"
)

// Listing is an app of the catalog (what can be installed), in one language.
type Listing struct {
	ID          string
	Title       string
	Description string
	Version     string
	URL         string // where the app's zip (<id>.app + app.json) is downloaded from
	IconURL     string // where its icon (a PNG) is, "" when the app has none
	Size        int64  // its size in bytes (0 when unknown)
	SHA256      string // hex digest of the download ("" when unknown)
	// AppSHA256 is the digest of the <id>.app of this build ("" when the release does not say): an
	// installed app with another digest is not the published one, whatever its version says.
	AppSHA256 string
	Manifest  []byte // the app.json, kept to register the app once installed
}

// Catalog is the port to the apps that exist but may not be installed (the
// repository's releases). It returns the listings in lang.
type Catalog interface {
	List(lang string) ([]Listing, error)
}

// Installer is the port that downloads and installs an app of the catalog on
// this reader (its file and its registration on the home screen).
type Installer interface {
	Install(l Listing) error
}

// Uninstaller is the port that removes an app and everything it left on the reader.
type Uninstaller interface {
	Uninstall(appID string) error
}

// ErrNoCatalog is the error of a Launcher that has no catalog configured.
var ErrNoCatalog = errors.New("launcher: no catalog configured")

// Matches reports whether every word of query appears in one of the fields,
// ignoring case and accents. An empty query matches everything.
func Matches(query string, fields ...string) bool {
	text := sortKey(strings.Join(fields, " "))
	for _, word := range strings.Fields(sortKey(query)) {
		if !strings.Contains(text, word) {
			return false
		}
	}
	return true
}

// FilterTiles keeps the tiles that match query (title, description and id).
func FilterTiles(tiles []Tile, query string) []Tile {
	var out []Tile
	for _, t := range tiles {
		if Matches(query, t.Title, t.Description, t.ID) {
			out = append(out, t)
		}
	}
	return out
}

// AppsOf is every app the person can see in the ecosystem, in lang, sorted by title: the installed
// ones, and the ones of the catalog that are not installed yet (so the home screen is the whole
// ecosystem and a tap on a missing app installs it). An installed app the catalog also has keeps
// its listing, which is what an update is made from. The Launcher (self) is not listed.
func AppsOf(installed []platform.InstalledApp, listings []Listing, lang, self string) []Tile {
	tiles := TilesFrom(installed, lang, self)
	byID := make(map[string]int, len(tiles))
	for i := range tiles {
		tiles[i].Installed = true
		byID[tiles[i].ID] = i
	}
	for i := range listings {
		l := listings[i]
		if l.ID == self {
			continue
		}
		if n, ok := byID[l.ID]; ok {
			tiles[n].Listing = &l
			continue
		}
		tiles = append(tiles, Tile{ID: l.ID, Title: l.Title, Description: l.Description, Version: l.Version, Listing: &l})
	}
	sort.SliceStable(tiles, func(i, j int) bool { return sortKey(tiles[i].Title) < sortKey(tiles[j].Title) })
	return tiles
}

// Visible keeps the tiles whose id is not hidden.
func Visible(tiles []Tile, hidden []string) []Tile {
	return partition(tiles, hidden, false)
}

// Hidden keeps the tiles whose id is hidden.
func Hidden(tiles []Tile, hidden []string) []Tile {
	return partition(tiles, hidden, true)
}

func partition(tiles []Tile, hidden []string, want bool) []Tile {
	set := make(map[string]bool, len(hidden))
	for _, id := range hidden {
		set[id] = true
	}
	var out []Tile
	for _, t := range tiles {
		if set[t.ID] == want {
			out = append(out, t)
		}
	}
	return out
}
