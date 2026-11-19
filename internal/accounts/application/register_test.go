package application

import (
	"errors"
	"testing"
	"time"

	"go-pocketbook-base/internal/accounts/domain"
)

func registrar() (Accounts, *store, *vault) {
	st, v := newStore(), newVault()
	return Accounts{Store: st, Vault: v}, st, v
}

func TestRegisterBuildsAKnownProviderFromTheAddress(t *testing.T) {
	u, st, v := registrar()
	acc, err := u.Register(Registration{Email: " Ana@Gmail.com ", Password: "app-pw", Name: "Ana"})
	if err != nil {
		t.Fatal(err)
	}
	if acc.Provider != "gmail" || acc.Auth != domain.Password || acc.Email != "Ana@Gmail.com" {
		t.Errorf("account = %+v", acc)
	}
	if _, ok := acc.Endpoint(domain.Mail, domain.IMAP); !ok {
		t.Error("no IMAP endpoint")
	}
	if _, err := st.Get(acc.ID); err != nil {
		t.Errorf("not stored: %v", err)
	}
	if got, _ := v.Get(acc.ID); got.Password != "app-pw" {
		t.Error("the password was not kept in the vault")
	}
}

func TestRegisterAUnknownProviderNeedsTheServers(t *testing.T) {
	u, _, _ := registrar()
	r := Registration{Email: "eu@empresa.com.br", Password: "x"}
	if _, err := u.Register(r); !errors.Is(err, ErrNeedServers) {
		t.Errorf("err = %v", err)
	}
	r.IMAPHost, r.SMTPHost = "imap.empresa.com.br", "smtp.empresa.com.br"
	acc, err := u.Register(r)
	if err != nil {
		t.Fatal(err)
	}
	imap, _ := acc.Endpoint(domain.Mail, domain.IMAP)
	smtp, _ := acc.Endpoint(domain.Mail, domain.SMTP)
	if imap.Host != "imap.empresa.com.br" || imap.Port != 993 || smtp.Security != domain.StartTLS || acc.Name != "eu@empresa.com.br" {
		t.Errorf("account = %+v", acc)
	}
}

func TestRegisterRefusesWhatCannotWork(t *testing.T) {
	u, _, _ := registrar()
	cases := []struct {
		r    Registration
		want error
	}{
		{Registration{Email: "sem-arroba", Password: "x"}, ErrInvalidEmail},
		{Registration{Email: "a@semponto", Password: "x"}, ErrInvalidEmail},
		{Registration{Email: "a@gmail.com"}, ErrNeedPassword},
		{Registration{Email: "a@outlook.com", Password: "x"}, ErrNeedOAuth},
	}
	for _, c := range cases {
		if _, err := u.Register(c.r); !errors.Is(err, c.want) {
			t.Errorf("%+v: err = %v, want %v", c.r, err, c.want)
		}
	}
}

func TestRegisterTwiceIsAnError(t *testing.T) {
	u, _, _ := registrar()
	r := Registration{Email: "a@gmail.com", Password: "x"}
	if _, err := u.Register(r); err != nil {
		t.Fatal(err)
	}
	if _, err := u.Register(r); !errors.Is(err, domain.ErrExists) {
		t.Errorf("err = %v", err)
	}
}

func TestProviderOf(t *testing.T) {
	if p := ProviderOf("a@gmail.com"); !p.Known || p.OAuth() {
		t.Errorf("gmail = %+v", p)
	}
	if p := ProviderOf("a@hotmail.com"); !p.Known || !p.OAuth() {
		t.Errorf("hotmail = %+v", p)
	}
	if p := ProviderOf("a@x.org"); p.Known || p.OAuth() {
		t.Errorf("custom = %+v", p)
	}
}

// fakeAuth is an Authorizer whose answers the test sets.
type fakeAuth struct {
	waitSecret domain.Secret
	waitErr    error
	refreshed  domain.Secret
	refreshErr error
	refreshes  int
	polls      int
}

func (f *fakeAuth) Start() (domain.Challenge, error) { return domain.Challenge{UserCode: "ABCD"}, nil }
func (f *fakeAuth) Wait(_ domain.Challenge, cancelled func() bool) (domain.Secret, error) {
	f.polls++
	if errors.Is(f.waitErr, domain.ErrCancelled) {
		cancelled() // the first ask is false: one poll
		if cancelled() {
			return domain.Secret{}, domain.ErrCancelled
		}
	}
	if f.waitErr != nil {
		return domain.Secret{}, f.waitErr
	}
	return f.waitSecret, nil
}
func (f *fakeAuth) Refresh(domain.Secret) (domain.Secret, error) {
	f.refreshes++
	return f.refreshed, f.refreshErr
}

