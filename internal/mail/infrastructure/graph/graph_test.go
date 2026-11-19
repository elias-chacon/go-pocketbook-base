package graph

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-pocketbook-base/internal/mail/domain"
)

// fake is a Graph server that records the requests it gets.
type fake struct {
	t        *testing.T
	requests []string
	bodies   []string
	handler  map[string]func(w http.ResponseWriter, r *http.Request)
	token    string
}

func newFake(t *testing.T) (*fake, Client) {
	t.Helper()
	f := &fake{t: t, handler: map[string]func(http.ResponseWriter, *http.Request){}, token: "TOKEN-123"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.requests = append(f.requests, r.Method+" "+r.URL.Path)
		f.bodies = append(f.bodies, string(body))
		if got := r.Header.Get("Authorization"); got != "Bearer "+f.token {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"error":{"code":"InvalidAuthenticationToken","message":"expired"}}`)
			return
		}
		if h, ok := f.handler[r.Method+" "+r.URL.Path]; ok {
			h(w, r)
			return
		}
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":{"code":"ErrorItemNotFound","message":"The specified object was not found.\nTrace: x"}}`)
	}))
	t.Cleanup(srv.Close)
	return f, Client{HTTP: srv.Client(), Base: srv.URL, Token: func() (string, error) { return f.token, nil }}
}

func (f *fake) json(route, body string) {
	f.handler[route] = func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }
}

func TestFoldersGetTheirRolesFromTheWellKnownOnes(t *testing.T) {
	f, c := newFake(t)
	f.json("GET /me/mailFolders/inbox", `{"id":"AAA","displayName":"Caixa de Entrada"}`)
	f.json("GET /me/mailFolders/sentitems", `{"id":"BBB","displayName":"Itens Enviados"}`)
	f.json("GET /me/mailFolders/deleteditems", `{"id":"CCC","displayName":"Itens Excluídos"}`)
	f.json("GET /me/mailFolders", `{"value":[
		{"id":"AAA","displayName":"Caixa de Entrada"},{"id":"BBB","displayName":"Itens Enviados"},
		{"id":"CCC","displayName":"Itens Excluídos"},{"id":"DDD","displayName":"Projetos"}]}`)

	got, err := c.Folders()
	if err != nil || len(got) != 4 {
		t.Fatalf("%+v %v", got, err)
	}
	roles := map[string]domain.FolderRole{}
	for _, fl := range got {
		roles[fl.ID] = fl.Role
	}
	if roles["AAA"] != domain.Inbox || roles["BBB"] != domain.Sent || roles["CCC"] != domain.Trash || roles["DDD"] != domain.Other {
		t.Errorf("%v", roles)
	}
	if got[0].Name != "Caixa de Entrada" {
		t.Errorf("%+v", got[0])
	}
}

