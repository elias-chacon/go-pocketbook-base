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
	got      application.Registration
	gotToken domain.Secret
	err      error
}

func (f *fakeRegistrar) Register(r application.Registration) (domain.Account, error) {
	f.got = r
	if f.err != nil {
		return domain.Account{}, f.err
	}
	return domain.Account{ID: "ana", Email: r.Email}, nil
}

func (f *fakeRegistrar) RegisterOAuth(r application.Registration, s domain.Secret) (domain.Account, error) {
	f.got, f.gotToken = r, s
	if f.err != nil {
		return domain.Account{}, f.err
	}
	return domain.Account{ID: "ana", Email: r.Email, Auth: domain.OAuth2}, nil
}

// fakeAuth answers the polls the test sets.
type fakeAuth struct {
	startErr  error
	waitErr   error
	secret    domain.Secret
	startedN  int
	challenge domain.Challenge
}

func (f *fakeAuth) Start() (domain.Challenge, error) {
	f.startedN++
	return f.challenge, f.startErr
}

func (f *fakeAuth) Wait(_ domain.Challenge, cancelled func() bool) (domain.Secret, error) {
	if errors.Is(f.waitErr, domain.ErrCancelled) {
		cancelled()
		if cancelled() {
			return domain.Secret{}, domain.ErrCancelled
		}
	}
	return f.secret, f.waitErr
}

func (f *fakeAuth) Refresh(domain.Secret) (domain.Secret, error) { return f.secret, nil }

