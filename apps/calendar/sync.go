package main

import (
	"errors"
	"net/http"
	"time"

	accounts "go-pocketbook-base/internal/accounts/domain"
	"go-pocketbook-base/internal/accounts/infrastructure/oauthconfig"
	"go-pocketbook-base/internal/calendar/application"
	"go-pocketbook-base/internal/calendar/domain"
	"go-pocketbook-base/internal/calendar/infrastructure/googlecal"
	"go-pocketbook-base/internal/calendar/infrastructure/graphcal"
)

// errNoService is returned for an account whose calendar the app does not know.
var errNoService = errors.New("calendar: this account has no calendar service")

// linked is the account the calendar synchronizes with: the first OAuth2 account that has a calendar
// service (Microsoft Graph or Google).
func (a *calendar) linked() (accounts.Account, bool) {
	list, err := a.accounts.List(accounts.Calendar)
	if err != nil {
		return accounts.Account{}, false
	}
	for _, acc := range list {
		if acc.Auth != accounts.OAuth2 {
			continue
		}
		_, g := acc.Endpoint(accounts.Calendar, accounts.Google)
		_, m := acc.Endpoint(accounts.Calendar, accounts.Graph)
		if g || m {
			return acc, true
		}
	}
	return accounts.Account{}, false
}

// backendFor connects an account to its calendar service.
func backendFor(acc accounts.Account, tokens func() (string, error), web *http.Client) (domain.Backend, error) {
	if _, ok := acc.Endpoint(accounts.Calendar, accounts.Graph); ok {
		return graphcal.Client{HTTP: web, Token: tokens}, nil
	}
	if _, ok := acc.Endpoint(accounts.Calendar, accounts.Google); ok {
		return googlecal.Client{HTTP: web, Token: tokens}, nil
	}
	return nil, errNoService
}

// syncFunc is the synchronization of the connected account (nil while there is none).
func (a *calendar) syncFunc() func() (string, error) {
	acc, ok := a.linked()
	if !ok {
		return nil
	}
	return func() (string, error) {
		microsoft, google := oauthconfig.Sources(a.dir, httpClient(a.env.HTTP))
		var renew accounts.Authorizer = microsoft
		if _, isGoogle := acc.Endpoint(accounts.Calendar, accounts.Google); isGoogle {
			renew = nil
			if google != nil {
				renew = google
			}
		}
		tokens := func() (string, error) { return a.accounts.AccessToken(acc.ID, renew, time.Now()) }
		backend, err := backendFor(acc, tokens, httpClient(a.env.HTTP))
		if err != nil {
			return "", err
		}
		res, err := application.Sync{AccountID: acc.ID, Store: a.store, Backend: backend}.Run()
		if err != nil {
			return "", err
		}
		return a.tr.T("cal.sync.done", res.Events, res.Calendars), nil
	}
}