func TestCheckOAuthReportsPendingApprovedAndFailed(t *testing.T) {
	approved := &fakeAuth{waitSecret: domain.Secret{AccessToken: "tok"}}
	if s, pending, err := CheckOAuth(approved, domain.Challenge{}); err != nil || pending || s.AccessToken != "tok" {
		t.Errorf("approved: %v %v %v", s, pending, err)
	}
	waiting := &fakeAuth{waitErr: domain.ErrCancelled}
	if _, pending, err := CheckOAuth(waiting, domain.Challenge{}); err != nil || !pending {
		t.Errorf("pending: %v %v", pending, err)
	}
	denied := &fakeAuth{waitErr: domain.ErrDenied}
	if _, pending, err := CheckOAuth(denied, domain.Challenge{}); !errors.Is(err, domain.ErrDenied) || pending {
		t.Errorf("denied: %v %v", pending, err)
	}
}

func TestRegisterOAuthAddsAnOutlookAccountWithItsTokens(t *testing.T) {
	u, st, v := registrar()
	acc, err := u.RegisterOAuth(Registration{Email: "ana@outlook.com", Name: "Ana"}, domain.Secret{AccessToken: "a", RefreshToken: "r"})
	if err != nil {
		t.Fatal(err)
	}
	if acc.Auth != domain.OAuth2 || acc.Provider != "outlook" {
		t.Errorf("account = %+v", acc)
	}
	if _, ok := acc.Endpoint(domain.Mail, domain.Graph); !ok {
		t.Error("no Graph endpoint")
	}
	if got, _ := v.Get(acc.ID); got.RefreshToken != "r" {
		t.Error("the tokens were not kept in the vault")
	}
	if _, err := st.Get(acc.ID); err != nil {
		t.Error(err)
	}
	cases := []struct {
		r    Registration
		s    domain.Secret
		want error
	}{
		{Registration{Email: "x"}, domain.Secret{AccessToken: "a"}, ErrInvalidEmail},
		{Registration{Email: "a@gmail.com"}, domain.Secret{AccessToken: "a"}, ErrNotOAuth},
		{Registration{Email: "b@outlook.com"}, domain.Secret{}, ErrNoTokens},
	}
	for _, c := range cases {
		if _, err := u.RegisterOAuth(c.r, c.s); !errors.Is(err, c.want) {
			t.Errorf("%+v: err = %v, want %v", c.r, err, c.want)
		}
	}
}

func TestAccessTokenIsReusedRenewedAndKept(t *testing.T) {
	u, _, v := registrar()
	now := time.Unix(1_000_000, 0)
	acc, _ := u.RegisterOAuth(Registration{Email: "ana@outlook.com"}, domain.Secret{AccessToken: "old", RefreshToken: "r", ExpiresAt: now.Unix() + 3600})
	auth := &fakeAuth{refreshed: domain.Secret{AccessToken: "new", RefreshToken: "r2", ExpiresAt: now.Unix() + 7200}}
	if tok, err := u.AccessToken(acc.ID, auth, now); err != nil || tok != "old" || auth.refreshes != 0 {
		t.Fatalf("valid token: %q %v refreshes=%d", tok, err, auth.refreshes)
	}
	later := now.Add(2 * time.Hour)
	if tok, err := u.AccessToken(acc.ID, auth, later); err != nil || tok != "new" || auth.refreshes != 1 {
		t.Fatalf("renewed token: %q %v refreshes=%d", tok, err, auth.refreshes)
	}
	if got, _ := v.Get(acc.ID); got.AccessToken != "new" || got.RefreshToken != "r2" {
		t.Errorf("the vault kept %v", got)
	}
	auth.refreshErr = errors.New("revoked")
	if _, err := u.AccessToken(acc.ID, auth, later.Add(5*time.Hour)); err == nil {
		t.Error("a refresh failure was hidden")
	}
	if _, err := u.AccessToken(acc.ID, nil, later.Add(5*time.Hour)); !errors.Is(err, domain.ErrNoRefresh) {
		t.Errorf("err = %v", err)
	}
	if _, err := u.AccessToken("ghost", auth, now); err == nil {
		t.Error("an unknown account")
	}
}
