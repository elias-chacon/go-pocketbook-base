package presentation

import (
	"errors"
	"strings"
	"testing"
	"time"

	"go-pocketbook-base/internal/tasks/application"
	"go-pocketbook-base/internal/tasks/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

type fakeSyncer struct {
	account    string
	res        application.SyncResult
	err        error
	ran        int
	connected  int
	disconnect error
}

func (f *fakeSyncer) Account() string { return f.account }
func (f *fakeSyncer) Run() (application.SyncResult, error) {
	f.ran++
	return f.res, f.err
}
func (f *fakeSyncer) Connect() { f.connected++ }
func (f *fakeSyncer) Disconnect() error {
	f.account = ""
	return f.disconnect
}

func syncScreen(t *testing.T, f *fakeSyncer) *Screen {
	t.Helper()
	st := &memStore{tasks: map[string]domain.Task{}}
	cfg := Config{Width: 1404, Height: 1872, Tr: NewTranslator("en"), Checklist: application.Checklist{AccountID: "local", Store: st, Now: func() time.Time { return time.Date(2026, 5, 20, 10, 0, 0, 0, time.UTC) }}, Syncer: f}
	s := NewScreen(cfg)
	s.Doc.Layout()
	return s
}

func TestSyncMenuIsInTheTitleRowOnlyWithASyncer(t *testing.T) {
	s := syncScreen(t, &fakeSyncer{})
	byID(t, s, "tasks-sync")
	plain, _ := newScreen()
	found := false
	plain.Doc.Walk(func(w widget.Widget) bool {
		if w.Elem().ID == "tasks-sync" {
			found = true
		}
		return true
	})
	if found {
		t.Fatal("the Sync menu appears without a Syncer")
	}
}

func TestSyncNowConnectsWhenThereIsNoAccountAndSyncsWhenThereIs(t *testing.T) {
	f := &fakeSyncer{}
	s := syncScreen(t, f)
	s.SyncNow()
	if f.connected != 1 || f.ran != 0 {
		t.Fatalf("connected %d ran %d", f.connected, f.ran)
	}
	f.account = "ana@outlook.com"
	f.res = application.SyncResult{Sent: 1, Received: 2, Removed: 3}
	s.SyncNow()
	if f.ran != 1 || !strings.Contains(s.notice, "1 sent, 2 received, 3 removed") {
		t.Fatalf("ran %d notice %q", f.ran, s.notice)
	}
	f.res = application.SyncResult{Errors: []error{errors.New("list timeout")}}
	s.SyncNow()
	if !strings.Contains(s.notice, "list timeout") {
		t.Fatalf("notice %q", s.notice)
	}
	f.err = errors.New("401")
	s.SyncNow()
	if !strings.Contains(s.notice, "Could not sync: 401") {
		t.Fatalf("notice %q", s.notice)
	}
}

func TestDisconnectForgetsTheAccountAndReportsFailure(t *testing.T) {
	f := &fakeSyncer{account: "ana@outlook.com"}
	s := syncScreen(t, f)
	s.Disconnect()
	if f.account != "" {
		t.Fatal("still connected")
	}
	f.account, f.disconnect = "x", errors.New("disk")
	s.Disconnect()
	if !strings.Contains(s.notice, "disk") {
		t.Fatalf("notice %q", s.notice)
	}
	s.cfg.Syncer = nil
	s.SyncNow() // no syncer: nothing happens
}
