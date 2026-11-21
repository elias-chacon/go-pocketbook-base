package presentation

import (
	"errors"
	"strings"
	"testing"

	"go-pocketbook-base/internal/accounts/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

type fakeCodeAuth struct {
	fakeAuth
	gotCode string
	exErr   error
}

func (f *fakeCodeAuth) Exchange(_ domain.Challenge, code string) (domain.Secret, error) {
	f.gotCode = code
	return f.secret, f.exErr
}

type connectFixture struct {
	c    *Connect
	reg  *fakeRegistrar
	done *domain.Account
	ms   *fakeAuth
	goog *fakeCodeAuth
}

func newConnect(t *testing.T, microsoft, google bool) connectFixture {
	t.Helper()
	f := connectFixture{reg: &fakeRegistrar{}, done: &domain.Account{}, ms: &fakeAuth{challenge: domain.Challenge{UserCode: "ABC", VerificationURI: "https://ms/device"}, secret: domain.Secret{AccessToken: "a"}},
		goog: &fakeCodeAuth{fakeAuth: fakeAuth{challenge: domain.Challenge{VerificationURI: "https://accounts.example/auth?x=1", DeviceCode: "ver"}, secret: domain.Secret{AccessToken: "g", RefreshToken: "r"}}}}
	cfg := ConnectConfig{Width: 1404, Height: 1872, Tr: NewTranslator("en"), Accounts: f.reg, OAuthSetup: "/x/oauth.json",
		OnDone: func(a domain.Account) { *f.done = a }, OnCancel: func() {}}
	if microsoft {
		cfg.Microsoft = f.ms
	}
	if google {
		cfg.Google = f.goog
	}
	f.c = NewConnect(cfg)
	f.c.Doc.Layout()
	return f
}

func (f connectFixture) find(t *testing.T, id string) widget.Widget {
	t.Helper()
	var out widget.Widget
	f.c.Doc.Walk(func(w widget.Widget) bool {
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

func (f connectFixture) tap(t *testing.T, id string) {
	t.Helper()
	f.c.Doc.Layout()
	c := f.find(t, id).Elem().Bounds.Center()
	if f.c.Doc.TargetAt(c) == nil {
		t.Fatalf("%s is not reachable", id)
	}
	f.c.Doc.HandlePointer(widget.PointerDown, c.X*f.c.Doc.Scale, c.Y*f.c.Doc.Scale)
	f.c.Doc.HandlePointer(widget.PointerUp, c.X*f.c.Doc.Scale, c.Y*f.c.Doc.Scale)
	f.c.Doc.Layout()
}

func (f connectFixture) text() string {
	var b strings.Builder
	f.c.Doc.Walk(func(w widget.Widget) bool {
		if l, ok := w.(*widget.Label); ok {
			b.WriteString(l.Text + "\n")
		}
		return true
	})
	return b.String()
}

func (f connectFixture) typeAddress(t *testing.T, email string) {
	t.Helper()
	f.find(t, "connect-email").(*widget.Input).Value = email
}

func TestConnectExplainsAProviderThatIsNotConfigured(t *testing.T) {
	f := newConnect(t, false, false)
	if !strings.Contains(f.text(), "/x/oauth.json") {
		t.Fatalf("no setup hint:\n%s", f.text())
	}
	for _, id := range []string{"oauth-microsoft", "oauth-google"} {
		if !f.find(t, id).Elem().Disabled {
			t.Errorf("%s must be disabled while there is no client id", id)
		}
	}
	if newConnect(t, true, true).c.Doc.GetElementByID("connect-setup") != nil {
		t.Error("with both providers configured there is nothing to explain")
	}
}

func TestConnectMicrosoftSignsInByCode(t *testing.T) {
	f := newConnect(t, true, true)
	f.tap(t, "oauth-microsoft") // without an address nothing starts
	if f.ms.startedN != 0 || !strings.Contains(f.text(), "Check the address") {
		t.Fatalf("the address is asked first:\n%s", f.text())
	}
	f.typeAddress(t, "ana@outlook.com")
	f.tap(t, "oauth-microsoft")
	if got := f.find(t, "oauth-code").(*widget.Label).Text; got != "ABC" {
		t.Fatalf("code = %q", got)
	}
	f.ms.waitErr = domain.ErrCancelled // not approved yet
	f.tap(t, "oauth-complete")
	if !strings.Contains(f.text(), "Not approved") {
		t.Fatalf("pending text missing:\n%s", f.text())
	}
	f.ms.waitErr = nil
	f.tap(t, "oauth-complete")
	if f.reg.got.Provider != "outlook" || f.reg.got.Email != "ana@outlook.com" || f.reg.gotToken.AccessToken != "a" || f.done.ID != "ana" {
		t.Fatalf("registration = %+v token %+v done %+v", f.reg.got, f.reg.gotToken, f.done)
	}
}

func TestConnectGoogleShowsAQRCodeAndTakesTheCode(t *testing.T) {
	f := newConnect(t, true, true)
	f.typeAddress(t, "ana@gmail.com")
	f.tap(t, "oauth-google")
	if im, ok := f.find(t, "oauth-qr").(*widget.Image); !ok || im.Source == nil {
		t.Fatal("no QR code")
	}
	f.find(t, "oauth-input").(*widget.Input).Value = "http://127.0.0.1/?code=4/0Abc"
	f.goog.exErr = errors.New("invalid_grant")
	f.tap(t, "oauth-complete")
	if !strings.Contains(f.text(), "invalid_grant") || f.done.ID != "" {
		t.Fatalf("a failed exchange must be shown:\n%s", f.text())
	}
	f.goog.exErr = nil
	f.tap(t, "oauth-complete")
	if f.goog.gotCode != "http://127.0.0.1/?code=4/0Abc" || f.reg.got.Provider != "google" || f.reg.gotToken.RefreshToken != "r" || f.done.ID != "ana" {
		t.Fatalf("code %q registration %+v done %+v", f.goog.gotCode, f.reg.got, f.done)
	}
}

func TestConnectShowsRegistrationAndStartFailures(t *testing.T) {
	f := newConnect(t, true, true)
	f.typeAddress(t, "ana@outlook.com")
	f.ms.startErr = errors.New("no network")
	f.tap(t, "oauth-microsoft")
	if !strings.Contains(f.text(), "no network") {
		t.Fatalf("start failure hidden:\n%s", f.text())
	}
	f.ms.startErr = nil
	f.tap(t, "oauth-microsoft")
	f.reg.err = domain.ErrExists
	f.tap(t, "oauth-complete")
	if !strings.Contains(f.text(), "already") {
		t.Fatalf("exists not explained:\n%s", f.text())
	}
	f.tap(t, "oauth-cancel")
	f.tap(t, "connect-cancel")
}
