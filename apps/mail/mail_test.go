package main

import (
	"go-pocketbook-base/internal/mail/infrastructure/graph"
	"net/http"
	"os"
	"strings"
	"testing"

	accountsapp "go-pocketbook-base/internal/accounts/application"
	accounts "go-pocketbook-base/internal/accounts/domain"
	"go-pocketbook-base/internal/accounts/infrastructure/jsonaccounts"
	"go-pocketbook-base/internal/accounts/infrastructure/securefile"
	"go-pocketbook-base/internal/mail/infrastructure/imapmail"
	"go-pocketbook-base/internal/mail/infrastructure/smtpsend"
	"go-pocketbook-base/internal/platform"
	"go-pocketbook-base/internal/platform/infrastructure/jsonfiles"
	"go-pocketbook-base/internal/toolkit/widget"
)

func testEnv(t *testing.T) platform.Env {
	t.Helper()
	m, err := platform.ParseManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return platform.Env{
		Width: 1404, Height: 1872, Manifest: m, Lang: "pt-BR", Version: "1.0.0",
		Settings: jsonfiles.NewSettings(t.TempDir()), Keyboard: &widget.FakeKeyboard{Reply: "x"}, Exit: func() {},
	}
}

func TestTheManifestFollowsTheContract(t *testing.T) {
	m, err := platform.ParseManifest(manifest)
	if err != nil || m.ID != "mail" {
		t.Fatalf("%+v %v", m, err)
	}
	if m.TitleIn("pt-BR") != "E-mail" || m.TitleIn("en") != "Mail" || m.DescriptionIn("en") == "" {
		t.Errorf("title=%q/%q", m.TitleIn("pt-BR"), m.TitleIn("en"))
	}
}

func labels(d *widget.Document) string {
	var b strings.Builder
	d.Walk(func(w widget.Widget) bool {
		if l, ok := w.(*widget.Label); ok {
			b.WriteString(l.Text + "\n")
		}
		return true
	})
	return b.String()
}

func TestWithoutAnAccountTheAppAsksForOne(t *testing.T) {
	app := newMailAt(testEnv(t), t.TempDir())
	t.Cleanup(app.close)
	app.Document().Layout()
	if app.screen != nil {
		t.Error("a mailbox without an account")
	}
	if !strings.Contains(labels(app.Document()), "Adicionar conta") {
		t.Errorf("no sign-in screen:\n%s", labels(app.Document()))
	}
	if app.Key(platform.KeyNext) {
		t.Error("a key was taken by the sign-in screen")
	}
}

func addAccount(t *testing.T, dir string) {
	t.Helper()
	u := accountsapp.Accounts{Store: jsonaccounts.New(dir), Vault: securefile.New(dir)}
	if _, err := u.Register(accountsapp.Registration{Email: "ana@gmail.com", Password: "app-pw"}); err != nil {
		t.Fatal(err)
	}
}

func TestWithAnAccountTheMailboxOpensAndTheKeysTurnPages(t *testing.T) {
	dir := t.TempDir()
	addAccount(t, dir)
	app := newMailAt(testEnv(t), dir)
	t.Cleanup(app.close)
	app.Document().Layout()
	if app.screen == nil {
		t.Fatalf("no mailbox:\n%s", labels(app.Document()))
	}
	if !strings.Contains(labels(app.Document()), "Nenhuma mensagem") {
		t.Errorf("an empty cache should say so:\n%s", labels(app.Document()))
	}
	if !app.Key(platform.KeyNext) || !app.Key(platform.KeyPrevious) || app.Key(platform.KeyMenu) {
		t.Error("the page keys must be handled and the menu key must not")
	}
}

func TestAnUnusableDataFolderShowsTheErrorInsteadOfCrashing(t *testing.T) {
	file := t.TempDir() + "/not-a-folder"
	if err := writeFile(file); err != nil {
		t.Fatal(err)
	}
	app := newMailAt(testEnv(t), file+"/sub")
	t.Cleanup(app.close)
	app.Document().Layout()
	if app.screen != nil || labels(app.Document()) == "" {
		t.Errorf("expected an error text, got %q", labels(app.Document()))
	}
}

