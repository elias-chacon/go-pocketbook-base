// Package oauthconfig reads the client ids of the apps registered at the providers, from
// oauth.json next to the accounts:
//
//	{"microsoft": "<client id>", "google": "<client id>", "google_secret": "<client secret>"}
//
// None of them is a secret in the sense of a password: they identify the app, which is a
// public client (a desktop application), and Google asks for its "secret" anyway.
package oauthconfig

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"go-pocketbook-base/internal/accounts/domain"
	"go-pocketbook-base/internal/accounts/infrastructure/oauthcode"
	"go-pocketbook-base/internal/accounts/infrastructure/oauthdevice"
)

// File is the name of the file.
const File = "oauth.json"

// Config holds the client ids ("" for a provider that is not set).
type Config struct {
	Microsoft    string `json:"microsoft"`
	Google       string `json:"google"`
	GoogleSecret string `json:"google_secret"`
}

// Load reads the file of a data folder; a missing or broken file gives no providers.
func Load(dir string) Config {
	data, err := os.ReadFile(filepath.Join(dir, File))
	if err != nil {
		return Config{}
	}
	var c Config
	if json.Unmarshal(data, &c) != nil {
		return Config{}
	}
	c.Microsoft, c.Google, c.GoogleSecret = strings.TrimSpace(c.Microsoft), strings.TrimSpace(c.Google), strings.TrimSpace(c.GoogleSecret)
	return c
}

// Sources builds the sign-in of each provider that has a client id in the data folder: Microsoft
// by device code, Google by authorization code. A provider that is not set is nil (the apps say
// how to set it). web is the network client of the platform.
func Sources(dir string, web *http.Client) (microsoft domain.Authorizer, google domain.CodeAuthorizer) {
	cfg := Load(dir)
	if cfg.Microsoft != "" {
		microsoft = oauthdevice.Client{HTTP: web, Provider: oauthdevice.Microsoft(cfg.Microsoft)}
	}
	if cfg.Google != "" {
		google = oauthcode.Client{HTTP: web, Provider: oauthcode.Google(cfg.Google, cfg.GoogleSecret)}
	}
	return microsoft, google
}
