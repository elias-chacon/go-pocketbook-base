// Package oauthcode signs in with OAuth2 by authorization code with PKCE (RFC 7636), for
// the providers whose device flow does not allow the scopes the apps need (Google
// Tasks and Calendar). The reader has no browser: it shows the sign-in address as a QR
// code, the person signs in on a phone, and the provider sends the phone to an address
// that does not exist (http://127.0.0.1) with the code in it; the person types that
// code (or the whole address) on the reader.
package oauthcode

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go-pocketbook-base/internal/accounts/domain"
)

// Doer is the part of http.Client the sign-in needs.
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Provider is where to sign in and what to ask for.
type Provider struct {
	AuthURL, TokenURL string
	ClientID          string
	// ClientSecret is not a secret for a desktop application: Google asks for it anyway.
	ClientSecret string
	Scopes       []string
}

// GoogleScopes are the permissions of the tasks and calendar apps.
var GoogleScopes = []string{
	"https://www.googleapis.com/auth/tasks", "https://www.googleapis.com/auth/calendar",
}

// Google is the provider of Google accounts, for a "Desktop app" client of a Google Cloud project.
func Google(clientID, clientSecret string) Provider {
	return Provider{
		AuthURL: "https://accounts.google.com/o/oauth2/v2/auth", TokenURL: "https://oauth2.googleapis.com/token",
		ClientID: clientID, ClientSecret: clientSecret, Scopes: GoogleScopes,
	}
}

// redirect is where the provider sends the phone: nothing listens there, the address
// bar of the phone shows the code.
const redirect = "http://127.0.0.1"

const maxAnswerBytes = 1 << 20

// ErrNeedCode is what Wait answers: this sign-in ends with the code the person types.
var ErrNeedCode = errors.New("oauth: type the code the provider gave on the phone")

// Client implements domain.CodeAuthorizer for a Provider.
type Client struct {
	HTTP     Doer
	Provider Provider
	// Rand and Now are crypto/rand.Reader and time.Now; tests replace them.
	Rand io.Reader
	Now  func() time.Time
}

func (c Client) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

// Start builds the sign-in address. The challenge keeps the PKCE verifier in DeviceCode.
func (c Client) Start() (domain.Challenge, error) {
	r := c.Rand
	if r == nil {
		r = rand.Reader
	}
	raw := make([]byte, 32)
	if _, err := io.ReadFull(r, raw); err != nil {
		return domain.Challenge{}, err
	}
	verifier := base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(verifier))
	q := url.Values{
		"client_id": {c.Provider.ClientID}, "redirect_uri": {redirect}, "response_type": {"code"},
		"scope": {strings.Join(c.Provider.Scopes, " ")}, "access_type": {"offline"}, "prompt": {"consent"},
		"code_challenge": {base64.RawURLEncoding.EncodeToString(sum[:])}, "code_challenge_method": {"S256"},
	}
	return domain.Challenge{
		VerificationURI: c.Provider.AuthURL + "?" + q.Encode(), ExpiresIn: 600, DeviceCode: verifier,
	}, nil
}

// Wait implements domain.Authorizer: this flow does not poll, it needs Exchange.
func (c Client) Wait(domain.Challenge, func() bool) (domain.Secret, error) {
	return domain.Secret{}, ErrNeedCode
}

// CodeOf finds the code in what the person typed: the code itself, or the address of
// the page the phone could not open (…?code=4/0A…&scope=…).
func CodeOf(typed string) string {
	typed = strings.TrimSpace(typed)
	if i := strings.Index(typed, "code="); i >= 0 {
		rest := typed[i+len("code="):]
		if j := strings.IndexAny(rest, "&# "); j >= 0 {
			rest = rest[:j]
		}
		if v, err := url.QueryUnescape(rest); err == nil {
			return v
		}
		return rest
	}
	return typed
}

// answer is the JSON of the token endpoint.
type answer struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
	Description  string `json:"error_description"`
}

// Exchange trades the code (or the address that holds it) for the tokens.
func (c Client) Exchange(ch domain.Challenge, typed string) (domain.Secret, error) {
	code := CodeOf(typed)
	if code == "" {
		return domain.Secret{}, errors.New("oauth: no code was typed")
	}
	return c.token(url.Values{
		"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {redirect}, "code_verifier": {ch.DeviceCode},
	}, "")
}

// Refresh implements domain.Authorizer.
func (c Client) Refresh(s domain.Secret) (domain.Secret, error) {
	if s.RefreshToken == "" {
		return domain.Secret{}, domain.ErrNoRefresh
	}
	return c.token(url.Values{"grant_type": {"refresh_token"}, "refresh_token": {s.RefreshToken}}, s.RefreshToken)
}

func (c Client) token(form url.Values, keepRefresh string) (domain.Secret, error) {
	form.Set("client_id", c.Provider.ClientID)
	if c.Provider.ClientSecret != "" {
		form.Set("client_secret", c.Provider.ClientSecret)
	}
	req, err := http.NewRequest(http.MethodPost, c.Provider.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return domain.Secret{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return domain.Secret{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxAnswerBytes))
	if err != nil {
		return domain.Secret{}, err
	}
	var a answer
	if err := json.Unmarshal(body, &a); err != nil {
		return domain.Secret{}, fmt.Errorf("oauth: unexpected answer (HTTP %d)", resp.StatusCode)
	}
	if a.Error != "" {
		if a.Description != "" {
			return domain.Secret{}, fmt.Errorf("oauth: %s: %s", a.Error, a.Description)
		}
		return domain.Secret{}, fmt.Errorf("oauth: %s", a.Error)
	}
	s := domain.Secret{AccessToken: a.AccessToken, RefreshToken: a.RefreshToken}
	if s.RefreshToken == "" {
		s.RefreshToken = keepRefresh
	}
	if a.ExpiresIn > 0 {
		s.ExpiresAt = c.now().Add(time.Duration(a.ExpiresIn) * time.Second).Unix()
	}
	return s, nil
}
