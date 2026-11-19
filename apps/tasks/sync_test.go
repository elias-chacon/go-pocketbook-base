package main

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	accounts "go-pocketbook-base/internal/accounts/domain"
)

// fakeWeb answers like Microsoft Graph and Google Tasks, and remembers what was asked.
type fakeWeb struct{ urls []string }

func (f *fakeWeb) RoundTrip(r *http.Request) (*http.Response, error) {
	f.urls = append(f.urls, r.Method+" "+r.URL.String())
	body := `{}`
	switch {
	case strings.HasSuffix(r.URL.Path, "/me/todo/lists"):
		body = `{"value":[{"id":"L1","displayName":"Tasks"}]}`
	case strings.Contains(r.URL.Path, "/me/todo/lists/L1/tasks"):
		body = `{"value":[{"id":"T1","title":"Pay rent","status":"notStarted","importance":"high"}]}`
	case strings.HasSuffix(r.URL.Path, "/users/@me/lists"):
		body = `{"items":[{"id":"G1","title":"My tasks"}]}`
	case strings.Contains(r.URL.Path, "/lists/G1/tasks"):
		body = `{"items":[{"id":"g1","title":"Call","status":"needsAction"}]}`
	}
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
}

func linkedApp(t *testing.T, preset, email string) (*tasks, *fakeWeb) {
	t.Helper()
	web := &fakeWeb{}
	env := testEnv(t)
	env.HTTP = &http.Client{Transport: web}
	a := newTasksAt(env, t.TempDir())
	t.Cleanup(a.close)
	p, _ := accounts.PresetByID(preset)
	acc := p.Account("Ana", email)
	acc.ID = "ana-" + preset
	secret := accounts.Secret{AccessToken: "tok", RefreshToken: "r", ExpiresAt: time.Now().Add(time.Hour).Unix()}
	if err := a.accounts.Add(acc, secret); err != nil {
		t.Fatal(err)
	}
	return a, web
}

func TestSyncWithAnOutlookAccountBringsMicrosoftToDoTasks(t *testing.T) {
	a, web := linkedApp(t, "outlook", "ana@outlook.com")
	svc := service{app: a}
	if svc.Account() != "ana@outlook.com" {
		t.Fatalf("account = %q", svc.Account())
	}
	res, err := svc.Run()
	if err != nil || res.Received != 1 || len(res.Errors) != 0 {
		t.Fatalf("res = %+v, %v (%v)", res, err, web.urls)
	}
	a.screen.SyncNow()
	a.Document().Layout()
	if !strings.Contains(labels(a.Document()), "Pay rent") && !strings.Contains(a.screenNotice(), "Sincronizado") {
		t.Errorf("the synced task is not shown:\n%s", labels(a.Document()))
	}
	if !strings.HasPrefix(web.urls[0], "GET https://graph.microsoft.com/v1.0/me/todo/lists") {
		t.Fatalf("urls = %v", web.urls)
	}
}

func TestSyncWithAGoogleAccountUsesGoogleTasks(t *testing.T) {
	a, web := linkedApp(t, "google", "ana@gmail.com")
	res, err := service{app: a}.Run()
	if err != nil || res.Received != 1 {
		t.Fatalf("res = %+v, %v (%v)", res, err, web.urls)
	}
	if !strings.Contains(web.urls[0], "tasks.googleapis.com") {
		t.Fatalf("urls = %v", web.urls)
	}
}

func TestDisconnectTurnsSyncOffWithoutRemovingTheAccount(t *testing.T) {
	a, _ := linkedApp(t, "outlook", "ana@outlook.com")
	svc := service{app: a}
	if err := svc.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if svc.Account() != "" {
		t.Fatal("still linked after disconnecting")
	}
	if _, err := svc.Run(); err == nil {
		t.Fatal("running without an account must fail")
	}
	if list, _ := a.accounts.List(accounts.Tasks); len(list) != 1 {
		t.Fatal("the account (the mail app may use it) was removed")
	}
	if err := writeLink(a.dir, link{Account: "ana-outlook"}); err != nil || svc.Account() == "" {
		t.Fatalf("choosing the account again must link it: %v", err)
	}
}

func TestConnectShowsTheChooserAndCancelGoesBack(t *testing.T) {
	a, _ := linkedApp(t, "outlook", "ana@outlook.com")
	service{app: a}.Connect()
	a.Document().Layout()
	if !strings.Contains(labels(a.Document()), "Conectar um serviço") {
		t.Fatalf("no chooser:\n%s", labels(a.Document()))
	}
}

func TestBackendAndAuthorizerChoiceByAccount(t *testing.T) {
	if _, err := backendFor(accounts.Account{}, nil, nil); err == nil {
		t.Fatal("an account with no to-do service has no backend")
	}
	a := newAuthorizers(t.TempDir(), nil)
	g, _ := accounts.PresetByID("google")
	if a.forAccount(g.Account("Ana", "a@gmail.com")) != nil {
		t.Fatal("no client id: no authorizer")
	}
}

// screenNotice is the footer line of the list screen.
func (a *tasks) screenNotice() string { return labels(a.Document()) }
