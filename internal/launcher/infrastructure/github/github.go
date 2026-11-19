// Package github is the domain.Catalog of the ecosystem: the apps published on
// the latest GitHub Release of the repository. The repository itself holds only
// source; a release carries one <id>.zip per app (the <id>.app built for the
// reader and its app.json) and a catalog.json that lists them with their
// manifests, sizes and SHA-256, so the Launcher lists the apps with one small
// download instead of one per app (tools/release builds all of it).
package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"go-pocketbook-base/internal/launcher/domain"
	platform "go-pocketbook-base/internal/platform/domain"
)

// DefaultBaseURL is GitHub's API.
const DefaultBaseURL = "https://api.github.com"

// CatalogAsset is the name of the release file that lists the apps.
const CatalogAsset = "catalog.json"

// AppAsset is the name of the release file that holds app id: a zip with
// <id>.app and app.json. (GitHub refuses release files named *.app.)
func AppAsset(id string) string { return id + ".zip" }

// Index is the content of catalog.json.
type Index struct {
	Apps []IndexEntry `json:"apps"`
}

// IndexEntry describes one app of the release.
type IndexEntry struct {
	ID       string          `json:"id"`
	Version  string          `json:"version"`
	Asset    string          `json:"asset"`  // the zip's name in the release
	Size     int64           `json:"size"`   // of the zip
	SHA256   string          `json:"sha256"` // of the zip, hex
	Manifest json.RawMessage `json:"manifest"`
}

const (
	userAgent    = "go-pocketbook-base-launcher"
	maxIndex     = 1 << 20
	maxReleaseJS = 1 << 20
	cacheFor     = 5 * time.Minute
)

// Doer is what the catalog needs of an HTTP client (*http.Client satisfies it).
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Catalog lists the apps of the latest release of Owner/Repo.
type Catalog struct {
	Client      Doer
	Owner, Repo string
	BaseURL     string           // DefaultBaseURL when empty
	Now         func() time.Time // time.Now when nil (tests)

	mu    sync.Mutex
	list  []domain.Listing
	langs string
	at    time.Time
}

type release struct {
	Tag    string `json:"tag_name"`
	Assets []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

// List implements domain.Catalog. The answer is kept for a few minutes: the
// search asks for it every time the text changes, the release does not.
func (c *Catalog) List(lang string) ([]domain.Listing, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	if c.list != nil && c.langs == lang && now.Sub(c.at) < cacheFor {
		return c.list, nil
	}
	rel, err := c.latest()
	if err != nil {
		return nil, err
	}
	urls := map[string]string{}
	for _, a := range rel.Assets {
		urls[a.Name] = a.URL
	}
	indexURL, ok := urls[CatalogAsset]
	if !ok {
		return nil, fmt.Errorf("github: release %s has no %s", rel.Tag, CatalogAsset)
	}
	raw, err := Get(c.Client, indexURL, maxIndex)
	if err != nil {
		return nil, err
	}
	var index Index
	if err := json.Unmarshal(raw, &index); err != nil {
		return nil, fmt.Errorf("github: unexpected %s: %w", CatalogAsset, err)
	}

	var out []domain.Listing
	for _, e := range index.Apps {
		url, ok := urls[e.Asset]
		m, err := platform.ParseManifest(e.Manifest)
		if !ok || err != nil || m.ID != e.ID {
			continue // missing from the release, or a manifest that does not follow the contract
		}
		out = append(out, domain.Listing{
			ID: e.ID, Title: m.TitleIn(lang), Description: m.DescriptionIn(lang),
			Version: strings.TrimPrefix(e.Version, "v"), URL: url, Size: e.Size, SHA256: e.SHA256,
			Manifest: e.Manifest,
		})
	}
	c.list, c.langs, c.at = out, lang, now
	return out, nil
}

func (c *Catalog) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c *Catalog) latest() (release, error) {
	base := c.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	body, err := Get(c.Client, fmt.Sprintf("%s/repos/%s/%s/releases/latest", base, c.Owner, c.Repo), maxReleaseJS)
	if err != nil {
		return release{}, err
	}
	var rel release
	if err := json.Unmarshal(body, &rel); err != nil {
		return release{}, fmt.Errorf("github: unexpected answer: %w", err)
	}
	return rel, nil
}

// Get is a GET that checks the status and reads at most limit bytes.
func Get(client Doer, url string, limit int64) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json, application/octet-stream")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github: %s answered %s", url, resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("github: %s is larger than %d bytes", url, limit)
	}
	return data, nil
}
