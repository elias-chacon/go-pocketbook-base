package widget

import (
	"errors"
	"strings"
	"testing"
)

type oauthFixture struct {
	login   *OAuthLogin
	d       *Document
	done    []string
	pending bool
	err     error
	typed   string
}

func newOAuthFixture() *oauthFixture {
	f := &oauthFixture{}
	f.login = NewOAuthLogin(
		OAuthProvider{ID: "microsoft", Label: "Microsoft",
			Start: func() (OAuthChallenge, error) {
				return OAuthChallenge{URL: "https://ms.example/device", Code: "AB12CD"}, nil
			},
			Complete: func(OAuthChallenge, string) (bool, error) { return f.pending, f.err }},
		OAuthProvider{ID: "google", Label: "Google",
			Start:    func() (OAuthChallenge, error) { return OAuthChallenge{URL: "https://g.example/auth"}, nil },
			Complete: func(_ OAuthChallenge, typed string) (bool, error) { f.typed = typed; return false, f.err }},
		OAuthProvider{ID: "other", Label: "Other"},
	)
	f.login.OnSuccess = func(id string) { f.done = append(f.done, id) }
	f.d = testDoc(f.login.Root)
	return f
}

func (f *oauthFixture) tapID(t *testing.T, id string) {
	t.Helper()
	f.d.Layout()
	w := f.d.GetElementByID(id)
	if w == nil {
		t.Fatalf("no element %q", id)
	}
	tap(f.d, w)
	f.d.Layout()
}

func (f *oauthFixture) note() string { return f.d.GetElementByID("oauth-note").(*Label).Text }

func TestOAuthLoginOffersAButtonPerProviderAndDisablesTheOneNotConfigured(t *testing.T) {
	f := newOAuthFixture()
	for _, id := range []string{"oauth-microsoft", "oauth-google", "oauth-other"} {
		if f.d.GetElementByID(id) == nil {
			t.Errorf("%s is missing", id)
		}
	}
	if !f.d.GetElementByID("oauth-other").Elem().Disabled {
		t.Error("a provider without Start is not configured")
	}
}

func TestOAuthLoginDeviceCodeShowsTheCodeAndWaitsForApproval(t *testing.T) {
	f := newOAuthFixture()
	f.tapID(t, "oauth-microsoft")
	code := f.d.GetElementByID("oauth-code").(*Label)
	if code.Text != "AB12CD" || f.d.GetElementByID("oauth-qr") != nil {
		t.Fatalf("a device code is shown as text: %q", code.Text)
	}
	f.pending = true
	f.tapID(t, "oauth-complete")
	if f.note() != DefaultOAuthTexts().Pending || len(f.done) != 0 {
		t.Errorf("not approved yet: %q %v", f.note(), f.done)
	}
	f.pending = false
	f.tapID(t, "oauth-complete")
	if len(f.done) != 1 || f.done[0] != "microsoft" || f.note() != DefaultOAuthTexts().Done {
		t.Errorf("approved: %v %q", f.done, f.note())
	}
	if f.d.GetElementByID("oauth-microsoft") == nil {
		t.Error("after signing in the providers are offered again")
	}
}

func TestOAuthLoginAuthorizationCodeShowsAQRCodeAndSendsWhatWasPasted(t *testing.T) {
	f := newOAuthFixture()
	f.tapID(t, "oauth-google")
	if f.d.GetElementByID("oauth-qr") == nil || f.d.GetElementByID("oauth-code") != nil {
		t.Fatal("an authorization code is shown as a QR code")
	}
	in := f.d.GetElementByID("oauth-input").(*Input)
	in.Value = "4/0Abc"
	f.tapID(t, "oauth-complete")
	if f.typed != "4/0Abc" || len(f.done) != 1 || f.done[0] != "google" {
		t.Errorf("typed %q done %v", f.typed, f.done)
	}
}

func TestOAuthLoginFailuresAreToldAndCancelGoesBack(t *testing.T) {
	f := newOAuthFixture()
	f.tapID(t, "oauth-microsoft")
	f.err = errors.New("denied")
	f.tapID(t, "oauth-complete")
	if !strings.Contains(f.note(), "denied") || len(f.done) != 0 {
		t.Errorf("the reason is told: %q", f.note())
	}
	f.tapID(t, "oauth-cancel")
	if f.d.GetElementByID("oauth-google") == nil || f.d.GetElementByID("oauth-code") != nil {
		t.Error("cancel goes back to the providers")
	}
	bad := NewOAuthLogin(OAuthProvider{ID: "x", Label: "X", Start: func() (OAuthChallenge, error) { return OAuthChallenge{}, errors.New("offline") }})
	d := testDoc(bad.Root)
	tap(d, d.GetElementByID("oauth-x"))
	if n := d.GetElementByID("oauth-note").(*Label).Text; !strings.Contains(n, "offline") {
		t.Errorf("a Start that fails stays on the choice and says why: %q", n)
	}
}

func TestOAuthLoginEveryButtonIsReachableAndBigEnough(t *testing.T) {
	f := newOAuthFixture()
	f.tapID(t, "oauth-google")
	for _, id := range []string{"oauth-complete", "oauth-cancel", "oauth-input"} {
		w := f.d.GetElementByID(id)
		if w == nil || w.Elem().Bounds.H < ControlHeight-1 || w.Elem().Bounds.Bottom() > f.d.H {
			t.Errorf("%s: %+v", id, w)
		}
	}
}
