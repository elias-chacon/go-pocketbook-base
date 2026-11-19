package main

import (
	"errors"
	"net/http"
	"path/filepath"
	"time"

	accounts "go-pocketbook-base/internal/accounts/domain"
	"go-pocketbook-base/internal/accounts/infrastructure/oauthcode"
	"go-pocketbook-base/internal/accounts/infrastructure/oauthconfig"
	"go-pocketbook-base/internal/accounts/infrastructure/oauthdevice"
	accountsui "go-pocketbook-base/internal/accounts/presentation"
	"go-pocketbook-base/internal/platform"
	"go-pocketbook-base/internal/tasks/application"
	"go-pocketbook-base/internal/tasks/domain"
	"go-pocketbook-base/internal/tasks/infrastructure/googletasks"
	"go-pocketbook-base/internal/tasks/infrastructure/graphtodo"
	"go-pocketbook-base/internal/tasks/presentation"
)

// errNoService is returned for an account that has no to-do service the app knows.
var errNoService = errors.New("tasks: this account has no to-do service")

// httpClient is the network client of the platform, or the standard one in tests.
func httpClient(env *http.Client) *http.Client {
	if env != nil {
		return env
	}
	return &http.Client{Timeout: 60 * time.Second}
}

// authorizers builds the sign-in of each provider that has a client id.
type authorizers struct {
	microsoft accounts.Authorizer
	google    accounts.CodeAuthorizer
}

func newAuthorizers(dir string, web *http.Client) authorizers {
	cfg := oauthconfig.Load(dir)
	var a authorizers
	if cfg.Microsoft != "" {
		a.microsoft = oauthdevice.Client{HTTP: web, Provider: oauthdevice.Microsoft(cfg.Microsoft)}
	}
	if cfg.Google != "" {
		a.google = oauthcode.Client{HTTP: web, Provider: oauthcode.Google(cfg.Google, cfg.GoogleSecret)}
	}
	return a
}

// forAccount is the authorizer that renews the tokens of an account.
func (a authorizers) forAccount(acc accounts.Account) accounts.Authorizer {
	if _, ok := acc.Endpoint(accounts.Tasks, accounts.Google); ok {
		if a.google == nil {
			return nil
		}
		return a.google
	}
	return a.microsoft
}

// backendFor connects an account to its to-do service.
func backendFor(acc accounts.Account, tokens func() (string, error), web *http.Client) (domain.Backend, error) {
	if _, ok := acc.Endpoint(accounts.Tasks, accounts.Graph); ok {
		return graphtodo.Client{HTTP: web, Token: tokens}, nil
	}
	if _, ok := acc.Endpoint(accounts.Tasks, accounts.Google); ok {
		return googletasks.Client{HTTP: web, Token: tokens}, nil
	}
	return nil, errNoService
}

// linked is the account the tasks are synchronized with: the first one that gives tasks
// through OAuth2 (the Outlook account of the mail app counts, no new sign-in needed).
func (a *tasks) linked() (accounts.Account, bool) {
	l := readLink(a.dir)
	if l.Off {
		return accounts.Account{}, false
	}
	list, err := a.accounts.List(accounts.Tasks)
	if err != nil {
		return accounts.Account{}, false
	}
	for _, acc := range list {
		if acc.Auth != accounts.OAuth2 || (l.Account != "" && acc.ID != l.Account) {
			continue
		}
		_, g := acc.Endpoint(accounts.Tasks, accounts.Google)
		_, m := acc.Endpoint(accounts.Tasks, accounts.Graph)
		if g || m {
			return acc, true
		}
	}
	return accounts.Account{}, false
}

// service is the presentation.Syncer of the app.
type service struct {
	app *tasks
}

// Account implements presentation.Syncer.
func (s service) Account() string {
	if acc, ok := s.app.linked(); ok {
		return acc.Email
	}
	return ""
}

// Run implements presentation.Syncer.
func (s service) Run() (application.SyncResult, error) {
	acc, ok := s.app.linked()
	if !ok {
		return application.SyncResult{}, errNoService
	}
	auth := s.app.auth.forAccount(acc)
	tokens := func() (string, error) { return s.app.accounts.AccessToken(acc.ID, auth, time.Now()) }
	backend, err := backendFor(acc, tokens, httpClient(s.app.env.HTTP))
	if err != nil {
		return application.SyncResult{}, err
	}
	return application.Sync{AccountID: localAccount, Store: s.app.store.Raw(), Log: s.app.store, Backend: backend}.Run()
}

// Connect implements presentation.Syncer: the sign-in to Microsoft To Do or Google Tasks.
func (s service) Connect() {
	a := s.app
	accountsui.NewConnect(accountsui.ConnectConfig{
		Width: a.env.Width, Height: a.env.Height, Doc: a.doc, Chrome: platform.Chrome(a.env),
		Tr: accountsui.NewTranslator(a.env.Lang), Accounts: a.accounts,
		Microsoft: a.auth.microsoft, Google: a.auth.google,
		OAuthSetup: filepath.Join(a.dir, oauthconfig.File),
		OnDone: func(acc accounts.Account) {
			_ = writeLink(a.dir, link{Account: acc.ID})
			a.showScreen(true)
		},
		OnCancel: func() { a.showScreen(false) },
	})
	a.doc.ResetFrame()
}

// Disconnect implements presentation.Syncer: the tasks stay on the reader, and so does the
// account (the mail app may use it); the synchronization is turned off.
func (s service) Disconnect() error {
	return writeLink(s.app.dir, link{Off: true})
}

var _ presentation.Syncer = service{}