func newScreen(f *fakeRegistrar, auth domain.Authorizer, cancel func()) (*SignIn, *domain.Account) {
	var done domain.Account
	s := NewSignIn(SignInConfig{
		Width: 1404, Height: 1872, Tr: NewTranslator("pt-BR"), Accounts: f, Authorizer: auth,
		OAuthSetup: "applications/.pbgo/oauth.json",
		OnDone:     func(a domain.Account) { done = a }, OnCancel: cancel,
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

func tap(t *testing.T, s *SignIn, id string) {
	t.Helper()
	s.Doc.Layout()
	c := find(t, s, id).Elem().Bounds.Center()
	s.Doc.HandlePointer(widget.PointerDown, c.X*s.Doc.Scale, c.Y*s.Doc.Scale)
	s.Doc.HandlePointer(widget.PointerUp, c.X*s.Doc.Scale, c.Y*s.Doc.Scale)
	s.Doc.Layout()
}

func labels(s *SignIn) string {
	var b strings.Builder
	s.Doc.Walk(func(w widget.Widget) bool {
		if l, ok := w.(*widget.Label); ok {
			b.WriteString(l.Text + "\n")
		}
		return true
	})
	return b.String()
}

func TestEveryTextIsTranslatedInBothLanguages(t *testing.T) {
	tr := NewTranslator("pt-BR")
	for _, lang := range []string{"pt-BR", "en"} {
		if missing := tr.Missing(lang); len(missing) != 0 {
			t.Errorf("%s lacks: %v", lang, missing)
		}
	}
}

func TestTheFirstScreenOffersTheProviders(t *testing.T) {
	cancelled := false
	s, _ := newScreen(&fakeRegistrar{}, nil, func() { cancelled = true })
	for _, id := range []string{"signin-outlook", "signin-gmail", "signin-icloud", "signin-other", "signin-cancel"} {
		find(t, s, id)
	}
	tap(t, s, "signin-cancel")
	if !cancelled {
		t.Error("cancel did nothing")
	}
	tap(t, s, "signin-gmail")
	if s.Provider() != providerGmail {
		t.Fatalf("provider = %q", s.Provider())
	}
	tap(t, s, "signin-back")
	if s.Provider() != "" {
		t.Error("back did not return to the choice")
	}
}

func TestAPasswordProviderAsksForAnAppPasswordAndOnlyOtherAsksForServers(t *testing.T) {
	f := &fakeRegistrar{}
	s, done := newScreen(f, nil, nil)
	tap(t, s, "signin-gmail")
	if !strings.Contains(labels(s), "senha de app") {
		t.Errorf("no app password hint:\n%s", labels(s))
	}
	if !s.imap.Hidden || !s.smtp.Hidden {
		t.Error("Gmail needs no servers")
	}
	s.email.Value, s.password.Value, s.name.Value = "ana@gmail.com", "pw", "Ana"
	tap(t, s, "signin-add")
	if f.got.Email != "ana@gmail.com" || f.got.Password != "pw" || f.got.Name != "Ana" || done.ID != "ana" {
		t.Errorf("registration = %+v done=%+v", f.got, done)
	}
	if s.password.Value != "" {
		t.Error("the password stayed in the field")
	}

	s, _ = newScreen(f, nil, nil)
	tap(t, s, "signin-other")
	if s.imap.Hidden || s.smtp.Hidden || !strings.Contains(labels(s), "não reconhecido") {
		t.Error("another server must ask for the servers")
	}
	s.email.Value = "ana@gmail.com"
	s.update()
	if s.imap.Hidden {
		t.Error("the choice of another server keeps the fields")
	}

	s, _ = newScreen(f, nil, nil)
	tap(t, s, "signin-icloud")
	s.email.Value = "eu@empresa.com.br"
	s.update()
	if s.imap.Hidden {
		t.Error("an unknown domain needs the servers")
	}
	s.email.Value = "eu@outlook.com"
	s.update()
	if !strings.Contains(s.hint.Text, "Outlook") {
		t.Errorf("hint = %q", s.hint.Text)
	}
	s.Submit()
	var none SignIn
	none.Submit()
}

func TestEachFailureIsExplainedWithoutLeakingThePassword(t *testing.T) {
	cases := map[error]string{
		application.ErrInvalidEmail: "Confira o endereço",
		application.ErrNeedPassword: "Digite a senha",
		application.ErrNeedServers:  "servidores IMAP e SMTP",
		application.ErrNeedOAuth:    "Outlook",
		application.ErrNotOAuth:     "não entra com a Microsoft",
		domain.ErrExists:            "já foi adicionada",
		errors.New("disco cheio"):   "disco cheio",
	}
	for err, want := range cases {
		f := &fakeRegistrar{err: err}
		s, done := newScreen(f, nil, nil)
		tap(t, s, "signin-gmail")
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

func TestOutlookWithoutAClientIDSaysHowToEnableIt(t *testing.T) {
	s, _ := newScreen(&fakeRegistrar{}, nil, nil)
	tap(t, s, "signin-outlook")
	if !strings.Contains(labels(s), "oauth.json") || !strings.Contains(labels(s), "client id") {
		t.Errorf("no setup help:\n%s", labels(s))
	}
}

func TestOutlookSignsInByCodeOnAnotherDevice(t *testing.T) {
	f := &fakeRegistrar{}
	auth := &fakeAuth{challenge: domain.Challenge{UserCode: "ABCD1234", VerificationURI: "https://microsoft.com/devicelogin"}, waitErr: domain.ErrCancelled}
	s, done := newScreen(f, auth, nil)
	tap(t, s, "signin-outlook")
	s.email.Value, s.name.Value = "ana@outlook.com", "Ana"
	tap(t, s, "oauth-microsoft")
	if auth.startedN != 1 || !strings.Contains(labels(s), "ABCD1234") || !strings.Contains(labels(s), "devicelogin") {
		t.Fatalf("the code was not shown:\n%s", labels(s))
	}
	tap(t, s, "oauth-complete")
	if !strings.Contains(labels(s), "Ainda não aprovado") || done.ID != "" {
		t.Errorf("pending:\n%s", labels(s))
	}
	auth.waitErr, auth.secret = nil, domain.Secret{AccessToken: "tok", RefreshToken: "r"}
	tap(t, s, "oauth-complete")
	if done.ID != "ana" || f.gotToken.RefreshToken != "r" || f.got.Email != "ana@outlook.com" || f.got.Name != "Ana" {
		t.Errorf("done %+v got %+v token %v", done, f.got, f.gotToken)
	}
}

func TestOutlookFailuresAreExplainedAndTheCodeCanBeCancelled(t *testing.T) {
	auth := &fakeAuth{challenge: domain.Challenge{UserCode: "X"}, waitErr: domain.ErrDenied}
	s, done := newScreen(&fakeRegistrar{}, auth, nil)
	tap(t, s, "signin-outlook")
	tap(t, s, "oauth-microsoft") // no address yet: it is asked before the network is used
	if !strings.Contains(labels(s), "Confira o endereço") || auth.startedN != 0 {
		t.Errorf("an invalid address must be refused first:\n%s", labels(s))
	}
	s.email.Value = "ana@outlook.com"
	tap(t, s, "oauth-microsoft")
	tap(t, s, "oauth-complete")
	if !strings.Contains(labels(s), "negado") || done.ID != "" {
		t.Errorf("denied:\n%s", labels(s))
	}
	auth.waitErr = domain.ErrExpired
	tap(t, s, "oauth-complete")
	if !strings.Contains(labels(s), "expirou") {
		t.Errorf("expired:\n%s", labels(s))
	}
	auth.waitErr = errors.New("sem rede")
	tap(t, s, "oauth-complete")
	if !strings.Contains(labels(s), "sem rede") {
		t.Errorf("network:\n%s", labels(s))
	}
	tap(t, s, "oauth-cancel") // back to the provider
	auth.startErr = errors.New("recusado")
	tap(t, s, "oauth-microsoft")
	if !strings.Contains(labels(s), "recusado") {
		t.Errorf("start failure:\n%s", labels(s))
	}

	auth.startErr, auth.waitErr, auth.secret = nil, nil, domain.Secret{AccessToken: "t"}
	s2, _ := newScreen(&fakeRegistrar{err: application.ErrNotOAuth}, auth, nil)
	tap(t, s2, "signin-outlook")
	s2.email.Value = "ana@gmail.com"
	tap(t, s2, "oauth-microsoft")
	tap(t, s2, "oauth-complete")
	if !strings.Contains(labels(s2), "não entra com a Microsoft") {
		t.Errorf("not oauth:\n%s", labels(s2))
	}
}

// checkScreen verifies that nothing overflows the screen and that every button gets its
// own touch, with a target of at least 56 px.
func checkScreen(t *testing.T, s *SignIn, where string) {
	t.Helper()
	s.Doc.Layout()
	h := s.Doc.Root.Elem().Bounds.H
	s.Doc.Walk(func(w widget.Widget) bool {
		checkWidget(t, s, where, w, h)
		return true
	})
}

func checkWidget(t *testing.T, s *SignIn, where string, w widget.Widget, screenH float64) {
	t.Helper()
	e := w.Elem()
	if !e.Visible() || e.ID == "" {
		return
	}
	if e.Bounds.Y+e.Bounds.H > screenH+0.5 {
		t.Errorf("%s: %s overflows (%v)", where, e.ID, e.Bounds)
	}
	if _, ok := w.(widget.PointerHandler); !ok || e.Disabled || e.Bounds.W <= 0 {
		return
	}
	if got := s.Doc.TargetAt(e.Bounds.Center()); got != w {
		t.Errorf("%s: %s does not get its own touch", where, e.ID)
	}
	if e.Bounds.H < 56 {
		t.Errorf("%s: %s is only %.0f px high", where, e.ID, e.Bounds.H)
	}
}

func TestEveryScreenFitsAndEveryButtonGetsItsTouch(t *testing.T) {
	auth := &fakeAuth{challenge: domain.Challenge{UserCode: "ABCD1234", VerificationURI: "https://microsoft.com/devicelogin"}}
	s, _ := newScreen(&fakeRegistrar{}, auth, func() {})
	checkScreen(t, s, "chooser")
	tap(t, s, "signin-gmail")
	checkScreen(t, s, "password")
	tap(t, s, "signin-back")
	tap(t, s, "signin-other")
	checkScreen(t, s, "other")
	tap(t, s, "signin-back")
	tap(t, s, "signin-outlook")
	checkScreen(t, s, "outlook")
	s.email.Value = "ana@outlook.com"
	tap(t, s, "oauth-microsoft")
	checkScreen(t, s, "code")
}
