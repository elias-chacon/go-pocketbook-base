package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	doc    *widget.Document
	db     *storage.Store
	screen *presentation.Screen
}

// newCalendar is the platform.Factory.
func newCalendar(env platform.Env) platform.App {
	return newCalendarAt(env, dataDir(platform.DefaultRoot), zoneName())
}

// newCalendarAt builds the app over a data folder and a time zone (the tests use
// a temporary folder).
func newCalendarAt(env platform.Env, dir, zone string) *calendar {
	a := &calendar{doc: widget.NewDocument(env.Width, env.Height)}
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
	tr := presentation.NewTranslator(env.Lang)
	agenda := application.Agenda{AccountID: localAccount, Store: store, Zone: zone}
	if _, err := agenda.EnsureCalendar(tr.T("cal.title")); err != nil {
		a.fail(err)
		return a
	}
	a.screen = presentation.NewScreen(presentation.Config{
		Width: env.Width, Height: env.Height, Doc: a.doc, Tr: tr, Agenda: agenda,
	})
	return a
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
