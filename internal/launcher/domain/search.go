package domain

import (
	"errors"
	"strings"
)

// Listing is an app of the catalog (what can be installed), in one language.
type Listing struct {
	ID          string
	Title       string
	Description string
	Version     string
	URL         string // where the app's zip (<id>.app + app.json) is downloaded from
	Size        int64  // its size in bytes (0 when unknown)
	SHA256      string // hex digest of the download ("" when unknown)
	Manifest    []byte // the app.json, kept to register the app once installed
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

// ErrNoCatalog is the error of a Launcher that has no catalog configured.
var ErrNoCatalog = errors.New("launcher: no catalog configured")

// SearchResult is the answer to a search: the installed apps that match and the
// apps of the catalog that match and are not installed yet. CatalogErr tells why
// the second list may be missing (no network, no catalog); it is not fatal.
type SearchResult struct {
	Installed  []Tile
	Available  []Listing
	CatalogErr error
}

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

// FilterListings keeps the listings that match query and are not in installed
// (a list of ids). Their order is kept.
func FilterListings(list []Listing, query string, installed []string) []Listing {
	have := make(map[string]bool, len(installed))
	for _, id := range installed {
		have[id] = true
	}
	var out []Listing
	for _, l := range list {
		if !have[l.ID] && Matches(query, l.Title, l.Description, l.ID) {
			out = append(out, l)
		}
	}
	return out
}
