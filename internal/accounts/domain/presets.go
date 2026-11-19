package domain

import (
	"strings"
	"unicode"
)

// Preset is what is known about a provider: which addresses belong to it and how
// its services are reached, so the person types an address and a password instead
// of server names.
type Preset struct {
	ID      string
	Label   string
	Domains []string // the address domains of this provider ("" for custom)
	auth    Auth
	build   func() map[Service][]Endpoint
}

// Custom is the preset of a provider that is not known: the person gives the
// servers; it starts with no services.
const Custom = "custom"

var presets = []Preset{
	{
		ID: "outlook", Label: "Outlook.com / Microsoft 365",
		Domains: []string{"outlook.com", "outlook.com.br", "hotmail.com", "hotmail.com.br", "live.com", "msn.com"},
		auth:    OAuth2,
		build: func() map[Service][]Endpoint {
			g := []Endpoint{{Kind: Graph}}
			return map[Service][]Endpoint{Mail: g, Calendar: g, Tasks: g}
		},
	},
	{
		ID: "gmail", Label: "Google (Gmail)",
		Domains: []string{"gmail.com", "googlemail.com"},
		// An app password, not OAuth2: Google's device code flow, the only one a reader
		// without a browser can do, does not allow the Gmail or Calendar scopes. The
		// calendar comes in read-only from the account's secret iCal address (T70).
		auth: Password,
		build: func() map[Service][]Endpoint {
			return map[Service][]Endpoint{
				Mail: {
					{Kind: IMAP, Host: "imap.gmail.com", Port: 993, Security: TLS},
					{Kind: SMTP, Host: "smtp.gmail.com", Port: 587, Security: StartTLS},
				},
			}
		},
	},
	{
		ID: "icloud", Label: "iCloud",
		Domains: []string{"icloud.com", "me.com", "mac.com"},
		auth:    Password, // an app-specific password
		build: func() map[Service][]Endpoint {
			dav := []Endpoint{{Kind: CalDAV, URL: "https://caldav.icloud.com"}}
			return map[Service][]Endpoint{
				Mail: {
					{Kind: IMAP, Host: "imap.mail.me.com", Port: 993, Security: TLS},
					{Kind: SMTP, Host: "smtp.mail.me.com", Port: 587, Security: StartTLS},
				},
				Calendar: dav, Tasks: dav,
			}
		},
	},
	{
		ID: "fastmail", Label: "Fastmail",
		Domains: []string{"fastmail.com", "fastmail.fm"},
		auth:    Password,
		build: func() map[Service][]Endpoint {
			dav := []Endpoint{{Kind: CalDAV, URL: "https://caldav.fastmail.com/dav/"}}
			return map[Service][]Endpoint{
				Mail: {
					{Kind: IMAP, Host: "imap.fastmail.com", Port: 993, Security: TLS},
					{Kind: SMTP, Host: "smtp.fastmail.com", Port: 465, Security: TLS},
				},
				Calendar: dav, Tasks: dav,
			}
		},
	},
	{ID: Custom, Label: "IMAP / SMTP / CalDAV / Exchange", auth: Password, build: func() map[Service][]Endpoint { return nil }},
}

// Presets are the providers the ecosystem knows, the custom one last.
func Presets() []Preset { return append([]Preset(nil), presets...) }

// PresetByID finds a preset by its ID.
func PresetByID(id string) (Preset, bool) {
	for _, p := range presets {
		if p.ID == id {
			return p, true
		}
	}
	return Preset{}, false
}

// PresetFor finds the provider of an address by its domain (case-insensitive).
func PresetFor(email string) (Preset, bool) {
	at := strings.LastIndex(email, "@")
	if at < 0 {
		return Preset{}, false
	}
	domain := strings.ToLower(strings.TrimSpace(email[at+1:]))
	for _, p := range presets {
		for _, d := range p.Domains {
			if d == domain {
				return p, true
			}
		}
	}
	return Preset{}, false
}

// Account builds the account of an address at this provider. The custom preset
// gives an account with no services: the caller adds the endpoints.
func (p Preset) Account(name, email string) Account {
	return Account{
		ID: Slug(name, p.ID), Name: name, Email: email, Provider: p.ID,
		Auth: p.auth, Services: p.build(),
	}
}

// Slug turns a name into an account ID: lower-case letters and digits separated
// by "-", starting with a letter, at most 32 characters; fallback when nothing is left.
func Slug(name, fallback string) string {
	var b strings.Builder
	dash := true // no leading dash
	for _, r := range strings.ToLower(name) {
		switch {
		case r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
			dash = false
		case !dash:
			b.WriteByte('-')
			dash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		slug = fallback
	}
	if slug == "" || !unicode.IsLetter(rune(slug[0])) {
		slug = "a" + slug
	}
	if len(slug) > 32 {
		slug = strings.TrimRight(slug[:32], "-")
	}
	return slug
}
