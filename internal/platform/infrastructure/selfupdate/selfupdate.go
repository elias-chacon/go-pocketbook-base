// Package selfupdate is how an app finds and installs its own newer build: the adapter of
// domain.UpdateSource and domain.UpdateInstaller over the latest GitHub Release of the
// repository (the one the Launcher lists the apps from: a catalog.json that says, for each app,
// its version, its zip, the zip's size and SHA-256).
//
// The platform context cannot import the Launcher's catalog, so it reads the few fields it
// needs of catalog.json itself.
package selfupdate

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"go-pocketbook-base/internal/platform/domain"
)

// DefaultBaseURL is GitHub's API.
const DefaultBaseURL = "https://api.github.com"

// CatalogAsset is the name of the release file that lists the apps.
const CatalogAsset = "catalog.json"

const (
	userAgent    = "go-pocketbook-base-app"
	maxJSON      = 1 << 20
	maxApp       = 64 << 20
	elfSignature = "\x7fELF"
)

// Doer is what the adapter needs of an HTTP client (*http.Client satisfies it).
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Source implements domain.UpdateSource over the latest release of Owner/Repo.
type Source struct {
	Client      Doer
	Owner, Repo string
	BaseURL     string // DefaultBaseURL when empty
}

var _ domain.UpdateSource = Source{}

type release struct {
	Tag    string `json:"tag_name"`
	Assets []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

type index struct {
	Apps []struct {
		ID      string `json:"id"`
		Version string `json:"version"`
		Asset   string `json:"asset"`
		Size    int64  `json:"size"`
		SHA256  string `json:"sha256"`
		AppSHA  string `json:"app_sha256"`
	} `json:"apps"`
}

// Latest implements domain.UpdateSource.
func (s Source) Latest(appID string) (domain.Update, error) {
	base := s.BaseURL
	if base == "" {
		base = DefaultBaseURL
	}
	body, err := get(s.Client, fmt.Sprintf("%s/repos/%s/%s/releases/latest", base, s.Owner, s.Repo), maxJSON)
	if err != nil {
		return domain.Update{}, err
	}
	var rel release
	if err := json.Unmarshal(body, &rel); err != nil {
		return domain.Update{}, fmt.Errorf("selfupdate: unexpected answer: %w", err)
	}
	urls := map[string]string{}
	for _, a := range rel.Assets {
		urls[a.Name] = a.URL
	}
	indexURL, ok := urls[CatalogAsset]
	if !ok {
		return domain.Update{}, fmt.Errorf("selfupdate: release %s has no %s", rel.Tag, CatalogAsset)
	}
	raw, err := get(s.Client, indexURL, maxJSON)
	if err != nil {
		return domain.Update{}, err
	}
	var idx index
	if err := json.Unmarshal(raw, &idx); err != nil {
		return domain.Update{}, fmt.Errorf("selfupdate: unexpected %s: %w", CatalogAsset, err)
	}
	for _, e := range idx.Apps {
		if e.ID != appID {
			continue
		}
		url, ok := urls[e.Asset]
		if !ok {
			return domain.Update{}, fmt.Errorf("selfupdate: release %s has no %s", rel.Tag, e.Asset)
		}
		return domain.Update{Version: strings.TrimPrefix(e.Version, "v"), URL: url, Size: e.Size, SHA256: e.SHA256, AppSHA256: e.AppSHA}, nil
	}
	return domain.Update{}, fmt.Errorf("selfupdate: %s is not in release %s", appID, rel.Tag)
}

// Installer implements domain.UpdateInstaller: it writes the new build over the file of the
// running app (Exe). On the reader's Linux a running program's file may be replaced: the
// process keeps the old one until it restarts.
type Installer struct {
	Client Doer
	Exe    string // the .app being replaced
}

var _ domain.UpdateInstaller = Installer{}

// Install downloads the zip, checks its size and SHA-256 against what the release announced,
// takes <id>.app out of it and puts it in place through a temporary file, so that an
// interrupted update never leaves half an app.
func (i Installer) Install(appID string, u domain.Update) error {
	data, err := get(i.Client, u.URL, maxApp)
	if err != nil {
		return err
	}
	if err := verify(data, u); err != nil {
		return err
	}
	binary, err := unpack(data, appID)
	if err != nil {
		return err
	}
	if u.AppSHA256 != "" && !strings.EqualFold(Sum(binary), u.AppSHA256) {
		return errors.New("the app in the zip is not the one the release announced")
	}
	tmp := i.Exe + ".new"
	if err := os.MkdirAll(filepath.Dir(i.Exe), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(tmp, binary, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp, i.Exe); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// Sum is the SHA-256 of data, in hex.
func Sum(data []byte) string {
	s := sha256.Sum256(data)
	return hex.EncodeToString(s[:])
}

func verify(data []byte, u domain.Update) error {
	if u.Size > 0 && int64(len(data)) != u.Size {
		return fmt.Errorf("download incomplete: %d of %d bytes", len(data), u.Size)
	}
	if u.SHA256 != "" {
		sum := sha256.Sum256(data)
		if !strings.EqualFold(hex.EncodeToString(sum[:]), u.SHA256) {
			return errors.New("download corrupted: SHA-256 does not match the release")
		}
	}
	return nil
}

// unpack reads <id>.app from the zip and refuses anything that is not a program (an ELF
// file): the app being replaced must never be left unable to start.
func unpack(data []byte, id string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("not a valid zip: %w", err)
	}
	for _, f := range zr.File {
		if f.Name != id+".app" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		bin, err := io.ReadAll(io.LimitReader(rc, maxApp+1))
		rc.Close()
		if err != nil {
			return nil, err
		}
		if len(bin) > maxApp {
			return nil, fmt.Errorf("%s is larger than %d bytes", f.Name, maxApp)
		}
		if !bytes.HasPrefix(bin, []byte(elfSignature)) {
			return nil, fmt.Errorf("%s is not a program", f.Name)
		}
		return bin, nil
	}
	return nil, fmt.Errorf("the zip has no %s.app", id)
}

// get is a GET that checks the status and reads at most limit bytes.
func get(client Doer, url string, limit int64) ([]byte, error) {
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
		return nil, fmt.Errorf("selfupdate: %s answered %s", url, resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("selfupdate: %s is larger than %d bytes", url, limit)
	}
	return data, nil
}
