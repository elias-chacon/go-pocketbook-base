package application

import (
	"errors"
	"net/mail"
	"strings"

	"go-pocketbook-base/internal/accounts/domain"
)

// Errors of Register that the screen explains to the person.
var (
	ErrInvalidEmail = errors.New("accounts: the email address is not valid")
	ErrNeedPassword = errors.New("accounts: a password is needed")
	ErrNeedServers  = errors.New("accounts: this provider needs its IMAP and SMTP servers")
	ErrNeedOAuth    = errors.New("accounts: this provider signs in with OAuth2, which is not available yet")
)

const (
	defaultIMAPPort  = 993
	defaultSMTPPort  = 587
	maxSlugFromEmail = 32
)

// Registration is what the person types to add an account.
type Registration struct {
	Name     string // shown in the lists; the address when empty
	Email    string
	Password string // a password or an app password
	// IMAPHost and SMTPHost are needed only for a provider that is not known.
	IMAPHost, SMTPHost string
}

// Provider is the preset of an address, and whether it needs the servers typed.
type Provider struct {
	Preset domain.Preset
	Known  bool
}

// ProviderOf recognizes the provider of an address.
func ProviderOf(email string) Provider {
	p, ok := domain.PresetFor(strings.TrimSpace(email))
	if !ok {
		p, _ = domain.PresetByID(domain.Custom)
	}
	return Provider{Preset: p, Known: ok}
}

// OAuth reports whether the provider signs in with OAuth2 instead of a password.
func (p Provider) OAuth() bool {
	return p.Preset.Account("x", "x@x").Auth == domain.OAuth2
}

// Register builds the account of an address from its provider (or the servers
// typed), and adds it with its password.
func (u Accounts) Register(r Registration) (domain.Account, error) {
	email := strings.TrimSpace(r.Email)
	addr, err := mail.ParseAddress(email)
	if err != nil || !strings.Contains(addr.Address[strings.Index(addr.Address, "@")+1:], ".") {
		return domain.Account{}, ErrInvalidEmail
	}
	email = addr.Address
	prov := ProviderOf(email)
	if prov.OAuth() {
		return domain.Account{}, ErrNeedOAuth
	}
	if r.Password == "" {
		return domain.Account{}, ErrNeedPassword
	}
	name := strings.TrimSpace(r.Name)
	if name == "" {
		name = email
	}
	acc := prov.Preset.Account(name, email)
	acc.ID = domain.Slug(email, "mail")
	if len(acc.ID) > maxSlugFromEmail {
		acc.ID = acc.ID[:maxSlugFromEmail]
	}
	if !prov.Known {
		imap, smtp := strings.TrimSpace(r.IMAPHost), strings.TrimSpace(r.SMTPHost)
		if imap == "" || smtp == "" {
			return domain.Account{}, ErrNeedServers
		}
		acc.Auth = domain.Password
		acc.Services = map[domain.Service][]domain.Endpoint{domain.Mail: {
			{Kind: domain.IMAP, Host: imap, Port: defaultIMAPPort, Security: domain.TLS},
			{Kind: domain.SMTP, Host: smtp, Port: defaultSMTPPort, Security: domain.StartTLS},
		}}
	}
	if err := u.Add(acc, domain.Secret{Password: r.Password}); err != nil {
		return domain.Account{}, err
	}
	return acc, nil
}
