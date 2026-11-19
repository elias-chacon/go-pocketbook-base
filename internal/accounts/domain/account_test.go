package domain

import (
	"fmt"
	"strings"
	"testing"
)

func TestEveryPresetBuildsAValidAccountExceptTheCustomOne(t *testing.T) {
	for _, p := range Presets() {
		a := p.Account("Pessoal", "ana@example.org")
		err := a.Validate()
		if p.ID == Custom {
			if err == nil || len(a.Services) != 0 {
				t.Errorf("custom has no services yet: %v", err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: %v", p.ID, err)
		}
		if a.Provider != p.ID || !a.Has(Mail) || !a.Has(Calendar) || !a.Has(Tasks) {
			t.Errorf("%s serves mail, calendar and tasks: %+v", p.ID, a.Services)
		}
	}
	if ps := Presets(); ps[len(ps)-1].ID != Custom {
		t.Error("the custom preset is last")
	}
}

func TestAProviderIsFoundByTheDomainOfTheAddress(t *testing.T) {
	cases := map[string]string{
		"a@outlook.com": "outlook", "B@HOTMAIL.COM": "outlook", "c@gmail.com": "gmail",
		"d@icloud.com": "icloud", "e@Fastmail.fm": "fastmail",
	}
	for email, want := range cases {
		if p, ok := PresetFor(email); !ok || p.ID != want {
			t.Errorf("%s: %v %v", email, p.ID, ok)
		}
	}
	for _, email := range []string{"x@unknown.example", "no-at-sign", ""} {
		if _, ok := PresetFor(email); ok {
			t.Errorf("%q has no preset", email)
		}
	}
	if p, ok := PresetByID("icloud"); !ok || p.Label != "iCloud" {
		t.Error("by id")
	}
	if _, ok := PresetByID("nope"); ok {
		t.Error("an unknown id")
	}
}

func TestPresetsUseTheRightAuthAndEndpoints(t *testing.T) {
	out, _ := PresetByID("outlook")
	if a := out.Account("MS", "a@outlook.com"); a.Auth != OAuth2 || a.Services[Mail][0].Kind != Graph {
		t.Errorf("outlook is oauth2 over graph: %+v", a)
	}
	ic, _ := PresetByID("icloud")
	a := ic.Account("Apple", "a@icloud.com")
	if a.Auth != Password {
		t.Error("icloud uses an app password")
	}
	if e, ok := a.Endpoint(Mail, IMAP); !ok || e.Host != "imap.mail.me.com" || e.Port != 993 || e.Security != TLS {
		t.Errorf("%+v", e)
	}
	if e, ok := a.Endpoint(Calendar, CalDAV); !ok || e.URL != "https://caldav.icloud.com" {
		t.Errorf("%+v", e)
	}
	if _, ok := a.Endpoint(Mail, Graph); ok {
		t.Error("no graph endpoint for icloud")
	}
	if a.Login() != "a@icloud.com" {
		t.Error("the login is the address by default")
	}
	a.Username = "ana"
	if a.Login() != "ana" {
		t.Error("or the username")
	}
}

func TestValidateReportsEveryProblemAtOnce(t *testing.T) {
	bad := Account{ID: "Bad Id", Name: " ", Email: "nobody", Auth: "token", Services: map[Service][]Endpoint{
		Mail:     {{Kind: IMAP, Host: "", Port: 0}},
		"chat":   {{Kind: Graph}},
		Calendar: {{Kind: CalDAV, URL: "http://insecure.example"}},
		Tasks:    nil,
	}}
	err := bad.Validate()
	if err == nil {
		t.Fatal("must be invalid")
	}
	for _, want := range []string{"id", "name is required", "not an address", "auth", "unknown service", "https", "needs a host", "no endpoint", "smtp endpoint"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the message lacks %q: %v", want, err)
		}
	}
	if (Account{ID: "x", Name: "n", Email: "a@b.c", Auth: Password}).Validate() == nil {
		t.Error("an account without services is invalid")
	}
}

func TestEndpointRules(t *testing.T) {
	cases := []struct {
		name    string
		service Service
		e       Endpoint
		ok      bool
	}{
		{"imap over tls", Mail, Endpoint{Kind: IMAP, Host: "h", Port: 993, Security: TLS}, true},
		{"smtp starttls", Mail, Endpoint{Kind: SMTP, Host: "h", Port: 587, Security: StartTLS}, true},
		{"imap plain text", Mail, Endpoint{Kind: IMAP, Host: "h", Port: 143}, false},
		{"imap port out of range", Mail, Endpoint{Kind: IMAP, Host: "h", Port: 70000, Security: TLS}, false},
		{"imap for calendar", Calendar, Endpoint{Kind: IMAP, Host: "h", Port: 993, Security: TLS}, false},
		{"caldav https", Calendar, Endpoint{Kind: CalDAV, URL: "https://d.example/dav"}, true},
		{"caldav for mail", Mail, Endpoint{Kind: CalDAV, URL: "https://d.example"}, false},
		{"ews https", Mail, Endpoint{Kind: EWS, URL: "https://mail.example/EWS/Exchange.asmx"}, true},
		{"ews without url", Mail, Endpoint{Kind: EWS}, false},
		{"graph needs nothing", Tasks, Endpoint{Kind: Graph}, true},
		{"google tasks", Tasks, Endpoint{Kind: Google}, true},
		{"google mail is imap here", Mail, Endpoint{Kind: Google}, false},
		{"unknown kind", Mail, Endpoint{Kind: "pigeon"}, false},
	}
	for _, c := range cases {
		if err := c.e.validate(c.service); (err == nil) != c.ok {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}

func TestSlugsAreSafeIDs(t *testing.T) {
	cases := map[string]string{
		"Pessoal":               "pessoal",
		"Trabalho da Ana!":      "trabalho-da-ana",
		"  --x--  ":             "x",
		"123 contas":            "a123-contas",
		"":                      "outlook",
		"Ação":                  "a-o",
		strings.Repeat("a", 50): strings.Repeat("a", 32),
	}
	for in, want := range cases {
		if got := Slug(in, "outlook"); got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
		if !idPattern.MatchString(Slug(in, "outlook")) {
			t.Errorf("Slug(%q) is not a valid id", in)
		}
	}
	if got := Slug("???", ""); got != "a" {
		t.Errorf("nothing left and no fallback: %q", got)
	}
}

func TestASecretNeverShowsItself(t *testing.T) {
	s := Secret{Password: "hunter2", AccessToken: "tok-abc", RefreshToken: "ref-xyz"}
	for _, text := range []string{fmt.Sprint(s), fmt.Sprintf("%v", s), fmt.Sprintf("%+v", s), fmt.Sprintf("%#v", s), fmt.Sprintf("%s", s)} {
		if strings.Contains(text, "hunter2") || strings.Contains(text, "tok-abc") || strings.Contains(text, "ref-xyz") {
			t.Errorf("a secret leaked: %s", text)
		}
	}
	if err := fmt.Errorf("could not log in: %v", s); strings.Contains(err.Error(), "hunter2") {
		t.Error("a secret leaked through an error")
	}
	if (Secret{}).Empty() == false || s.Empty() {
		t.Error("Empty")
	}
}
