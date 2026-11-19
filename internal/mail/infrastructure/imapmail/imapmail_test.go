package imapmail

import (
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/backend/memory"
	"github.com/emersion/go-imap/server"

	"go-pocketbook-base/internal/mail/domain"
)

// startServer runs go-imap's in-memory server: one user, an INBOX with one message.
func startServer(t *testing.T) *Client {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := server.New(memory.New())
	s.AllowInsecureAuth = true
	go func() { _ = s.Serve(ln) }()
	t.Cleanup(func() { _ = s.Close() })
	c := &Client{Addr: ln.Addr().String(), Security: NoTLS, Timeout: 5 * time.Second, SignIn: Password("username", "password")}
	t.Cleanup(c.Close)
	return c
}

func TestFoldersGivesTheInboxItsRole(t *testing.T) {
	c := startServer(t)
	folders, err := c.Folders()
	if err != nil {
		t.Fatal(err)
	}
	if len(folders) != 1 || folders[0].ID != "INBOX" || folders[0].Role != domain.Inbox {
		t.Errorf("folders = %+v", folders)
	}
}

func TestRecentReadsTheHeaders(t *testing.T) {
	c := startServer(t)
	list, err := c.Recent("INBOX", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("got %d messages", len(list))
	}
	m := list[0]
	if m.Subject != "A little message, just for you" || m.From.Email != "contact@example.org" {
		t.Errorf("message = %+v", m)
	}
	if m.RemoteID != "6" || m.FolderID != "INBOX" || !m.Has(domain.Seen) {
		t.Errorf("ids/flags = %q %q %v", m.RemoteID, m.FolderID, m.Flags)
	}
	if m.Body != "" {
		t.Errorf("the list must not carry bodies: %q", m.Body)
	}
}

func TestRecentOfAnEmptyLimitOrFolder(t *testing.T) {
	c := startServer(t)
	if list, err := c.Recent("INBOX", 0); err != nil || len(list) != 0 {
		t.Errorf("limit 0: %v %v", list, err)
	}
	if _, err := c.Recent("Nope", 5); err == nil {
		t.Error("a missing folder was opened")
	}
	// A server error about one command leaves the session usable.
	if list, err := c.Recent("INBOX", 5); err != nil || len(list) != 1 {
		t.Errorf("after the error: %v %v", list, err)
	}
}

func TestFetchGivesTheBody(t *testing.T) {
	c := startServer(t)
	m, err := c.Fetch("INBOX", "6")
	if err != nil {
		t.Fatal(err)
	}
	if m.Body != "Hi there :)" || m.RemoteID != "6" {
		t.Errorf("message = %+v", m)
	}
	if _, err := c.Fetch("INBOX", "999"); err == nil {
		t.Error("a missing message was fetched")
	}
	if _, err := c.Fetch("INBOX", "abc"); err == nil {
		t.Error("a bad id was accepted")
	}
}

func TestSetReadChangesTheFlag(t *testing.T) {
	c := startServer(t)
	if err := c.SetRead("INBOX", "6", false); err != nil {
		t.Fatal(err)
	}
	m, _ := c.Fetch("INBOX", "6")
	if m.Has(domain.Seen) {
		t.Error("still read")
	}
	if err := c.SetRead("INBOX", "6", true); err != nil {
		t.Fatal(err)
	}
	m, _ = c.Fetch("INBOX", "6")
	if !m.Has(domain.Seen) {
		t.Error("not read")
	}
}

func TestDeleteWithoutATrashRemovesTheMessage(t *testing.T) {
	c := startServer(t)
	if err := c.Delete("INBOX", "6"); err != nil {
		t.Fatal(err)
	}
	if list, _ := c.Recent("INBOX", 10); len(list) != 0 {
		t.Errorf("still %d messages", len(list))
	}
}

func TestWrongPasswordIsUnauthorized(t *testing.T) {
	c := startServer(t)
	c.SignIn = Password("username", "wrong")
	_, err := c.Folders()
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("err = %v", err)
	}
	if err != nil && strings.Contains(err.Error(), "wrong") {
		t.Errorf("the password leaked into the error: %v", err)
	}
}

func TestUnreachableServerAndMissingSignIn(t *testing.T) {
	c := &Client{Addr: "127.0.0.1:1", Security: NoTLS, SignIn: Password("a", "b"), Timeout: time.Second}
	if _, err := c.Folders(); err == nil || !strings.Contains(err.Error(), "cannot reach") {
		t.Errorf("err = %v", err)
	}
	if _, err := (&Client{Addr: "sem-porta"}).Folders(); err == nil {
		t.Error("bad address accepted")
	}
	srv := startServer(t)
	srv.SignIn = nil
	if _, err := srv.Folders(); err == nil {
		t.Error("no sign-in accepted")
	}
}

