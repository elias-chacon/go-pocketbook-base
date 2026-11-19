package application

import (
	"errors"
	"testing"

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
