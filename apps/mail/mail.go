package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	accountsapp "go-pocketbook-base/internal/accounts/application"
	accounts "go-pocketbook-base/internal/accounts/domain"
	"go-pocketbook-base/internal/accounts/infrastructure/jsonaccounts"
	"go-pocketbook-base/internal/accounts/infrastructure/oauthconfig"
	"go-pocketbook-base/internal/accounts/infrastructure/securefile"
	accountsui "go-pocketbook-base/internal/accounts/presentation"
	"go-pocketbook-base/internal/kernel/geom"
	mailapp "go-pocketbook-base/internal/mail/application"
	maildomain "go-pocketbook-base/internal/mail/domain"
	"go-pocketbook-base/internal/mail/infrastructure/sqlstore"
	"go-pocketbook-base/internal/mail/presentation"
	"go-pocketbook-base/internal/platform"
	"go-pocketbook-base/internal/storage"
	"go-pocketbook-base/internal/toolkit/widget"
)

// dataDir is where the productivity apps keep what they share: the accounts, their
// secrets and the database.
func dataDir(root string) string { return filepath.Join(root, "applications", ".pbgo") }

// mail is the app: one document that shows the sign-in screen until there is a
// mail account, and the mailbox after that.
type mail struct {
	env      platform.Env
	doc      *widget.Document
	accounts accountsapp.Accounts
	db       *storage.Store
	store    *sqlstore.Store
	dir      string
	auth     accounts.Authorizer     // Microsoft
	google   accounts.CodeAuthorizer // Google
	screen   *presentation.Screen
}

// newMail is the platform.Factory.
func newMail(env platform.Env) platform.App { return newMailAt(env, dataDir(platform.DefaultRoot)) }

// newMailAt builds the app over a data folder (the tests use a temporary one).
func newMailAt(env platform.Env, dir string) *mail {
	m := &mail{env: env, dir: dir, doc: widget.NewDocument(env.Width, env.Height)}
	m.doc.Keyboard = env.Keyboard
	if err := os.MkdirAll(dir, 0o755); err != nil {
		m.fail(err)
		return m
	}
	db, err := storage.Open(filepath.Join(dir, "productivity.db"))
	if err != nil {
		m.fail(err)
		return m
	}
	m.db = db
	if m.store, err = sqlstore.New(db); err != nil {
		m.fail(err)
		return m
	}
	m.auth, m.google = oauthconfig.Sources(dir, httpClient(env.HTTP))
	m.accounts = accountsapp.Accounts{Store: jsonaccounts.New(dir), Vault: securefile.New(dir), Eraser: db}
	m.start()
	return m
}

// close releases the database (the process ending does it on the reader; tests need it).
func (m *mail) close() {
	if m.db != nil {
		_ = m.db.Close()
	}
}

// Document implements platform.App.
func (m *mail) Document() *widget.Document { return m.doc }

// Key implements platform.KeyHandler: the page keys turn the list.
func (m *mail) Key(k platform.Key) bool {
	if m.screen == nil {
		return false
	}
	switch k {
	case platform.KeyPrevious:
		return m.screen.Key(true)
	case platform.KeyNext:
		return m.screen.Key(false)
	}
	return false
}

// start shows the mailbox of the first mail account, or asks for one.
func (m *mail) start() {
	list, err := m.accounts.List(accounts.Mail)
	if err != nil {
		m.fail(err)
		return
	}
	if len(list) == 0 {
		m.signIn()
		return
	}
	m.open(list[0])
}

func (m *mail) signIn() {
	m.screen = nil
	accountsui.NewSignIn(accountsui.SignInConfig{
		Width: m.env.Width, Height: m.env.Height, Doc: m.doc, Chrome: platform.Chrome(m.env), Tr: accountsui.NewTranslator(m.env.Lang),
		Accounts: m.accounts, Authorizer: m.auth, NoGoogle: true, OAuthSetup: filepath.Join(m.dir, oauthFile), OnDone: m.open,
	})
	m.doc.ResetFrame()
}

func (m *mail) open(acc accounts.Account) {
	_, secret, err := m.accounts.Credentials(acc.ID)
	if err != nil {
		m.fail(err)
		return
	}
	renew := m.authorizerFor(acc)
	tokens := func() (string, error) { return m.accounts.AccessToken(acc.ID, renew, time.Now()) }
	backend, err := backendFor(acc, secret, tokens, httpClient(m.env.HTTP))
	if err != nil {
		m.fail(err)
		return
	}
	box := mailapp.Mailbox{
		Sync:   mailapp.Sync{AccountID: acc.ID, Backend: backend, Store: m.store},
		Me:     maildomain.Address{Name: acc.Name, Email: acc.Email},
		Drafts: m.store,
	}
	m.screen = presentation.NewScreen(presentation.Config{
		Width: m.env.Width, Height: m.env.Height, Doc: m.doc, Chrome: platform.Chrome(m.env), Tr: presentation.NewTranslator(m.env.Lang),
		Mailbox: box, Flush: m.env.Flush, Signature: acc.Signature,
		OnAccount: func() { m.editAccount(acc) },
	})
}

// authorizerFor is the sign-in that renews the tokens of an account: Google's for an account with a
// Google service, Microsoft's for the others (nil when its client id is not set).
func (m *mail) authorizerFor(acc accounts.Account) accounts.Authorizer {
	if _, ok := acc.Endpoint(accounts.Calendar, accounts.Google); ok {
		if m.google == nil {
			return nil
		}
		return m.google
	}
	return m.auth
}

// fail shows what went wrong where the person can read it, and logs it.
func (m *mail) fail(err error) {
	platform.Errorf("mail: %v", err)
	text := widget.NewLabel(fmt.Sprintf("%v", err))
	text.Style.Padding = geom.All(40)
	root := widget.NewColumn(0, text)
	root.Style.Background = widget.Paper
	m.doc.SetRoot(root)
}

// editAccount shows the settings of the account: servers, security, login, password and
// signature. Saving opens the mailbox again with them.
func (m *mail) editAccount(acc accounts.Account) {
	m.screen = nil
	accountsui.NewEditor(accountsui.EditorConfig{
		Width: m.env.Width, Height: m.env.Height, Doc: m.doc, Chrome: platform.Chrome(m.env), Tr: accountsui.NewTranslator(m.env.Lang),
		Account: acc, Manager: m.accounts, Test: m.testConnection(acc),
		OnSaved:   m.open,
		OnRemoved: m.start,
		OnBack:    func() { m.open(acc) },
	})
	m.doc.ResetFrame()
}

// testConnection connects to the server with the settings typed, and lists the folders.
func (m *mail) testConnection(acc accounts.Account) func(accounts.Account, string) error {
	return func(edited accounts.Account, password string) error {
		_, secret, err := m.accounts.Credentials(acc.ID)
		if err != nil {
			return err
		}
		if password != "" {
			secret.Password = password
		}
		backend, err := backendFor(edited, secret, nil, httpClient(m.env.HTTP))
		if err != nil {
			return err
		}
		if closer, ok := backend.(interface{ Close() }); ok {
			defer closer.Close()
		}
		_, err = backend.Folders()
		return err
	}
}
