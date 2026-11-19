package main

import (
	"fmt"
	"os"
	"path/filepath"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/platform"
	"go-pocketbook-base/internal/storage"
	"go-pocketbook-base/internal/tasks/application"
	"go-pocketbook-base/internal/tasks/infrastructure/sqlstore"
	"go-pocketbook-base/internal/tasks/presentation"
	"go-pocketbook-base/internal/toolkit/widget"
)

// localAccount owns the tasks that live only on the reader (no server yet).
const localAccount = "local"

// dataDir is where the productivity apps keep what they share.
func dataDir(root string) string { return filepath.Join(root, "applications", ".pbgo") }

// tasks is the app: one document with the to-do screens.
type tasks struct {
	doc    *widget.Document
	db     *storage.Store
	screen *presentation.Screen
}

// newTasks is the platform.Factory.
func newTasks(env platform.Env) platform.App { return newTasksAt(env, dataDir(platform.DefaultRoot)) }

// newTasksAt builds the app over a data folder (the tests use a temporary one).
func newTasksAt(env platform.Env, dir string) *tasks {
	a := &tasks{doc: widget.NewDocument(env.Width, env.Height)}
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
	list := application.Checklist{AccountID: localAccount, Store: store}
	if _, err := list.EnsureList(tr.T("tasks.default.list")); err != nil {
		a.fail(err)
		return a
	}
	a.screen = presentation.NewScreen(presentation.Config{
		Width: env.Width, Height: env.Height, Doc: a.doc, Chrome: platform.Chrome(env), Tr: tr, Checklist: list,
	})
	return a
}

// Document implements platform.App.
func (a *tasks) Document() *widget.Document { return a.doc }

// close releases the database (the process ending does it on the reader; tests need it).
func (a *tasks) close() {
	if a.db != nil {
		_ = a.db.Close()
	}
}

// Key implements platform.KeyHandler: the page keys turn the list.
func (a *tasks) Key(k platform.Key) bool {
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
func (a *tasks) fail(err error) {
	platform.Errorf("tasks: %v", err)
	text := widget.NewLabel(fmt.Sprintf("%v", err))
	text.Style.Padding = geom.All(40)
	root := widget.NewColumn(0, text)
	root.Style.Background = widget.Paper
	a.doc.SetRoot(root)
}
