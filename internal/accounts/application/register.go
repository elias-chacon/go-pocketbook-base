package application

import (
	"errors"
	"net/mail"
	"strings"
	"time"

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

// ErrNotOAuth is returned when signing in by code with an address whose provider
// uses a password.
var ErrNotOAuth = errors.New("accounts: this provider does not sign in with OAuth2")

// ErrNoTokens is returned when the sign-in ended without an access token.
var ErrNoTokens = errors.New("accounts: the sign-in gave no access token")

// CheckOAuth asks the provider once whether the person approved the code on the other
// device. pending is true while they have not: ask again later. It waits one polling
// interval, as the provider asks.
func CheckOAuth(auth domain.Authorizer, ch domain.Challenge) (secret domain.Secret, pending bool, err error) {
	asked := 0
	secret, err = auth.Wait(ch, func() bool { asked++; return asked > 1 })
	if errors.Is(err, domain.ErrCancelled) {
		return domain.Secret{}, true, nil
	}
	return secret, false, err
}

// RegisterOAuth adds the account of an address that signs in with OAuth2 (Outlook.com,
// Microsoft 365) with the tokens the sign-in by code gave.
func (u Accounts) RegisterOAuth(r Registration, secret domain.Secret) (domain.Account, error) {
	addr, err := mail.ParseAddress(strings.TrimSpace(r.Email))
	if err != nil || !strings.Contains(addr.Address[strings.Index(addr.Address, "@")+1:], ".") {
		return domain.Account{}, ErrInvalidEmail
	}
	prov := ProviderOf(addr.Address)
	if !prov.OAuth() {
		return domain.Account{}, ErrNotOAuth
	}
	if secret.AccessToken == "" && secret.RefreshToken == "" {
		return domain.Account{}, ErrNoTokens
	}
	name := strings.TrimSpace(r.Name)
	if name == "" {
		name = addr.Address
	}
	acc := prov.Preset.Account(name, addr.Address)
	acc.ID = domain.Slug(addr.Address, "mail")
	if len(acc.ID) > maxSlugFromEmail {
		acc.ID = acc.ID[:maxSlugFromEmail]
	}
	if err := u.Add(acc, secret); err != nil {
		return domain.Account{}, err
	}
	return acc, nil
}

// refreshMargin renews the access token a little before it expires.
const refreshMargin = 60

// AccessToken returns a valid access token of an OAuth2 account, renewing it (and
// keeping the new tokens in the vault) when it expired or is about to.
func (u Accounts) AccessToken(id string, auth domain.Authorizer, now time.Time) (string, error) {
	_, secret, err := u.Credentials(id)
	if err != nil {
		return "", err
	}
	if secret.AccessToken != "" && secret.ExpiresAt-refreshMargin > now.Unix() {
		return secret.AccessToken, nil
	}
	if auth == nil || secret.RefreshToken == "" {
		return "", domain.ErrNoRefresh
	}
	fresh, err := auth.Refresh(secret)
	if err != nil {
		return "", err
	}
	if err := u.Refresh(id, fresh); err != nil {
		return "", err
	}
	return fresh.AccessToken, nil
}
