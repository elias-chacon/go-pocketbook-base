package presentation

import (
	"errors"
	"strings"
	"testing"

	"go-pocketbook-base/internal/accounts/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

type fakeManager struct {
	updated   domain.Account
	secret    domain.Secret
	removed   bool
	updateErr error
}

func (f *fakeManager) Update(a domain.Account) error {
	f.updated = a
	return f.updateErr
}
func (f *fakeManager) Refresh(_ string, s domain.Secret) error { f.secret = s; return nil }
func (f *fakeManager) Remove(string) error                     { f.removed = true; return nil }

func gmailAccount() domain.Account {
	p, _ := domain.PresetByID("gmail")
	a := p.Account("Ana", "ana@gmail.com")
	a.ID = "ana"
	return a
}

type editorFixture struct {
	e       *Editor
	mgr     *fakeManager
	saved   *domain.Account
	removed *bool
	back    *bool
	tested  *string
}

func newEditorFixture(t *testing.T, acc domain.Account, testErr error) editorFixture {
	t.Helper()
	f := editorFixture{mgr: &fakeManager{}, saved: &domain.Account{}, removed: new(bool), back: new(bool), tested: new(string)}
	f.e = NewEditor(EditorConfig{
		Width: 1404, Height: 1872, Tr: NewTranslator("en"), Account: acc, Manager: f.mgr,
		Test:      func(a domain.Account, pw string) error { *f.tested = a.Name + "|" + pw; return testErr },
		OnSaved:   func(a domain.Account) { *f.saved = a },
		OnRemoved: func() { *f.removed = true },
		OnBack:    func() { *f.back = true },
	})
	f.e.Doc.Layout()
	return f
}

func (f editorFixture) tap(t *testing.T, id string) {
	t.Helper()
	f.e.Doc.Layout()
	var w widget.Widget
	f.e.Doc.Walk(func(x widget.Widget) bool {
		if x.Elem().ID == id {
			w = x
		}
		return true
	})
	if w == nil {
		t.Fatalf("no element %q", id)
	}
	c := w.Elem().Bounds.Center()
	if f.e.Doc.TargetAt(c) == nil {
		t.Fatalf("%s is not reachable", id)
	}
	f.e.Doc.HandlePointer(widget.PointerDown, c.X*f.e.Doc.Scale, c.Y*f.e.Doc.Scale)
	f.e.Doc.HandlePointer(widget.PointerUp, c.X*f.e.Doc.Scale, c.Y*f.e.Doc.Scale)
}

func TestEditorSavesServersSecuritySignatureAndPassword(t *testing.T) {
	f := newEditorFixture(t, gmailAccount(), nil)
	f.e.name.Value = "Ana Souza"
	f.e.login.Value = "ana.login"
	f.e.password.Value = "new-pass"
	f.e.incoming.host.Value, f.e.incoming.port.Value = "imap.example.org", "143"
	f.e.incoming.setSecurity(f.e, domain.StartTLS)
	f.e.signature.Value = "Ana\n"
	f.tap(t, "acct-save")
	got := f.mgr.updated
	imap, _ := got.Endpoint(domain.Mail, domain.IMAP)
	if got.Name != "Ana Souza" || got.Username != "ana.login" || got.Signature != "Ana" {
		t.Fatalf("identity not kept: %+v", got)
	}
	if imap.Host != "imap.example.org" || imap.Port != 143 || imap.Security != domain.StartTLS {
		t.Fatalf("imap = %+v", imap)
	}
	if smtp, ok := got.Endpoint(domain.Mail, domain.SMTP); !ok || smtp.Host != "smtp.gmail.com" {
		t.Fatalf("smtp lost: %+v", smtp)
	}
	if f.mgr.secret.Password != "new-pass" || f.e.password.Value != "" || f.saved.ID != "ana" {
		t.Fatalf("password/saved wrong: %+v %q %+v", f.mgr.secret, f.e.password.Value, f.saved)
	}
}

func TestEditorKeepsThePasswordWhenBlankAndRejectsABadPort(t *testing.T) {
	f := newEditorFixture(t, gmailAccount(), nil)
	f.e.incoming.port.Value = "abc"
	f.tap(t, "acct-save")
	if !strings.Contains(f.e.note.Text, "imap.gmail.com") || f.mgr.updated.ID != "" {
		t.Fatalf("a bad port was accepted: %q", f.e.note.Text)
	}
	f.e.incoming.port.Value = "993"
	f.tap(t, "acct-save")
	if f.mgr.secret.Password != "" || f.mgr.updated.ID != "ana" {
		t.Fatalf("blank password must keep the stored one: %+v", f.mgr.secret)
	}
}

func TestEditorShowsAStoreFailure(t *testing.T) {
	f := newEditorFixture(t, gmailAccount(), nil)
	f.mgr.updateErr = errors.New("disk full")
	f.tap(t, "acct-save")
	if !strings.Contains(f.e.note.Text, "disk full") {
		t.Fatalf("note = %q", f.e.note.Text)
	}
}

func TestEditorTestsTheConnection(t *testing.T) {
	f := newEditorFixture(t, gmailAccount(), nil)
	f.e.password.Value = "pw"
	f.tap(t, "acct-test")
	if *f.tested != "Ana|pw" || !strings.Contains(f.e.note.Text, "Connected") {
		t.Fatalf("tested %q note %q", *f.tested, f.e.note.Text)
	}
	f = newEditorFixture(t, gmailAccount(), errors.New("app password required"))
	f.tap(t, "acct-test")
	if !strings.Contains(f.e.note.Text, "app password required") {
		t.Fatalf("the reason is hidden: %q", f.e.note.Text)
	}
}

func TestEditorRemovesAfterConfirmationAndGoesBack(t *testing.T) {
	f := newEditorFixture(t, gmailAccount(), nil)
	f.tap(t, "acct-remove")
	if *f.removed || f.mgr.removed {
		t.Fatal("removed on the first tap")
	}
	f.tap(t, "acct-remove")
	if !*f.removed || !f.mgr.removed {
		t.Fatal("not removed on the second tap")
	}
	f.tap(t, "acct-back")
	if !*f.back {
		t.Fatal("back did nothing")
	}
}

func TestEditorOfAnOAuthAccountOnlyEditsNameAndSignature(t *testing.T) {
	acc := domain.Account{ID: "ana", Name: "Ana", Email: "ana@outlook.com", Auth: domain.OAuth2,
		Services: map[domain.Service][]domain.Endpoint{domain.Mail: {{Kind: domain.Graph}}}}
	f := newEditorFixture(t, acc, nil)
	if f.e.login != nil || f.e.password != nil {
		t.Fatal("an OAuth account has no login or password to edit")
	}
	f.e.signature.Value = "Ana"
	f.tap(t, "acct-save")
	if f.mgr.updated.Signature != "Ana" || f.mgr.secret.Password != "" {
		t.Fatalf("updated %+v", f.mgr.updated)
	}
}
