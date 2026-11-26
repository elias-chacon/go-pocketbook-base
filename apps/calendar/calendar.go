package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	accountsapp "go-pocketbook-base/internal/accounts/application"
	accounts "go-pocketbook-base/internal/accounts/domain"
	"go-pocketbook-base/internal/accounts/infrastructure/jsonaccounts"
	"go-pocketbook-base/internal/accounts/infrastructure/oauthconfig"
	"go-pocketbook-base/internal/accounts/infrastructure/securefile"
	accountsui "go-pocketbook-base/internal/accounts/presentation"
	"go-pocketbook-base/internal/localization/i18n"

	"go-pocketbook-base/internal/calendar/application"
	"go-pocketbook-base/internal/calendar/infrastructure/sqlstore"
	"go-pocketbook-base/internal/calendar/presentation"
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/platform"
	"go-pocketbook-base/internal/storage"
	"go-pocketbook-base/internal/toolkit/widget"
)

// localAccount owns the events that live only on the reader (no server yet).
const localAccount = "local"

// dataDir is where the productivity apps keep what they share.
func dataDir(root string) string { return filepath.Join(root, "applications", ".pbgo") }

// zoneName is the IANA name of the person's clock: the TZ variable, or the zone file
// of the system when there is one. "" means UTC: the reader may know nothing better.
func zoneName() string {
	if tz := strings.TrimSpace(os.Getenv("TZ")); tz != "" {
		return strings.TrimPrefix(tz, ":")
	}
	if data, err := os.ReadFile("/etc/timezone"); err == nil {
		return strings.TrimSpace(string(data))
	}
	return ""
}

// calendar is the app: one document with the calendar screens.
type calendar struct {
	env      platform.Env
	dir      string
	doc      *widget.Document
	db       *storage.Store
	agenda   application.Agenda
	store    *sqlstore.Store
	tr       *i18n.Bundle
	accounts accountsapp.Accounts
	screen   *presentation.Screen
}

// newCalendar is the platform.Factory.
func newCalendar(env platform.Env) platform.App {
	return newCalendarAt(env, dataDir(platform.DefaultRoot), zoneName())
}

// newCalendarAt builds the app over a data folder and a time zone (the tests use
// a temporary folder).
func newCalendarAt(env platform.Env, dir, zone string) *calendar {
	a := &calendar{env: env, dir: dir, doc: widget.NewDocument(env.Width, env.Height)}
	a.doc.Keyboard = env.Keyboard
	if err := os.MkdirAll(dir, 0o755); err != nil {
		a.fail(err)
		return a
	}
	db, err := storage.Open(filepath.Join(dir, "productivity.db"))
	if err != nil {
		a.fail(err)
		return a
	}
	a.db = db
	store, err := sqlstore.New(db)
	if err != nil {
		a.fail(err)
		return a
	}
	a.store = store
	a.tr = presentation.NewTranslator(env.Lang)
	a.agenda = application.Agenda{AccountID: localAccount, Store: store, Zone: zone}
	if _, err := a.agenda.EnsureCalendar(a.tr.T("cal.title")); err != nil {
		a.fail(err)
		return a
	}
	a.accounts = accountsapp.Accounts{Store: jsonaccounts.New(dir), Vault: securefile.New(dir), Eraser: db}
	a.showScreen("")
	return a
}

// showScreen shows the calendar, with a notice at the foot when there is one.
func (a *calendar) showScreen(notice string) {
	a.screen = presentation.NewScreen(presentation.Config{
		Width: a.env.Width, Height: a.env.Height, Doc: a.doc, Chrome: platform.Chrome(a.env), Tr: a.tr, Agenda: a.agenda,
		Flush: a.env.Flush, Notice: notice,
		Importer: icsImporter{agenda: a.agenda, web: a.env.HTTP, roots: []string{platform.DefaultRoot, filepath.Join(platform.DefaultRoot, "Download")}},
		Connect:  a.connect, Account: a.account(),
	})
}

// account is the address of the account the calendar synchronizes with ("" when there is none).
func (a *calendar) account() string {
	if acc, ok := a.linked(); ok {
		return acc.Email
	}
	return ""
}

// connect opens the sign-in to a Microsoft account (the shared accounts of the mail, tasks and
// calendar apps). Google's OAuth sign-in is not offered: its restricted Gmail scope needs a
// verification only a published app can have.
func (a *calendar) connect() {
	microsoft, _ := oauthconfig.Sources(a.dir, httpClient(a.env.HTTP))
	accountsui.NewConnect(accountsui.ConnectConfig{
		Width: a.env.Width, Height: a.env.Height, Doc: a.doc, Chrome: platform.Chrome(a.env),
		Tr: accountsui.NewTranslator(a.env.Lang), Accounts: a.accounts, Microsoft: microsoft, Providers: []string{accountsui.ProviderMicrosoft},
		OAuthSetup: filepath.Join(a.dir, oauthconfig.File),
		OnDone:     func(acc accounts.Account) { a.showScreen(a.tr.T("cal.account.done", acc.Email)) },
		OnCancel:   func() { a.showScreen("") },
	})
	a.doc.ResetFrame()
}

// httpClient is the network client of the platform, or the standard one in tests.
func httpClient(env *http.Client) *http.Client {
	if env != nil {
		return env
	}
	return &http.Client{Timeout: 60 * time.Second}
}

// Document implements platform.App.
func (a *calendar) Document() *widget.Document { return a.doc }

// close releases the database (the process ending does it on the reader; tests need it).
func (a *calendar) close() {
	if a.db != nil {
		_ = a.db.Close()
	}
}

// Key implements platform.KeyHandler: the page keys turn the day's events.
func (a *calendar) Key(k platform.Key) bool {
	if a.screen == nil {
		return false
	}
	switch k {
	case platform.KeyPrevious:
		return a.screen.Key(true)
	case platform.KeyNext:
		return a.screen.Key(false)
	}
	return false
}

// fail shows what went wrong where the person can read it, and logs it.
func (a *calendar) fail(err error) {
	platform.Errorf("calendar: %v", err)
	text := widget.NewLabel(fmt.Sprintf("%v", err))
	text.Style.Padding = geom.All(40)
	root := widget.NewColumn(0, text)
	root.Style.Background = widget.Paper
	a.doc.SetRoot(root)
}