func TestBackendForGmailUsesImapAndSmtpWithoutSavingSent(t *testing.T) {
	acc, _ := accounts.PresetByID("gmail")
	a := acc.Account("Ana", "ana@gmail.com")
	b, err := backendFor(a, accounts.Secret{Password: "pw"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	c := b.(*imapmail.Client)
	if c.Addr != "imap.gmail.com:993" || c.Security != imapmail.TLS || c.SaveSent {
		t.Errorf("client = %+v", c)
	}
	s := c.Sender.(smtpsend.Sender)
	if s.Addr != "smtp.gmail.com:587" || s.Security != smtpsend.StartTLS || s.Auth == nil {
		t.Errorf("sender = %+v", s)
	}
}

func TestBackendForGoogleOAuthUsesImapWithXOAuth2AndTheTokenOfTheMoment(t *testing.T) {
	p, _ := accounts.PresetByID("google")
	a := p.Account("Ana", "ana@gmail.com")
	calls := 0
	b, err := backendFor(a, accounts.Secret{}, func() (string, error) { calls++; return "tok", nil }, nil)
	if err != nil {
		t.Fatal(err)
	}
	c, ok := b.(*imapmail.Client)
	if !ok || c.Addr != "imap.gmail.com:993" || c.SignIn == nil || c.SaveSent {
		t.Fatalf("client = %T %+v", b, b)
	}
	s := c.Sender.(smtpsend.Sender)
	if s.Addr != "smtp.gmail.com:587" || s.Auth == nil {
		t.Errorf("sender = %+v", s)
	}
	if calls != 0 {
		t.Error("the token is asked when the connection is made, not before")
	}
	if _, err := backendFor(a, accounts.Secret{}, nil, nil); err == nil {
		t.Error("an OAuth2 account without a token source was accepted")
	}
}

func TestBackendForRefusesWhatItCannotDo(t *testing.T) {
	p, _ := accounts.PresetByID("outlook")
	if _, err := backendFor(p.Account("Ana", "ana@outlook.com"), accounts.Secret{}, nil, nil); err == nil {
		t.Error("an OAuth2 account was accepted")
	}
	if _, err := backendFor(accounts.Account{Auth: accounts.Password}, accounts.Secret{}, nil, nil); err == nil {
		t.Error("an account without an IMAP server was accepted")
	}
	c, _ := accounts.PresetByID("icloud")
	b, _ := backendFor(c.Account("Ana", "ana@icloud.com"), accounts.Secret{Password: "x"}, nil, nil)
	if !b.(*imapmail.Client).SaveSent {
		t.Error("iCloud does not keep a copy of what is sent: the app must")
	}
}

func writeFile(path string) error { return os.WriteFile(path, []byte("x"), 0o644) }

func TestEveryScreenHasTheStandardAboutAndExitMenu(t *testing.T) {
	exited := false
	env := testEnv(t)
	env.Exit = func() { exited = true }
	app := newMailAt(env, t.TempDir())
	t.Cleanup(app.close)
	app.Document().Layout()
	var bar *widget.MenuBar
	app.Document().Walk(func(w widget.Widget) bool {
		if b, ok := w.(*widget.MenuBar); ok {
			bar = b
		}
		return true
	})
	if bar == nil || bar.Menus[0].Label != "E-mail" {
		t.Fatalf("no standard bar: %v", bar)
	}
	items := bar.Menus[0].Items
	if len(items) != 3 || items[0].Label != "Sobre" || items[2].Label != "Sair" {
		t.Fatalf("items = %+v", items)
	}
	items[0].OnSelect()
	app.Document().Layout()
	found := false
	app.Document().Walk(func(w widget.Widget) bool {
		found = found || w.Elem().ID == "modal-about"
		return true
	})
	if !found {
		t.Error("About did not open")
	}
	items[2].OnSelect()
	if !exited {
		t.Error("Exit did not leave the app")
	}
}

func TestAnOutlookAccountUsesGraphWithTheRenewedToken(t *testing.T) {
	p, _ := accounts.PresetByID("outlook")
	acc := p.Account("Ana", "ana@outlook.com")
	calls := 0
	b, err := backendFor(acc, accounts.Secret{}, func() (string, error) { calls++; return "tok", nil }, http.DefaultClient)
	if err != nil {
		t.Fatal(err)
	}
	c, ok := b.(graph.Client)
	if !ok {
		t.Fatalf("backend = %T", b)
	}
	if tok, _ := c.Token(); tok != "tok" || calls != 1 {
		t.Errorf("token = %q calls = %d", tok, calls)
	}
	if _, err := backendFor(acc, accounts.Secret{}, nil, nil); err == nil {
		t.Error("an OAuth2 account without a token source was accepted")
	}
}

func TestTheClientIDComesFromTheDataFolder(t *testing.T) {
	dir := t.TempDir()
	if authorizer(dir, nil) != nil {
		t.Error("no file must mean no OAuth")
	}
	os.WriteFile(dir+"/oauth.json", []byte(`{"microsoft": " abc-123 "}`), 0o644)
	if authorizer(dir, nil) == nil {
		t.Error("a client id must enable the sign-in")
	}
	os.WriteFile(dir+"/oauth.json", []byte(`not json`), 0o644)
	if authorizer(dir, nil) != nil {
		t.Error("a broken file must be ignored")
	}
	if httpClient(nil) == nil {
		t.Error("no default client")
	}
}

func TestWithAClientIDTheSignInOffersOutlook(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(dir+"/oauth.json", []byte(`{"microsoft":"id"}`), 0o644)
	app := newMailAt(testEnv(t), dir)
	t.Cleanup(app.close)
	if app.auth == nil {
		t.Fatal("the authorizer was not built")
	}
}
