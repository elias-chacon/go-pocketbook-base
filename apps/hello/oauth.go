package main

import (
	"net/http"
	"path/filepath"
	"time"

	"go-pocketbook-base/internal/accounts/domain"
	"go-pocketbook-base/internal/accounts/infrastructure/oauthconfig"
	accountsui "go-pocketbook-base/internal/accounts/presentation"
	"go-pocketbook-base/internal/platform"
	"go-pocketbook-base/internal/toolkit/widget"
)

// dataDir is where the apps of the ecosystem keep what they share, the client ids of the OAuth
// apps among it.
func dataDir(root string) string { return filepath.Join(root, "applications", ".pbgo") }

// oauthProviders are the real sign-ins of the Account page: Google by QR code and Microsoft by
// code, each enabled when oauth.json has its client id. The demo only proves that the sign-in
// works: it keeps no token and adds no account.
func oauthProviders(env platform.Env, dir string) []widget.OAuthProvider {
	web := env.HTTP
	if web == nil {
		web = &http.Client{Timeout: 60 * time.Second}
	}
	microsoft, google := oauthconfig.Sources(dir, web)
	providers := accountsui.OAuthProviders(accountsui.NewTranslator(env.Lang),
		accountsui.OAuthSources{Microsoft: microsoft, Google: google},
		func(string, domain.Secret) error { return nil }, accountsui.ProviderGoogle, accountsui.ProviderMicrosoft)
	for i := range providers {
		providers[i].Label = map[string]string{accountsui.ProviderGoogle: "Google", accountsui.ProviderMicrosoft: "Microsoft"}[providers[i].ID]
	}
	return providers
}