func TestFoldersFollowThePagesOfALongList(t *testing.T) {
	f, c := newFake(t)
	calls := 0
	f.handler["GET /me/mailFolders"] = func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			fmt.Fprintf(w, `{"value":[{"id":"1","displayName":"A"}],"@odata.nextLink":"%s/me/mailFolders?$skip=1"}`, c.Base)
			return
		}
		fmt.Fprint(w, `{"value":[{"id":"2","displayName":"B"}]}`)
	}
	got, err := c.Folders()
	if err != nil || len(got) != 2 || got[1].Name != "B" {
		t.Errorf("%+v %v", got, err)
	}
	// a next link to somewhere else is not followed (the token must not be sent elsewhere)
	f.handler["GET /me/mailFolders"] = func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"value":[{"id":"1","displayName":"A"}],"@odata.nextLink":"https://evil.example/steal"}`)
	}
	if got, err := c.Folders(); err != nil || len(got) != 1 {
		t.Errorf("%+v %v", got, err)
	}
}

func TestRecentMapsGraphMessagesToTheDomain(t *testing.T) {
	f, c := newFake(t)
	f.handler["GET /me/mailFolders/AAA/messages"] = func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("$top") != "2" || q.Get("$orderby") != "receivedDateTime desc" || !strings.Contains(q.Get("$select"), "bodyPreview") {
			t.Errorf("the query: %v", q)
		}
		fmt.Fprint(w, `{"value":[
		 {"id":"m2","internetMessageId":"<m2@x>","subject":"Reunião","bodyPreview":"  Podemos às 10h?  ","receivedDateTime":"2026-01-05T10:00:00Z",
		  "isRead":true,"hasAttachments":true,"flag":{"flagStatus":"flagged"},
		  "from":{"emailAddress":{"name":"Ana Souza","address":"ana@example.org"}},
		  "toRecipients":[{"emailAddress":{"name":"Eu","address":"eu@example.org"}}],
		  "ccRecipients":[{"emailAddress":{"address":"dir@example.org"}}],"replyTo":[{"emailAddress":{"address":"lista@example.org"}}]},
		 {"id":"m1","subject":"Antiga","receivedDateTime":"2026-01-04T08:00:00+02:00","isRead":false,"flag":{"flagStatus":"notFlagged"},
		  "from":{"emailAddress":{"name":"","address":"bia@example.org"}},"toRecipients":[],"ccRecipients":[],"replyTo":[]}]}`)
	}
	got, err := c.Recent("AAA", 2)
	if err != nil || len(got) != 2 {
		t.Fatalf("%+v %v", got, err)
	}
	m := got[0]
	if m.RemoteID != "m2" || m.FolderID != "AAA" || m.MessageID != "<m2@x>" || m.Subject != "Reunião" || m.Preview != "Podemos às 10h?" {
		t.Errorf("%+v", m)
	}
	if !m.Has(domain.Seen) || !m.Has(domain.Flagged) || m.From.Name != "Ana Souza" || len(m.To) != 1 || m.Cc[0].Email != "dir@example.org" || m.ReplyTo[0].Email != "lista@example.org" {
		t.Errorf("%+v", m)
	}
	if m.Date.Hour() != 10 || len(m.Attachments) != 1 {
		t.Errorf("%v %+v", m.Date, m.Attachments)
	}
	old := got[1]
	if old.Has(domain.Seen) || old.Has(domain.Flagged) || old.From.Email != "bia@example.org" || old.Date.Hour() != 6 || len(old.Attachments) != 0 {
		t.Errorf("an unread, unflagged message with a time zone offset: %+v", old)
	}
}

func TestRecentBoundsTheRequestedPage(t *testing.T) {
	f, c := newFake(t)
	var top string
	f.handler["GET /me/mailFolders/AAA/messages"] = func(w http.ResponseWriter, r *http.Request) {
		top = r.URL.Query().Get("$top")
		fmt.Fprint(w, `{"value":[]}`)
	}
	c.Recent("AAA", 0)
	if top != "100" {
		t.Errorf("%s", top)
	}
	c.Recent("AAA", 5000)
	if top != "100" {
		t.Errorf("%s", top)
	}
}

const mimeMessage = "From: Ana <ana@example.org>\r\nTo: eu@example.org\r\nSubject: Contrato\r\nContent-Type: multipart/mixed; boundary=\"M\"\r\n\r\n" +
	"--M\r\nContent-Type: text/plain; charset=utf-8\r\n\r\nSegue o contrato.\r\n--M\r\nContent-Type: application/pdf; name=\"c.pdf\"\r\n" +
	"Content-Disposition: attachment; filename=\"c.pdf\"\r\nContent-Transfer-Encoding: base64\r\n\r\nSGVsbG8=\r\n--M--\r\n"

func TestFetchReadsTheMessageAsMIME(t *testing.T) {
	f, c := newFake(t)
	f.handler["GET /me/messages/m1/$value"] = func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, mimeMessage) }
	m, err := c.Fetch("AAA", "m1")
	if err != nil || m.Subject != "Contrato" || m.Body != "Segue o contrato." || m.RemoteID != "m1" || m.FolderID != "AAA" {
		t.Fatalf("%+v %v", m, err)
	}
	if len(m.Attachments) != 1 || m.Attachments[0].Name != "c.pdf" {
		t.Errorf("%+v", m.Attachments)
	}
	f.handler["GET /me/messages/bad/$value"] = func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "") }
	if _, err := c.Fetch("AAA", "bad"); err == nil {
		t.Error("an answer that is not a message")
	}
	if _, err := c.Fetch("AAA", "ghost"); err == nil || !strings.Contains(err.Error(), "ErrorItemNotFound") {
		t.Errorf("%v", err)
	}
}

func TestAMessageTooBigToOpenIsRefusedWithoutReadingItAll(t *testing.T) {
	f, c := newFake(t)
	f.handler["GET /me/messages/big/$value"] = func(w http.ResponseWriter, r *http.Request) {
		chunk := strings.Repeat("x", 1<<20)
		for i := 0; i < 12; i++ {
			fmt.Fprint(w, chunk)
		}
	}
	if _, err := c.Fetch("AAA", "big"); err == nil || !strings.Contains(err.Error(), "too big") {
		t.Errorf("%v", err)
	}
}

func TestSendPostsTheMIMEInBase64WithTheBccRecipients(t *testing.T) {
	f, c := newFake(t)
	f.handler["POST /me/sendMail"] = func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "text/plain" {
			t.Errorf("content type %q", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusAccepted)
	}
	d := domain.Draft{
		From: domain.Address{Name: "Eu", Email: "eu@example.org"}, To: []domain.Address{{Email: "ana@example.org"}},
		Bcc: []domain.Address{{Name: "Secreta", Email: "s@example.org"}}, Subject: "Oi", Body: "Olá!",
	}
	if err := c.Send(d); err != nil {
		t.Fatal(err)
	}
	raw, err := base64.StdEncoding.DecodeString(f.bodies[len(f.bodies)-1])
	if err != nil {
		t.Fatalf("the body is base64: %v", err)
	}
	text := string(raw)
	for _, want := range []string{"Bcc: ", "s@example.org", "To: ", "ana@example.org", "Subject: Oi", "Ol=C3=A1!"} {
		if !strings.Contains(text, want) {
			t.Errorf("the MIME lacks %q:\n%s", want, text)
		}
	}
	d.Bcc = nil
	f.bodies = nil
	if err := c.Send(d); err != nil {
		t.Fatal(err)
	}
	raw, _ = base64.StdEncoding.DecodeString(f.bodies[0])
	if strings.Contains(string(raw), "Bcc") {
		t.Error("no Bcc header without Bcc recipients")
	}
	if err := c.Send(domain.Draft{}); err == nil {
		t.Error("a draft that cannot be sent is not posted")
	}
}

func TestMarkingReadAndDeleting(t *testing.T) {
	f, c := newFake(t)
	f.handler["PATCH /me/messages/m1"] = func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{}`) }
	f.handler["DELETE /me/messages/m1"] = func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }
	if err := c.SetRead("AAA", "m1", true); err != nil {
		t.Fatal(err)
	}
	if f.bodies[len(f.bodies)-1] != `{"isRead":true}` {
		t.Errorf("%s", f.bodies[len(f.bodies)-1])
	}
	c.SetRead("AAA", "m1", false)
	if f.bodies[len(f.bodies)-1] != `{"isRead":false}` {
		t.Errorf("%s", f.bodies[len(f.bodies)-1])
	}
	if err := c.Delete("AAA", "m1"); err != nil {
		t.Fatal(err)
	}
	if err := c.Delete("AAA", "ghost"); err == nil {
		t.Error("a message that does not exist")
	}
}

