package main

import (
	"fmt"
	"os"
	"path/filepath"

	"go-pocketbook-base/internal/bible/application"
	"go-pocketbook-base/internal/bible/domain"
	"go-pocketbook-base/internal/bible/infrastructure/midvash"
	"go-pocketbook-base/internal/bible/infrastructure/sqlstore"
	"go-pocketbook-base/internal/bible/presentation"
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/platform"
	"go-pocketbook-base/internal/storage"
	"go-pocketbook-base/internal/toolkit/widget"
)

// dataDir is where the apps keep what they own on the reader.
func dataDir(root string) string { return filepath.Join(root, "applications", ".pbgo") }

// dbName is the Bible's database: apart from the productivity apps' one, because the text of a
// version takes some megabytes and is only ever added to or dropped as a whole.
const dbName = "bible.db"

// catalogBase is the address of the Bible API (a test points it at a local server).
var catalogBase = midvash.DefaultBase

// bible is the app: one document with the Bible screens.
type bible struct {
	doc    *widget.Document
	db     *storage.Store
	screen *presentation.Screen
}

// newBible is the platform.Factory.
func newBible(env platform.Env) platform.App {
	return newBibleAt(env, dataDir(platform.DefaultRoot), environSpeaker())
}

// newBibleAt builds the app over a data folder and a speech engine (the tests use a temporary
// folder and a fake one).
func newBibleAt(env platform.Env, dir string, speaker domain.Speaker) *bible {
	a := &bible{doc: widget.NewDocument(env.Width, env.Height)}
	a.doc.Keyboard = env.Keyboard
	if err := os.MkdirAll(dir, 0o755); err != nil {
		a.fail(err)
		return a
	}
	db, err := storage.Open(filepath.Join(dir, dbName))
	if err != nil {
		a.fail(err)
		return a
	}
	a.db = db
	lib, err := sqlstore.New(db)
	if err != nil {
		a.fail(err)
		return a
	}
	svc := application.Bible{Library: lib}
	if env.HTTP != nil { // without a client (tests, a desktop) the app still reads what is stored
		svc.Catalog = midvash.Client{HTTP: env.HTTP, Base: catalogBase}
	}
	screen, err := presentation.NewScreen(presentation.Config{
		Width: env.Width, Height: env.Height, Doc: a.doc, Chrome: platform.Chrome(env), Keyboard: env.Keyboard,
		Bible: svc, DeviceLang: deviceLanguage(env.Lang), Speaker: speaker, Flush: env.Flush,
	})
	if err != nil {
		a.fail(err)
		return a
	}
	a.screen = screen
	return a
}

// Document implements platform.App.
func (a *bible) Document() *widget.Document { return a.doc }

// close releases the database (the process ending does it on the reader; tests need it).
func (a *bible) close() {
	if a.db != nil {
		_ = a.db.Close()
	}
}

// Key implements platform.KeyHandler: the page keys turn the page of the text, or the page of the app.
func (a *bible) Key(k platform.Key) bool {
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
func (a *bible) fail(err error) {
	platform.Errorf("bible: %v", err)
	text := widget.NewLabel(fmt.Sprintf("%v", err))
	text.Style.Padding = geom.All(40)
	root := widget.NewColumn(0, text)
	root.Style.Background = widget.Paper
	a.doc.SetRoot(root)
}
