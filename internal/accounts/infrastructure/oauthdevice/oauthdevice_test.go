package oauthdevice

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go-pocketbook-base/internal/accounts/domain"
)

// provider serves the two endpoints. tokenAnswers are returned in turn (the last
// one repeats); each is "status|json".
func provider(t *testing.T, deviceJSON string, tokenAnswers []string, forms *[]url.Values) (Client, *[]time.Duration) {
	t.Helper()
	var calls int32
	mux := http.NewServeMux()
	mux.HandleFunc("/device", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		*forms = append(*forms, r.PostForm)
		fmt.Fprint(w, deviceJSON)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		*forms = append(*forms, r.PostForm)
		i := int(atomic.AddInt32(&calls, 1)) - 1
		if i >= len(tokenAnswers) {
			i = len(tokenAnswers) - 1
		}
		status, body := 200, tokenAnswers[i]
		if k := strings.Index(body, "|"); k > 0 && k < 5 {
			fmt.Sscanf(body[:k], "%d", &status)
			body = body[k+1:]
		}
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	var slept []time.Duration
	now := time.Unix(1_000_000, 0)
	return Client{
		HTTP:     srv.Client(),
		Provider: Provider{DeviceURL: srv.URL + "/device", TokenURL: srv.URL + "/token", ClientID: "app-id", Scopes: []string{"offline_access", "Mail.Send"}},
		Sleep:    func(d time.Duration) { slept = append(slept, d); now = now.Add(d) },
		Now:      func() time.Time { return now },
	}, &slept
}

const device = `{"device_code":"DEV-SECRET","user_code":"ABCD-EFGH","verification_uri":"https://microsoft.com/devicelogin","expires_in":900,"interval":5}`

func TestStartAsksForACodeWithTheScopes(t *testing.T) {
	var forms []url.Values
	c, _ := provider(t, device, nil, &forms)
	ch, err := c.Start()
	if err != nil {
		t.Fatal(err)
	}
	if ch.UserCode != "ABCD-EFGH" || ch.VerificationURI != "https://microsoft.com/devicelogin" || ch.ExpiresIn != 900 || ch.Interval != 5 {
		t.Errorf("%+v", ch)
	}
	if forms[0].Get("client_id") != "app-id" || forms[0].Get("scope") != "offline_access Mail.Send" {
		t.Errorf("the request: %v", forms[0])
	}
	if strings.Contains(ch.String(), "DEV-SECRET") || !strings.Contains(ch.String(), "ABCD-EFGH") {
		t.Errorf("the device code is never shown: %s", ch)
	}
}

func TestStartAcceptsGoogleSpellingAndDefaultsTheInterval(t *testing.T) {
	var forms []url.Values
	c, _ := provider(t, `{"device_code":"d","user_code":"u","verification_url":"https://g.example/device"}`, nil, &forms)
	ch, err := c.Start()
	if err != nil || ch.VerificationURI != "https://g.example/device" || ch.Interval != 5 {
		t.Errorf("%+v %v", ch, err)
	}
}

func TestStartReportsWhatGoesWrong(t *testing.T) {
	var forms []url.Values
	c, _ := provider(t, `{"error":"invalid_client","error_description":"AADSTS700016: app not found\r\nTrace ID: x"}`, nil, &forms)
	if _, err := c.Start(); err == nil || !strings.Contains(err.Error(), "invalid_client") || strings.Contains(err.Error(), "Trace") {
		t.Errorf("the provider's words, first line only: %v", err)
	}
	c, _ = provider(t, `{"user_code":"u"}`, nil, &forms)
	if _, err := c.Start(); err == nil {
		t.Error("an answer without a device code")
	}
	c, _ = provider(t, `<html>`, nil, &forms)
	if _, err := c.Start(); err == nil || !strings.Contains(err.Error(), "unexpected") {
		t.Errorf("%v", err)
	}
	c.Provider.DeviceURL = "http://127.0.0.1:1"
	if _, err := c.Start(); err == nil {
		t.Error("no network")
	}
	c.Provider.DeviceURL = "://bad"
	if _, err := c.Start(); err == nil {
		t.Error("a bad address")
	}
}

func TestWaitPollsUntilApprovedHonoringSlowDown(t *testing.T) {
	var forms []url.Values
	c, slept := provider(t, device, []string{
		`400|{"error":"authorization_pending"}`,
		`400|{"error":"slow_down"}`,
		`400|{"error":"authorization_pending"}`,
		`{"access_token":"AT","refresh_token":"RT","expires_in":3600}`,
	}, &forms)
	ch, _ := c.Start()

	secret, err := c.Wait(ch, nil)
	if err != nil {
		t.Fatal(err)
	}
	if secret.AccessToken != "AT" || secret.RefreshToken != "RT" || secret.ExpiresAt != 1_000_000+30+3600 /* the sleeps: 5+5+10+10 */ {
		t.Errorf("%+v (expires %d)", secret, secret.ExpiresAt)
	}
	want := []time.Duration{5 * time.Second, 5 * time.Second, 10 * time.Second, 10 * time.Second}
	if len(*slept) != 4 {
		t.Fatalf("slept %v", *slept)
	}
	for i, d := range want {
		if (*slept)[i] != d {
			t.Errorf("sleep %d = %v, want %v (slow_down adds 5 s)", i, (*slept)[i], d)
		}
	}
	token := forms[len(forms)-1]
	if token.Get("grant_type") != "urn:ietf:params:oauth:grant-type:device_code" || token.Get("device_code") != "DEV-SECRET" || token.Get("client_id") != "app-id" {
		t.Errorf("the token request: %v", token)
	}
}

func TestWaitEndsInTheWaysTheProviderOrThePersonChoose(t *testing.T) {
	var forms []url.Values
	for answer, want := range map[string]error{
		`400|{"error":"access_denied"}`:          domain.ErrDenied,
		`400|{"error":"authorization_declined"}`: domain.ErrDenied,
		`400|{"error":"expired_token"}`:          domain.ErrExpired,
	} {
		c, _ := provider(t, device, []string{answer}, &forms)
		ch, _ := c.Start()
		if _, err := c.Wait(ch, nil); err != want {
			t.Errorf("%s: %v", answer, err)
		}
	}

	c, _ := provider(t, device, []string{`400|{"error":"authorization_pending"}`}, &forms)
	ch, _ := c.Start()
	ch.ExpiresIn = 12 // the local clock runs out: 5 s, 5 s, 5 s
	if _, err := c.Wait(ch, nil); err != domain.ErrExpired {
		t.Errorf("a code that ran out: %v", err)
	}

	asked := 0
	ch.ExpiresIn = 900
	if _, err := c.Wait(ch, func() bool { asked++; return asked > 2 }); err != domain.ErrCancelled {
		t.Errorf("cancelled: %v", err)
	}

	c, _ = provider(t, device, []string{`400|{"error":"invalid_grant","error_description":"bad grant"}`}, &forms)
	ch, _ = c.Start()
	if _, err := c.Wait(ch, nil); err == nil || !strings.Contains(err.Error(), "invalid_grant") {
		t.Errorf("another error is reported: %v", err)
	}
	c, _ = provider(t, device, []string{`500|oops`}, &forms)
	ch, _ = c.Start()
	if _, err := c.Wait(ch, nil); err == nil {
		t.Error("a server that fails")
	}
	c, _ = provider(t, device, []string{`503|{"nothing":"useful"}`}, &forms)
	ch, _ = c.Start()
	if _, err := c.Wait(ch, nil); err == nil || !strings.Contains(err.Error(), "503") {
		t.Errorf("a failure status without an error field: %v", err)
	}
	ch.Interval = 0 // the default applies
	c.Provider.TokenURL = "http://127.0.0.1:1"
	if _, err := c.Wait(ch, nil); err == nil {
		t.Error("no network")
	}
}

func TestRefreshRenewsTheAccessTokenAndKeepsTheRefreshTokenWhenNoneComes(t *testing.T) {
	var forms []url.Values
	c, _ := provider(t, device, []string{`{"access_token":"NEW","expires_in":60}`}, &forms)
	got, err := c.Refresh(domain.Secret{AccessToken: "OLD", RefreshToken: "RT"})
	if err != nil || got.AccessToken != "NEW" || got.RefreshToken != "RT" || got.ExpiresAt != 1_000_060 {
		t.Fatalf("%+v %v", got, err)
	}
	f := forms[len(forms)-1]
	if f.Get("grant_type") != "refresh_token" || f.Get("refresh_token") != "RT" || f.Get("scope") != "offline_access Mail.Send" {
		t.Errorf("the request: %v", f)
	}

	c, _ = provider(t, device, []string{`{"access_token":"N2","refresh_token":"RT2"}`}, &forms)
	if got, _ := c.Refresh(domain.Secret{RefreshToken: "RT"}); got.RefreshToken != "RT2" || got.ExpiresAt != 0 {
		t.Errorf("a new refresh token replaces the old one: %+v", got)
	}
	c, _ = provider(t, device, []string{`400|{"error":"invalid_grant","error_description":"expired"}`}, &forms)
	if _, err := c.Refresh(domain.Secret{RefreshToken: "RT"}); err == nil || !strings.Contains(err.Error(), "invalid_grant") {
		t.Errorf("%v", err)
	}
	if _, err := c.Refresh(domain.Secret{}); err == nil {
		t.Error("without a refresh token the person has to sign in again")
	}
	c.Provider.TokenURL = "http://127.0.0.1:1"
	if _, err := c.Refresh(domain.Secret{RefreshToken: "RT"}); err == nil {
		t.Error("no network")
	}
}

func TestAClientSecretIsSentOnlyWhenTheProviderNeedsOne(t *testing.T) {
	var forms []url.Values
	c, _ := provider(t, device, []string{`{"access_token":"A"}`}, &forms)
	c.Provider.ClientSecret = "shh"
	ch, _ := c.Start()
	c.Wait(ch, nil)
	c.Refresh(domain.Secret{RefreshToken: "r"})
	withSecret := 0
	for _, f := range forms {
		if f.Get("client_secret") == "shh" {
			withSecret++
		}
	}
	if withSecret != 2 {
		t.Errorf("the token and refresh requests carry it: %d", withSecret)
	}
}

func TestMicrosoftIsAPublicClientOfTheCommonTenantWithTheScopesOfTheApps(t *testing.T) {
	p := Microsoft("my-client")
	if p.ClientID != "my-client" || p.ClientSecret != "" || !strings.HasSuffix(p.DeviceURL, "/oauth2/v2.0/devicecode") || !strings.HasSuffix(p.TokenURL, "/oauth2/v2.0/token") {
		t.Errorf("%+v", p)
	}
	have := strings.Join(p.Scopes, " ")
	for _, scope := range []string{"offline_access", "Mail.ReadWrite", "Mail.Send", "Calendars.ReadWrite", "Tasks.ReadWrite"} {
		if !strings.Contains(have, scope) {
			t.Errorf("the scope %s is missing", scope)
		}
	}
	var _ domain.Authorizer = Client{}
}

func TestTheDefaultClockAndSleepWork(t *testing.T) {
	c := Client{}
	if c.now().IsZero() {
		t.Error("the real clock")
	}
	c.sleep(time.Millisecond) // the real sleep
}