func TestIDsWithSpecialCharactersAreEscaped(t *testing.T) {
	f, c := newFake(t)
	f.handler["DELETE /me/messages/AAMk=/x+y"] = func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }
	c.Delete("AAA", "AAMk=/x+y")
	if len(f.requests) != 1 || strings.Contains(f.requests[0], "//") {
		t.Errorf("%v", f.requests)
	}
}

func TestAnExpiredTokenIsAnUnauthorizedError(t *testing.T) {
	f, c := newFake(t)
	f.token = "NEW"
	c.Token = func() (string, error) { return "OLD", nil } // the client still holds the expired one
	if _, err := c.Recent("AAA", 5); err != domain.ErrUnauthorized {
		t.Errorf("%v", err)
	}
	if _, err := c.Folders(); err != domain.ErrUnauthorized {
		t.Errorf("Folders: %v", err)
	}
	if err := c.SetRead("AAA", "m", true); err != domain.ErrUnauthorized {
		t.Errorf("%v", err)
	}
}

func TestFailuresAreReportedWithoutLeakingTheToken(t *testing.T) {
	f, c := newFake(t)
	_, err := c.Recent("ghost", 5)
	if err == nil || !strings.Contains(err.Error(), "ErrorItemNotFound") || strings.Contains(err.Error(), "TOKEN") || strings.Contains(err.Error(), "Trace") {
		t.Errorf("%v", err)
	}
	f.handler["GET /me/mailFolders/AAA/messages"] = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "<html>")
	}
	if _, err := c.Recent("AAA", 5); err == nil || !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("%v", err)
	}
	f.handler["GET /me/mailFolders/BBB/messages"] = func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "not json") }
	if _, err := c.Recent("BBB", 5); err == nil || !strings.Contains(err.Error(), "unexpected") {
		t.Errorf("%v", err)
	}
	f.handler["GET /me/mailFolders"] = func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "not json") }
	if _, err := c.Folders(); err == nil {
		t.Error("a folder list that is not JSON")
	}

	broken := c
	broken.Token = func() (string, error) { return "", errors.New("no token") }
	if _, err := broken.Recent("AAA", 5); err == nil || err.Error() != "no token" {
		t.Errorf("a token that cannot be had: %v", err)
	}
	broken = c
	broken.Base = "http://127.0.0.1:1"
	if _, err := broken.Recent("AAA", 5); err == nil {
		t.Error("no network")
	}
	broken.Base = "://bad"
	if _, err := broken.Recent("AAA", 5); err == nil {
		t.Error("a bad address")
	}
}

func TestTheDefaultBaseIsGraph(t *testing.T) {
	if (Client{}).base() != "https://graph.microsoft.com/v1.0" {
		t.Error((Client{}).base())
	}
}

func TestFlaggingSendsTheFlagStatus(t *testing.T) {
	f, c := newFake(t)
	f.handler["PATCH /me/messages/m1"] = func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{}`) }
	if err := c.SetFlagged("AAA", "m1", true); err != nil {
		t.Fatal(err)
	}
	if f.bodies[len(f.bodies)-1] != `{"flag":{"flagStatus":"flagged"}}` {
		t.Errorf("%s", f.bodies[len(f.bodies)-1])
	}
	c.SetFlagged("AAA", "m1", false)
	if f.bodies[len(f.bodies)-1] != `{"flag":{"flagStatus":"notFlagged"}}` {
		t.Errorf("%s", f.bodies[len(f.bodies)-1])
	}
}
