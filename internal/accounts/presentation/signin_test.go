package presentation

import (
	"errors"
	"strings"
	"testing"

	"go-pocketbook-base/internal/accounts/application"
	"go-pocketbook-base/internal/accounts/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

type fakeRegistrar struct {
	got  application.Registration
	err  error
	done int
}

func (f *fakeRegistrar) Register(r application.Registration) (domain.Account, error) {
	f.got = r
	if f.err != nil {
		return domain.Account{}, f.err
	}
	return domain.Account{ID: "ana", Email: r.Email}, nil
}

func newScreen(f *fakeRegistrar, cancel func()) (*SignIn, *domain.Account) {
	var done domain.Account
	s := NewSignIn(SignInConfig{
		Width: 1404, Height: 1872, Tr: NewTranslator("pt-BR"), Accounts: f,
		OnDone: func(a domain.Account) { done = a }, OnCancel: cancel,
	})
	s.Doc.Layout()
	return s, &done
}

func find(t *testing.T, s *SignIn, id string) widget.Widget {
	t.Helper()
	var out widget.Widget
	s.Doc.Walk(func(w widget.Widget) bool {
		if w.Elem().ID == id {
			out = w
		}
		return true
	})
	if out == nil {
		t.Fatalf("no element %q", id)
	}
	return out
}

func TestEveryTextIsTranslatedInBothLanguages(t *testing.T) {
	tr := NewTranslator("pt-BR")
	for _, lang := range []string{"pt-BR", "en"} {
		if missing := tr.Missing(lang); len(missing) != 0 {
			t.Errorf("%s lacks: %v", lang, missing)
		}
	}
}

func TestTheServersAppearOnlyForAnUnknownProvider(t *testing.T) {
	s, _ := newScreen(&fakeRegistrar{}, nil)
	if !s.imap.Hidden || !s.smtp.Hidden {
		t.Error("servers shown before an address")
	}
	s.email.Value = "ana@gmail.com"
	s.update()
	if !s.imap.Hidden || !strings.Contains(s.hint.Text, "senha de app") {
		t.Errorf("gmail: hidden=%v hint=%q", s.imap.Hidden, s.hint.Text)
	}
	s.email.Value = "ana@empresa.com.br"
	s.update()
	if s.imap.Hidden || s.smtp.Hidden {
		t.Error("servers hidden for a custom provider")
	}
	s.email.Value = "ana@outlook.com"
	s.update()
	if !s.add.Disabled || !strings.Contains(s.hint.Text, "ainda não está disponível") {
		t.Errorf("outlook: disabled=%v hint=%q", s.add.Disabled, s.hint.Text)
	}
}

func TestSubmitHandsWhatWasTypedToTheUseCase(t *testing.T) {
	f := &fakeRegistrar{}
	s, done := newScreen(f, nil)
	s.email.Value, s.password.Value, s.name.Value = "ana@gmail.com", "pw", "Ana"
	s.imap.Value, s.smtp.Value = "i", "s"
	s.Submit()
	if f.got.Email != "ana@gmail.com" || f.got.Password != "pw" || f.got.Name != "Ana" || f.got.IMAPHost != "i" {
		t.Errorf("registration = %+v", f.got)
	}
	if done.ID != "ana" {
		t.Errorf("done = %+v", done)
	}
	if s.password.Value != "" {
		t.Error("the password stayed in the field")
	}
}

func TestEachFailureIsExplainedWithoutLeakingThePassword(t *testing.T) {
	cases := map[error]string{
		application.ErrInvalidEmail: "Confira o endereço",
		application.ErrNeedPassword: "Digite a senha",
		application.ErrNeedServers:  "servidores IMAP e SMTP",
		application.ErrNeedOAuth:    "ainda não está disponível",
		domain.ErrExists:            "já foi adicionada",
		errors.New("disco cheio"):   "disco cheio",
	}
	for err, want := range cases {
		f := &fakeRegistrar{err: err}
		s, done := newScreen(f, nil)
		s.password.Value = "SEGREDO"
		s.Submit()
		if !strings.Contains(s.note.Text, want) {
			t.Errorf("%v: note = %q, want %q", err, s.note.Text, want)
		}
		if strings.Contains(s.note.Text, "SEGREDO") || done.ID != "" {
			t.Errorf("%v: leaked or finished (%q, %v)", err, s.note.Text, done)
		}
	}
}

func TestTheButtonsReceiveTheirTouch(t *testing.T) {
	cancelled := false
	f := &fakeRegistrar{}
	s, done := newScreen(f, func() { cancelled = true })
	s.email.Value = "ana@gmail.com"
	s.update()
	s.Doc.Layout()
	for _, id := range []string{"signin-add", "signin-cancel", "signin-email", "signin-password", "signin-name"} {
		w := find(t, s, id)
		if got := s.Doc.TargetAt(w.Elem().Bounds.Center()); got != w {
			t.Errorf("%s does not get its own touch", id)
		}
	}
	add := find(t, s, "signin-add").Elem().Bounds.Center()
	s.Doc.HandlePointer(widget.PointerDown, add.X*s.Doc.Scale, add.Y*s.Doc.Scale)
	s.Doc.HandlePointer(widget.PointerUp, add.X*s.Doc.Scale, add.Y*s.Doc.Scale)
	if f.got.Email != "ana@gmail.com" || done.ID != "ana" {
		t.Errorf("add did not submit: %+v", f.got)
	}
	cancel := find(t, s, "signin-cancel").Elem().Bounds.Center()
	s.Doc.HandlePointer(widget.PointerDown, cancel.X*s.Doc.Scale, cancel.Y*s.Doc.Scale)
	s.Doc.HandlePointer(widget.PointerUp, cancel.X*s.Doc.Scale, cancel.Y*s.Doc.Scale)
	if !cancelled {
		t.Error("cancel did nothing")
	}
}
