// Package iconcache fetches the icons of the apps of the catalog (small PNGs
// published next to each app) and keeps them on the reader, so a search shows an
// app's icon before it is installed and the installer can use the same picture
// for the app's icon on the home screen, without asking GitHub twice.
package iconcache

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"go-pocketbook-base/internal/launcher/domain"
	"go-pocketbook-base/internal/launcher/infrastructure/github"
)

// maxIcon bounds a download: an icon is a few kilobytes.
const maxIcon = 256 << 10

// ErrNoIcon is returned for an app that publishes no icon.
var ErrNoIcon = errors.New("iconcache: the app has no icon")

// Cache downloads icons through Client and keeps them in Dir: one file per app (<id>.png) and
// beside it the address it came from (<id>.url), so a new release replaces the old picture instead
// of adding another one, and Prune drops what the catalog no longer lists.
type Cache struct {
	Client github.Doer
	Dir    string
}

// Cached is the icon of a listing when one is kept (it may be of an older release: Image replaces
// it): it never uses the network, so a screen can draw many of them without waiting for one.
func (c Cache) Cached(l domain.Listing) (image.Image, bool) {
	data, err := os.ReadFile(c.path(l))
	if err != nil {
		return nil, false
	}
	img, err := png.Decode(bytes.NewReader(data))
	return img, err == nil
}

func (c Cache) path(l domain.Listing) string { return filepath.Join(c.Dir, l.ID+".png") }

// urlPath is where the address of the kept picture is written.
func (c Cache) urlPath(l domain.Listing) string { return filepath.Join(c.Dir, l.ID+".url") }

// fresh reports whether the kept picture came from the address the listing has now (every release
// publishes its icons under its own address).
func (c Cache) fresh(l domain.Listing) bool {
	got, err := os.ReadFile(c.urlPath(l))
	return err == nil && string(got) == l.IconURL
}

// Image is the icon of a listing: from the cache when it is there and came from the same address,
// otherwise downloaded and kept in the place of the old one.
func (c Cache) Image(l domain.Listing) (image.Image, error) {
	if l.IconURL == "" {
		return nil, ErrNoIcon
	}
	path := c.path(l)
	if data, err := os.ReadFile(path); err == nil && c.fresh(l) {
		if img, err := png.Decode(bytes.NewReader(data)); err == nil {
			return img, nil
		}
		// a damaged file: fetch it again
	}
	data, err := github.Get(c.Client, l.IconURL, maxIcon)
	if err != nil {
		return nil, err
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if os.MkdirAll(c.Dir, 0o755) == nil {
		// the cache is an optimization: a failure to keep it is not an error
		if os.WriteFile(path, data, 0o644) == nil {
			os.WriteFile(c.urlPath(l), []byte(l.IconURL), 0o644)
		}
	}
	return img, nil
}

// Prune deletes every kept picture that is not of one of the listings: what an older release
// listed and the files an older version of this cache named by version. It only touches .png and
// .url files, and does nothing for an empty list (a catalog that did not come must not empty it).
func (c Cache) Prune(keep []domain.Listing) {
	if len(keep) == 0 {
		return
	}
	wanted := map[string]bool{}
	for _, l := range keep {
		wanted[l.ID+".png"], wanted[l.ID+".url"] = true, true
	}
	entries, err := os.ReadDir(c.Dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		ext := filepath.Ext(e.Name())
		if !e.IsDir() && (ext == ".png" || ext == ".url") && !wanted[e.Name()] {
			os.Remove(filepath.Join(c.Dir, e.Name()))
		}
	}
}
