package main

import (
	"net/http"
	"path/filepath"
	"time"

	accountsapp "go-pocketbook-base/internal/accounts/application"
	"go-pocketbook-base/internal/accounts/domain"
	"go-pocketbook-base/internal/accounts/infrastructure/jsonaccounts"
	"go-pocketbook-base/internal/accounts/infrastructure/oauthconfig"
	"go-pocketbook-base/internal/accounts/infrastructure/securefile"
	accountsui "go-pocketbook-base/internal/accounts/presentation"
	"go-pocketbook-base/internal/platform"
	"go-pocketbook-base/internal/toolkit/widget"
)

// dataDir is where the apps of the ecosystem keep what they share: the accounts and their secrets,
// and the client ids of the OAuth apps.
func dataDir(root string) string { return filepath.Join(root, "applications", ".pbgo") }

// oauthProviders are the real sign-ins of the Account page: Google by QR code and Microsoft by
// code, each enabled when oauth.json has its client id. Signing in adds the account (the address
// typed on the page, the tokens in the vault) to the ones the Mail, Tasks and Calendar apps share.
func oauthProviders(env platform.Env, dir string, address func() string) []widget.OAuthProvider {
	web := env.HTTP
	if web == nil {
		web = &http.Client{Timeout: 60 * time.Second}
	}
	microsoft, google := oauthconfig.Sources(dir, web)
	store := accountsapp.Accounts{Store: jsonaccounts.New(dir), Vault: securefile.New(dir)}
	tr := accountsui.NewTranslator(env.Lang)
	providers := accountsui.OAuthProviders(tr, accountsui.OAuthSources{Microsoft: microsoft, Google: google},
		func(id string, secret domain.Secret) error {
			_, err := store.RegisterOAuth(accountsapp.Registration{Name: "Hello", Email: address(), Provider: accountsui.PresetOf(id)}, secret)
			return accountsui.ExplainRegistration(tr, err)
		}, accountsui.ProviderGoogle, accountsui.ProviderMicrosoft)
	for i := range providers {
		providers[i].Label = map[string]string{accountsui.ProviderGoogle: "Google", accountsui.ProviderMicrosoft: "Microsoft"}[providers[i].ID]
		if start := providers[i].Start; start != nil {
			providers[i].Start = func() (widget.OAuthChallenge, error) {
				if address() == "" {
					return widget.OAuthChallenge{}, accountsui.ExplainRegistration(tr, accountsapp.ErrInvalidEmail)
				}
				return start()
			}
		}
	}
	return providers
}
