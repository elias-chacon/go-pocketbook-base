// Package catalogcache keeps the last catalog the Launcher got, on the reader, so the home screen
// shows the apps that are not installed (and their versions) at once and also when the reader is
// offline. It wraps another Catalog: List asks it, keeps the answer and, when it fails, gives the
// last answer kept instead of an error.
package catalogcache

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"go-pocketbook-base/internal/launcher/domain"
	platform "go-pocketbook-base/internal/platform/domain"
)

// ErrNothingKept is Cached's answer when no catalog was ever kept.
var ErrNothingKept = errors.New("catalogcache: no catalog kept yet")

// Catalog implements domain.Catalog over Inner, keeping the answers in the file Path.
type Catalog struct {
	Inner domain.Catalog
	Path  string

	mu sync.Mutex
}

var _ domain.Catalog = (*Catalog)(nil)

// kept is what is written: the listings without their texts, which come from the manifest in
// the language asked, so a change of language does not need the network.
type kept struct {
	ID        string          `json:"id"`
	Version   string          `json:"version"`
	URL       string          `json:"url"`
	IconURL   string          `json:"icon,omitempty"`
	Size      int64           `json:"size,omitempty"`
	SHA256    string          `json:"sha256,omitempty"`
	AppSHA256 string          `json:"app_sha256,omitempty"`
	Manifest  json.RawMessage `json:"manifest"`
}

// List implements domain.Catalog: the network's answer, or the kept one when the network fails
// (an error only when nothing was ever kept).
func (c *Catalog) List(lang string) ([]domain.Listing, error) {
	list, err := c.Inner.List(lang)
	if err == nil {
		c.save(list)
		return list, nil
	}
	if old, kerr := c.Cached(lang); kerr == nil {
		return old, nil
	}
	return nil, err
}

// Cached is the kept catalog in lang, without asking the network.
func (c *Catalog) Cached(lang string) ([]domain.Listing, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, err := os.ReadFile(c.Path)
	if err != nil {
		return nil, ErrNothingKept
	}
	var entries []kept
	if json.Unmarshal(data, &entries) != nil {
		return nil, ErrNothingKept
	}
	var out []domain.Listing
	for _, e := range entries {
		m, err := platform.ParseManifest(e.Manifest)
		if err != nil || m.ID != e.ID {
			continue
		}
		out = append(out, domain.Listing{
			ID: e.ID, Title: m.TitleIn(lang), Description: m.DescriptionIn(lang), Version: e.Version, URL: e.URL,
			IconURL: e.IconURL, Size: e.Size, SHA256: e.SHA256, AppSHA256: e.AppSHA256, Manifest: e.Manifest,
		})
	}
	return out, nil
}

func (c *Catalog) save(list []domain.Listing) {
	entries := make([]kept, len(list))
	for i, l := range list {
		entries[i] = kept{ID: l.ID, Version: l.Version, URL: l.URL, IconURL: l.IconURL, Size: l.Size,
			SHA256: l.SHA256, AppSHA256: l.AppSHA256, Manifest: l.Manifest}
	}
	data, err := json.Marshal(entries)
	if err != nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if os.MkdirAll(filepath.Dir(c.Path), 0o755) != nil {
		return
	}
	tmp := c.Path + ".tmp"
	if os.WriteFile(tmp, data, 0o644) == nil { // a cache: failing to keep it is not an error
		os.Rename(tmp, c.Path)
	}
}
