// Package domain is the heart of the Accounts context: the accounts of the
// person (an e-mail address and what it gives access to) and their secrets, for
// the mail, tasks and calendar apps to share. It knows nothing of files, ciphers
// or networks.
//
// Ubiquitous language:
//
//   - Account: an identity at a provider, e.g. "Pessoal", elias@outlook.com.
//   - Service: what an account is used for: mail, calendar or tasks.
//   - Endpoint: how one Service is reached (IMAP server, SMTP server, a CalDAV
//     address, Microsoft Graph...).
//   - Secret: the password or the OAuth tokens of an account. Never logged.
package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Service is what an account is used for.
type Service string

// The services of the ecosystem.
const (
	Mail     Service = "mail"
	Calendar Service = "calendar"
	Tasks    Service = "tasks"
)

// Kind is how an endpoint is reached.
type Kind string

// The kinds of endpoint.
const (
	IMAP   Kind = "imap"   // incoming mail
	SMTP   Kind = "smtp"   // outgoing mail
	Graph  Kind = "graph"  // Microsoft Graph (Outlook.com, Microsoft 365): mail, calendar, tasks
	EWS    Kind = "ews"    // Exchange Web Services (Exchange on premises)
	CalDAV Kind = "caldav" // calendars and tasks over WebDAV
	Google Kind = "google" // Google's calendar and tasks APIs
)

// Security is how a connection to a server is protected. There is no plain text:
// a password or a token never travels unprotected.
type Security string

// The ways to protect a connection.
const (
	TLS      Security = "tls"      // encrypted from the first byte (IMAP 993, SMTP 465)
	StartTLS Security = "starttls" // upgraded after the greeting (SMTP 587)
)

// Auth is how the person proves who they are.
type Auth string

// The kinds of authentication.
const (
	Password Auth = "password" // a password or an app password
	OAuth2   Auth = "oauth2"   // tokens obtained by the device code flow
)

// Endpoint is how one service of an account is reached.
type Endpoint struct {
	Kind     Kind     `json:"kind"`
	Host     string   `json:"host,omitempty"`
	Port     int      `json:"port,omitempty"`
	Security Security `json:"security,omitempty"`
	URL      string   `json:"url,omitempty"` // CalDAV and EWS: an https address
}

// Account is an identity at a provider and the services it gives access to.
type Account struct {
	ID        string                 `json:"id"` // lower-case slug, unique, names the account's data
	Name      string                 `json:"name"`
	Email     string                 `json:"email"`
	Provider  string                 `json:"provider"` // a preset's ID, "custom" when none
	Auth      Auth                   `json:"auth"`
	Username  string                 `json:"username,omitempty"`  // the login when it is not the address
	Signature string                 `json:"signature,omitempty"` // text added to the end of new messages
	Services  map[Service][]Endpoint `json:"services"`
}

var idPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

// Login is the name to log in with: the username, or the address.
func (a Account) Login() string {
	if a.Username != "" {
		return a.Username
	}
	return a.Email
}

// Has reports whether the account provides a service.
func (a Account) Has(s Service) bool { return len(a.Services[s]) > 0 }

// Endpoint returns the endpoint of a kind for a service.
func (a Account) Endpoint(s Service, k Kind) (Endpoint, bool) {
	for _, e := range a.Services[s] {
		if e.Kind == k {
			return e, true
		}
	}
	return Endpoint{}, false
}

// Validate checks the rules of an account. It returns every problem at once.
func (a Account) Validate() error {
	problems := a.checkIdentity()
	problems = append(problems, a.checkServices()...)
	if len(problems) > 0 {
		return errors.New("account: " + strings.Join(problems, "; "))
	}
	return nil
}

// checkIdentity checks who the account is: its id, name, address and login.
func (a Account) checkIdentity() []string {
	var problems []string
	if !idPattern.MatchString(a.ID) {
		problems = append(problems, fmt.Sprintf("id %q must be lower-case letters, digits, - or _, starting with a letter", a.ID))
	}
	if strings.TrimSpace(a.Name) == "" {
		problems = append(problems, "name is required")
	}
	if at := strings.Index(a.Email, "@"); at < 1 || at == len(a.Email)-1 || strings.ContainsAny(a.Email, " \t") {
		problems = append(problems, fmt.Sprintf("email %q is not an address", a.Email))
	}
	if a.Auth != Password && a.Auth != OAuth2 {
		problems = append(problems, fmt.Sprintf("auth %q must be password or oauth2", a.Auth))
	}
	return problems
}

// checkServices checks the services and their endpoints.
func (a Account) checkServices() []string {
	if len(a.Services) == 0 {
		return []string{"an account needs at least one service"}
	}
	var problems []string
	for s, endpoints := range a.Services {
		if s != Mail && s != Calendar && s != Tasks {
			problems = append(problems, fmt.Sprintf("unknown service %q", s))
		}
		if len(endpoints) == 0 {
			problems = append(problems, fmt.Sprintf("service %s has no endpoint", s))
		}
		for _, e := range endpoints {
			if err := e.validate(s); err != nil {
				problems = append(problems, fmt.Sprintf("%s: %v", s, err))
			}
		}
	}
	return append(problems, a.checkMailPair()...)
}