// fakeSender records what it was asked to send.
type fakeSender struct {
	mu   sync.Mutex
	from string
	to   []string
	raw  string
	err  error
}

func (f *fakeSender) Send(from string, to []string, raw []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.from, f.to, f.raw = from, to, string(raw)
	return f.err
}

func draft() domain.Draft {
	return domain.Draft{
		From:    domain.Address{Email: "ana@example.org", Name: "Ana"},
		To:      []domain.Address{{Email: "bia@example.org"}},
		Bcc:     []domain.Address{{Email: "cris@example.org"}},
		Subject: "Oi",
		Body:    "Olá!",
	}
}

func TestSendUsesTheEnvelopeWithBcc(t *testing.T) {
	c := startServer(t)
	snd := &fakeSender{}
	c.Sender = snd
	if err := c.Send(draft()); err != nil {
		t.Fatal(err)
	}
	if snd.from != "ana@example.org" || len(snd.to) != 2 {
		t.Errorf("envelope = %q %q", snd.from, snd.to)
	}
	if strings.Contains(snd.raw, "cris@example.org") {
		t.Error("the Bcc recipient is in the message header")
	}
	if !strings.Contains(snd.raw, "Subject: Oi") {
		t.Errorf("raw = %q", snd.raw)
	}
}

func TestSendKeepsACopyInSentWhenAsked(t *testing.T) {
	c := startServer(t)
	srvC := c
	srvC.Sender = &fakeSender{}
	srvC.SaveSent = true
	// The in-memory server has no Sent folder: the copy is skipped, the send is not.
	if err := srvC.Send(draft()); err != nil {
		t.Fatalf("a missing Sent folder broke the send: %v", err)
	}
}

func TestSendErrors(t *testing.T) {
	c := startServer(t)
	if err := c.Send(draft()); err == nil {
		t.Error("sent without an SMTP server")
	}
	c.Sender = &fakeSender{err: errors.New("refused")}
	if err := c.Send(draft()); err == nil || err.Error() != "refused" {
		t.Errorf("sender error = %v", err)
	}
	c.Sender = &fakeSender{}
	if err := c.Send(domain.Draft{}); err == nil {
		t.Error("an empty draft was sent")
	}
}

func TestRolesComeFromAttributesThenNames(t *testing.T) {
	cases := []struct {
		name  string
		attrs []string
		role  domain.FolderRole
	}{
		{"INBOX", nil, domain.Inbox},
		{"inbox", nil, domain.Inbox},
		{"[Gmail]/Sent Mail", []string{`\Sent`}, domain.Sent},
		{"Qualquer", []string{`\Trash`}, domain.Trash},
		{"Enviados", nil, domain.Sent},
		{"INBOX/Rascunhos", nil, domain.Drafts},
		{"Spam", nil, domain.Junk},
		{"Projetos", nil, domain.Other},
	}
	for _, tc := range cases {
		info := &imap.MailboxInfo{Name: tc.name, Attributes: tc.attrs, Delimiter: "/"}
		if got := roleOf(info); got != tc.role {
			t.Errorf("%s (%v) = %s, want %s", tc.name, tc.attrs, got, tc.role)
		}
	}
	if _, ok := folderOf(&imap.MailboxInfo{Name: "Pai", Attributes: []string{`\Noselect`}}); ok {
		t.Error("a \\Noselect mailbox was listed")
	}
	if f, _ := folderOf(&imap.MailboxInfo{Name: "INBOX/Trabalho", Delimiter: "/"}); f.Name != "Trabalho" || f.ID != "INBOX/Trabalho" {
		t.Errorf("folder = %+v", f)
	}
	if rank(domain.Inbox) >= rank(domain.Trash) || rank(domain.Other) <= rank(domain.Trash) {
		t.Error("the folder order is wrong")
	}
}

func TestFlagsOf(t *testing.T) {
	f := flagsOf([]string{`\Seen`, `\Flagged`, `\Answered`, `\Draft`, `\Custom`})
	if !(f&domain.Seen != 0 && f&domain.Flagged != 0 && f&domain.Answered != 0 && f&domain.DraftFlag != 0) {
		t.Errorf("flags = %b", f)
	}
}

func TestSetFlaggedChangesTheFlag(t *testing.T) {
	c := startServer(t)
	if err := c.SetFlagged("INBOX", "6", true); err != nil {
		t.Fatal(err)
	}
	m, _ := c.Fetch("INBOX", "6")
	if !m.Has(domain.Flagged) {
		t.Error("not flagged")
	}
	if err := c.SetFlagged("INBOX", "6", false); err != nil {
		t.Fatal(err)
	}
	if m, _ = c.Fetch("INBOX", "6"); m.Has(domain.Flagged) {
		t.Error("still flagged")
	}
}
