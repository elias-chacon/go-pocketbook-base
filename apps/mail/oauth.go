package main

import (
	"net/http"
	"time"

	"go-pocketbook-base/internal/accounts/domain"
	"go-pocketbook-base/internal/accounts/infrastructure/oauthconfig"
)

// oauthFile is the file with the client ids of the apps registered at the providers (see oauthconfig).
const oauthFile = oauthconfig.File

// httpClient is the network client of the platform, or the standard one in tests.
func httpClient(env *http.Client) *http.Client {
	if env != nil {
		return env
	}
	return &http.Client{Timeout: 60 * time.Second}
}

// authorizer signs in to Outlook.com by code; nil while there is no client id.
func authorizer(dir string, client *http.Client) domain.Authorizer {
	microsoft, _ := oauthconfig.Sources(dir, client)
	return microsoft
}