// checkMailPair: mail over IMAP needs the outgoing server too, and the other way round.
func (a Account) checkMailPair() []string {
	var problems []string
	if !usesIMAP(a.Services[Mail]) {
		return nil
	}
	if _, ok := a.Endpoint(Mail, IMAP); !ok {
		problems = append(problems, "mail needs an imap endpoint")
	}
	if _, ok := a.Endpoint(Mail, SMTP); !ok {
		problems = append(problems, "mail needs an smtp endpoint")
	}
	return problems
}

func usesIMAP(endpoints []Endpoint) bool {
	for _, e := range endpoints {
		if e.Kind == IMAP || e.Kind == SMTP {
			return true
		}
	}
	return false
}

// validate checks one endpoint of a service.
func (e Endpoint) validate(s Service) error {
	switch e.Kind {
	case IMAP, SMTP:
		return e.validateServer(s)
	case CalDAV, EWS:
		return e.validateURL(s)
	case Graph:
		return nil // reached through Microsoft's fixed addresses: nothing to configure
	case Google:
		if s == Mail {
			return fmt.Errorf("google mail is reached by imap and smtp here")
		}
		return nil
	}
	return fmt.Errorf("unknown endpoint kind %q", e.Kind)
}

// validateServer: a mail server has a host, a port and a protected connection.
func (e Endpoint) validateServer(s Service) error {
	switch {
	case s != Mail:
		return fmt.Errorf("%s is only for mail", e.Kind)
	case strings.TrimSpace(e.Host) == "" || e.Port < 1 || e.Port > 65535:
		return fmt.Errorf("%s needs a host and a port (1 to 65535)", e.Kind)
	case e.Security != TLS && e.Security != StartTLS:
		return fmt.Errorf("%s must use tls or starttls, never plain text", e.Kind)
	}
	return nil
}

// validateURL: CalDAV and EWS are reached by an https address.
func (e Endpoint) validateURL(s Service) error {
	switch {
	case !strings.HasPrefix(e.URL, "https://"):
		return fmt.Errorf("%s needs an https address", e.Kind)
	case e.Kind == CalDAV && s == Mail:
		return fmt.Errorf("caldav is not for mail")
	}
	return nil
}

// Secret is what proves an account's owner: a password, or the OAuth2 tokens.
// It is never printed: String and GoString hide it, so a stray log or a %v of an
// error cannot leak it.
type Secret struct {
	Password     string `json:"password,omitempty"`
	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
	ExpiresAt    int64  `json:"expiresAt,omitempty"` // Unix seconds of the access token
}

// String implements fmt.Stringer: it never shows the secret.
func (Secret) String() string { return "Secret{***}" }

// GoString implements fmt.GoStringer: the same for %#v.
func (Secret) GoString() string { return "Secret{***}" }

// Empty reports whether there is nothing in the secret.
func (s Secret) Empty() bool { return s == Secret{} }

// Errors of the ports.
var (
	// ErrNotFound is returned for an account or a secret that does not exist.
	ErrNotFound = errors.New("accounts: not found")
	// ErrLocked is returned by a vault that needs its PIN (or whose PIN is wrong).
	ErrLocked = errors.New("accounts: the vault is locked or the PIN is wrong")
	// ErrNoRefresh is returned when an access token expired and there is nothing to renew it with.
	ErrNoRefresh = errors.New("accounts: the sign-in expired: sign in again")
	// ErrExists is returned when adding an account whose ID is taken.
	ErrExists = errors.New("accounts: an account with this id already exists")
)

// AccountStore is the port that keeps the accounts (not their secrets).
type AccountStore interface {
	List() ([]Account, error)
	Get(id string) (Account, error) // ErrNotFound when there is none
	Save(a Account) error           // creates or replaces
	Delete(id string) error         // ErrNotFound when there is none
}

// Vault is the port that keeps the secrets, encrypted.
type Vault interface {
	Put(accountID string, s Secret) error
	Get(accountID string) (Secret, error) // ErrNotFound, or ErrLocked
	Delete(accountID string) error
}

// Eraser is the port that removes the data an account left behind (its cached
// messages, events and tasks) when the account is removed.
type Eraser interface {
	Erase(accountID string) error
}

// Challenge is what a person has to do to let the reader into an account without
// a browser: open VerificationURI on another device and type UserCode there.
type Challenge struct {
	UserCode        string
	VerificationURI string
	ExpiresIn       int    // seconds the code lives
	Interval        int    // seconds between the reader's checks
	DeviceCode      string // the reader's own handle for the request: never shown
}

// String shows what the person needs, and never the device code.
func (c Challenge) String() string {
	return fmt.Sprintf("open %s and type %s", c.VerificationURI, c.UserCode)
}

// The ways a sign-in by device code ends without tokens.
var (
	ErrDenied    = errors.New("accounts: the sign-in was denied")
	ErrExpired   = errors.New("accounts: the code expired before the sign-in was approved")
	ErrCancelled = errors.New("accounts: the sign-in was cancelled")
)

// Authorizer is the port that signs in with OAuth2 by device code (RFC 8628) and
// renews tokens.
type Authorizer interface {
	// Start asks the provider for a code to show the person.
	Start() (Challenge, error)
	// Wait blocks until the person approved (the tokens), denied, the code expired,
	// or cancelled() says so (it is asked between the checks).
	Wait(c Challenge, cancelled func() bool) (Secret, error)
	// Refresh trades the refresh token of s for a new access token.
	Refresh(s Secret) (Secret, error)
}
