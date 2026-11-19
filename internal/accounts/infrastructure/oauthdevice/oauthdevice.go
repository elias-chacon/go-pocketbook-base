// Package oauthdevice signs in with OAuth2 by device code (RFC 8628): the reader
// shows a short code, the person types it on a phone or computer, and the reader
// polls until it gets the tokens. It is how an e-reader without a browser can use
// an account that has no password to type.
package oauthdevice

import (
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

// Doer is what the client needs of an HTTP client (*http.Client satisfies it).
type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Provider describes an OAuth2 server that supports the device code flow.
type Provider struct {
	DeviceURL    string   // the device authorization endpoint
	TokenURL     string   // the token endpoint
	ClientID     string   // of an application registered as a public client
	ClientSecret string   // only for providers that ask for one with a public client
	Scopes       []string // what the app asks to use
}

// MicrosoftScopes are the permissions of the mail, calendar and tasks apps.
var MicrosoftScopes = []string{
	"offline_access", "User.Read", "Mail.ReadWrite", "Mail.Send", "Calendars.ReadWrite", "Tasks.ReadWrite",
}

// Microsoft is the provider of Outlook.com and Microsoft 365 for an application
// registered in Entra ID as a public client with "allow public client flows" on.
func Microsoft(clientID string) Provider {
	base := "https://login.microsoftonline.com/common/oauth2/v2.0/"
	return Provider{DeviceURL: base + "devicecode", TokenURL: base + "token", ClientID: clientID, Scopes: MicrosoftScopes}
}

// Client implements domain.Authorizer for a Provider.
type Client struct {
	HTTP     Doer
	Provider Provider
	// Sleep and Now are time.Sleep and time.Now; tests replace them.
	Sleep func(time.Duration)
	Now   func() time.Time
}

const (
	grantDevice    = "urn:ietf:params:oauth:grant-type:device_code"
	grantRefresh   = "refresh_token"
	defaultPoll    = 5 // seconds, when the provider does not say
	slowDownBy     = 5 // seconds added when the provider says slow_down
	maxAnswerBytes = 1 << 20
)

func (c Client) sleep(d time.Duration) {
	if c.Sleep != nil {
		c.Sleep(d)
		return
	}
	time.Sleep(d)
}

func (c Client) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

// answer is the JSON of both endpoints (the fields of success and of error).
type answer struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	VerificationURL string `json:"verification_url"` // Google's spelling
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`

	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`

	Error       string `json:"error"`
	Description string `json:"error_description"`
}

// post sends a form and decodes the answer; the status is not trusted alone,
// because the token endpoint answers 400 for "authorization_pending".
func (c Client) post(endpoint string, form url.Values) (answer, error) {
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return answer{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return answer{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxAnswerBytes))
	if err != nil {
		return answer{}, err
	}
	var a answer
	if err := json.Unmarshal(body, &a); err != nil {
		return answer{}, fmt.Errorf("oauth: unexpected answer (HTTP %d)", resp.StatusCode)
	}
	if a.Error == "" && resp.StatusCode != http.StatusOK {
		return answer{}, fmt.Errorf("oauth: HTTP %d", resp.StatusCode)
	}
	return a, nil
}

// Start implements domain.Authorizer.
func (c Client) Start() (domain.Challenge, error) {
	form := url.Values{"client_id": {c.Provider.ClientID}, "scope": {strings.Join(c.Provider.Scopes, " ")}}
	a, err := c.post(c.Provider.DeviceURL, form)
	if err != nil {
		return domain.Challenge{}, err
	}
	if a.Error != "" {
		return domain.Challenge{}, providerError(a)
	}
	uri := a.VerificationURI
	if uri == "" {
		uri = a.VerificationURL
	}
	if a.DeviceCode == "" || a.UserCode == "" || uri == "" {
		return domain.Challenge{}, errors.New("oauth: the provider did not give a code")
	}
	if a.Interval <= 0 {
		a.Interval = defaultPoll
	}
	return domain.Challenge{
		UserCode: a.UserCode, VerificationURI: uri, ExpiresIn: a.ExpiresIn, Interval: a.Interval, DeviceCode: a.DeviceCode,
	}, nil
}

// Wait implements domain.Authorizer.
func (c Client) Wait(ch domain.Challenge, cancelled func() bool) (domain.Secret, error) {
	interval := ch.Interval
	if interval <= 0 {
		interval = defaultPoll
	}
	deadline := c.now().Add(time.Duration(ch.ExpiresIn) * time.Second)
	for {
		if cancelled != nil && cancelled() {
			return domain.Secret{}, domain.ErrCancelled
		}
		if ch.ExpiresIn > 0 && !c.now().Before(deadline) {
			return domain.Secret{}, domain.ErrExpired
		}
		c.sleep(time.Duration(interval) * time.Second)

		a, err := c.post(c.Provider.TokenURL, c.deviceForm(ch))
		if err != nil {
			return domain.Secret{}, err
		}
		switch a.Error {
		case "":
			return c.secretOf(a, ""), nil
		case "authorization_pending":
			// not yet: ask again after the interval
		case "slow_down":
			interval += slowDownBy
		case "access_denied", "authorization_declined":
			return domain.Secret{}, domain.ErrDenied
		case "expired_token", "code_expired":
			return domain.Secret{}, domain.ErrExpired
		default:
			return domain.Secret{}, providerError(a)
		}
	}
}

func (c Client) deviceForm(ch domain.Challenge) url.Values {
	form := url.Values{"grant_type": {grantDevice}, "device_code": {ch.DeviceCode}, "client_id": {c.Provider.ClientID}}
	if c.Provider.ClientSecret != "" {
		form.Set("client_secret", c.Provider.ClientSecret)
	}
	return form
}

// Refresh implements domain.Authorizer. The old refresh token is kept when the
// provider does not send a new one.
func (c Client) Refresh(s domain.Secret) (domain.Secret, error) {
	if s.RefreshToken == "" {
		return domain.Secret{}, errors.New("oauth: there is no refresh token: sign in again")
	}
	form := url.Values{
		"grant_type": {grantRefresh}, "refresh_token": {s.RefreshToken}, "client_id": {c.Provider.ClientID},
		"scope": {strings.Join(c.Provider.Scopes, " ")},
	}
	if c.Provider.ClientSecret != "" {
		form.Set("client_secret", c.Provider.ClientSecret)
	}
	a, err := c.post(c.Provider.TokenURL, form)
	if err != nil {
		return domain.Secret{}, err
	}
	if a.Error != "" {
		return domain.Secret{}, providerError(a)
	}
	return c.secretOf(a, s.RefreshToken), nil
}

// secretOf turns a token answer into a Secret; fallbackRefresh is kept when the
// answer has no refresh token.
func (c Client) secretOf(a answer, fallbackRefresh string) domain.Secret {
	s := domain.Secret{AccessToken: a.AccessToken, RefreshToken: a.RefreshToken}
	if s.RefreshToken == "" {
		s.RefreshToken = fallbackRefresh
	}
	if a.ExpiresIn > 0 {
		s.ExpiresAt = c.now().Add(time.Duration(a.ExpiresIn) * time.Second).Unix()
	}
	return s
}

// providerError is the error of an answer. It carries the provider's own words
// (never a token: the error answers have none).
func providerError(a answer) error {
	if a.Description != "" {
		return fmt.Errorf("oauth: %s: %s", a.Error, firstLine(a.Description))
	}
	return fmt.Errorf("oauth: %s", a.Error)
}

func firstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return s[:i]
	}
	return s
}
