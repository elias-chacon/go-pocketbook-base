package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-pocketbook-base/internal/accounts/domain"
	"go-pocketbook-base/internal/accounts/infrastructure/oauthdevice"
)

// oauthFile is where the client id of the app registered at Microsoft goes, next to the
// accounts: {"microsoft": "<client id>"}. It is not a secret (the app is a public client).
const oauthFile = "oauth.json"

// microsoftClientID reads the client id from the data folder ("" when there is none).
func microsoftClientID(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, oauthFile))
	if err != nil {
		return ""
	}
	var cfg struct {
		Microsoft string `json:"microsoft"`
	}
	if json.Unmarshal(data, &cfg) != nil {
		return ""
	}
	return strings.TrimSpace(cfg.Microsoft)
}

// httpClient is the network client of the platform, or the standard one in tests.
func httpClient(env *http.Client) *http.Client {
	if env != nil {
		return env
	}
	return &http.Client{Timeout: 60 * time.Second}
}

// authorizer signs in to Outlook.com by code; nil while there is no client id.
func authorizer(dir string, client *http.Client) domain.Authorizer {
	id := microsoftClientID(dir)
	if id == "" {
		return nil
	}
	return oauthdevice.Client{HTTP: client, Provider: oauthdevice.Microsoft(id)}
}
