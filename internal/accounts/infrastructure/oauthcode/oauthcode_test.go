package oauthcode

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"go-pocketbook-base/internal/accounts/domain"
)

type fakeHTTP struct {
	status int
	body   string
	form   url.Values
	err    error
}

func (f *fakeHTTP) Do(r *http.Request) (*http.Response, error) {
	if f.err != nil {
		return nil, f.err
	}
	b, _ := io.ReadAll(r.Body)
	f.form, _ = url.ParseQuery(string(b))
	return &http.Response{StatusCode: f.status, Body: io.NopCloser(strings.NewReader(f.body))}, nil
}

func client(h *fakeHTTP) Client {
	return Client{
		HTTP: h, Provider: Google("id", "sec"), Rand: bytes.NewReader(make([]byte, 64)),
		Now: func() time.Time { return time.Unix(1000, 0) },
	}
}

func TestStartBuildsASignInAddressWithPKCE(t *testing.T) {
	ch, err := client(&fakeHTTP{}).Start()
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(ch.VerificationURI)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if q.Get("client_id") != "id" || q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") == "" ||
		q.Get("access_type") != "offline" || !strings.Contains(q.Get("scope"), "tasks") {
		t.Fatalf("query = %v", q)
	}
	if ch.DeviceCode == "" || strings.Contains(ch.VerificationURI, ch.DeviceCode) {
		t.Fatal("the verifier must be kept and never put in the address")
	}
	if _, err := (Client{Provider: Google("i", ""), Rand: bytes.NewReader(nil)}).Start(); err == nil {
		t.Fatal("no randomness must fail")
	}
}

func TestCodeOfAcceptsTheCodeOrTheWholeAddress(t *testing.T) {
	cases := map[string]string{
		"4/0Abc":    "4/0Abc",
		"  4/0Abc ": "4/0Abc",
		"http://127.0.0.1/?code=4%2F0Abc&scope=tasks": "4/0Abc",
		"http://127.0.0.1/?state=x&code=4/0Abc#":      "4/0Abc",
		"http://127.0.0.1/?code=4/0Abc":               "4/0Abc",
	}
	for in, want := range cases {
		if got := CodeOf(in); got != want {
			t.Errorf("CodeOf(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExchangeGivesTheTokens(t *testing.T) {
	h := &fakeHTTP{status: 200, body: `{"access_token":"A","refresh_token":"R","expires_in":3600}`}
	c := client(h)
	s, err := c.Exchange(domain.Challenge{DeviceCode: "ver"}, "http://127.0.0.1/?code=abc")
	if err != nil || s.AccessToken != "A" || s.RefreshToken != "R" || s.ExpiresAt != 4600 {
		t.Fatalf("secret = %+v, %v", s, err)
	}
	if h.form.Get("code") != "abc" || h.form.Get("code_verifier") != "ver" || h.form.Get("client_secret") != "sec" || h.form.Get("grant_type") != "authorization_code" {
		t.Fatalf("form = %v", h.form)
	}
	if _, err := c.Exchange(domain.Challenge{}, "  "); err == nil {
		t.Fatal("an empty code must fail")
	}
}

func TestRefreshKeepsTheRefreshTokenWhenTheAnswerHasNone(t *testing.T) {
	h := &fakeHTTP{status: 200, body: `{"access_token":"B","expires_in":60}`}
	s, err := client(h).Refresh(domain.Secret{RefreshToken: "R"})
	if err != nil || s.AccessToken != "B" || s.RefreshToken != "R" {
		t.Fatalf("secret = %+v, %v", s, err)
	}
	if _, err := client(h).Refresh(domain.Secret{}); !errors.Is(err, domain.ErrNoRefresh) {
		t.Fatalf("err = %v", err)
	}
}

func TestProviderErrorsAreExplainedAndNeverCarryTokens(t *testing.T) {
	for _, h := range []*fakeHTTP{
		{status: 400, body: `{"error":"invalid_grant","error_description":"Bad code"}`},
		{status: 400, body: `{"error":"invalid_client"}`},
		{status: 500, body: `not json`},
		{err: errors.New("offline")},
	} {
		if _, err := client(h).Exchange(domain.Challenge{}, "x"); err == nil {
			t.Fatalf("%+v must fail", h)
		}
	}
	if _, err := client(&fakeHTTP{}).Wait(domain.Challenge{}, nil); !errors.Is(err, ErrNeedCode) {
		t.Fatal("Wait must ask for the code")
	}
}
