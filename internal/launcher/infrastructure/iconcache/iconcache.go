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

// Cache downloads icons through Client and keeps them in Dir.
type Cache struct {
	Client github.Doer
	Dir    string
}

// Image is the icon of a listing: from the cache when it is there, otherwise
// downloaded and kept. A version change is a new file, so a new icon shows up.
func (c Cache) Image(l domain.Listing) (image.Image, error) {
	if l.IconURL == "" {
		return nil, ErrNoIcon
	}
	path := filepath.Join(c.Dir, l.ID+"-"+l.Version+".png")
	if data, err := os.ReadFile(path); err == nil {
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
		os.WriteFile(path, data, 0o644) // the cache is an optimization: a failure to keep it is not an error
	}
	return img, nil
}
