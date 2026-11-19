// Package imapmail is the mail Backend for any IMAP server (and the SMTP server that
// goes with it): Gmail, iCloud, Yahoo, Fastmail, a company server. It keeps one
// connection while the app talks to the server and drops it on the first error, so
// the next call starts clean; nothing is kept between runs.
package imapmail

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-sasl"

	"go-pocketbook-base/internal/mail/domain"
	"go-pocketbook-base/internal/mail/infrastructure/mime"
)

// Security is how the connection is protected.
type Security int

// The ways to protect the connection. NoTLS is for tests and a trusted local relay.
const (
	TLS      Security = iota // port 993: encrypted from the first byte
	StartTLS                 // port 143: starts plain, upgrades before signing in
	NoTLS
)

// Limits: the reader holds a message in memory while it reads it.
const (
	DefaultTimeout = 60 * time.Second
	maxMessage     = 10 << 20
)

// Sender sends a composed message: the SMTP server of the account.
type Sender interface {
	Send(from string, to []string, raw []byte) error
}

// Client is the IMAP side of an account.
type Client struct {
	Addr      string // host:port of the IMAP server
	Security  Security
	TLSConfig *tls.Config // nil: defaults, verifying the host of Addr
	Timeout   time.Duration

	// SignIn proves who the person is. Use Password or SASL.
	SignIn func(*client.Client) error

	// Sender sends mail; when SaveSent is set the message is also appended to the
	// Sent folder (servers such as Gmail and Outlook do that themselves).
	Sender   Sender
	SaveSent bool

	mu   sync.Mutex
	conn *client.Client
}

// Password signs in with the LOGIN command.
func Password(user, secret string) func(*client.Client) error {
	return func(c *client.Client) error { return c.Login(user, secret) }
}

// SASL signs in with a SASL mechanism (XOAUTH2, see package xoauth2).
func SASL(mech func() sasl.Client) func(*client.Client) error {
	return func(c *client.Client) error { return c.Authenticate(mech()) }
}

// Close ends the connection, if there is one.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.drop(true)
}

func (c *Client) drop(polite bool) {
	if c.conn == nil {
		return
	}
	if polite {
		_ = c.conn.Logout()
	} else {
		_ = c.conn.Terminate()
	}
	c.conn = nil
}

// run connects if needed and runs fn on the connection. An error drops the
// connection, except when it is the answer of the server to one command, which
// leaves the session healthy.
func (c *Client) run(fn func(*client.Client) error) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		conn, err := c.dial()
		if err != nil {
			return err
		}
		c.conn = conn
	}
	err := fn(c.conn)
	var status *imap.ErrStatusResp
	if err != nil && !errors.As(err, &status) && !errors.Is(err, errNoMessage) {
		c.drop(false)
	}
	return err
}

func (c *Client) dial() (*client.Client, error) {
	host, _, err := net.SplitHostPort(c.Addr)
	if err != nil {
		return nil, fmt.Errorf("imap: bad address %q: %w", c.Addr, err)
	}
	cfg := c.TLSConfig
	if cfg == nil {
		cfg = &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	}
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	dialer := &net.Dialer{Timeout: timeout}
	var conn *client.Client
	if c.Security == TLS {
		conn, err = client.DialWithDialerTLS(dialer, c.Addr, cfg)
	} else {
		conn, err = client.DialWithDialer(dialer, c.Addr)
	}
	if err != nil {
		return nil, fmt.Errorf("imap: cannot reach %s: %w", c.Addr, err)
	}
	conn.Timeout = timeout
	if c.Security == StartTLS {
		if err := conn.StartTLS(cfg); err != nil {
			_ = conn.Terminate()
			return nil, fmt.Errorf("imap: cannot start TLS: %w", err)
		}
	}
	if c.SignIn == nil {
		_ = conn.Terminate()
		return nil, errors.New("imap: no way to sign in")
	}
	if err := c.SignIn(conn); err != nil {
		_ = conn.Terminate()
		var status *imap.ErrStatusResp
		if errors.As(err, &status) {
			return nil, fmt.Errorf("%w (%s)", domain.ErrUnauthorized, firstLine(status.Resp.Info))
		}
		return nil, fmt.Errorf("%w (%s)", domain.ErrUnauthorized, firstLine(err.Error()))
	}
	return conn, nil
}

func firstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		s = s[:i]
	}
	if len(s) > 120 {
		s = s[:120]
	}
	return s
}

// Folders lists the mailboxes the person can open.
func (c *Client) Folders() ([]domain.Folder, error) {
	var out []domain.Folder
	err := c.run(func(conn *client.Client) error {
		ch := make(chan *imap.MailboxInfo, 16)
		done := make(chan error, 1)
		go func() { done <- conn.List("", "*", ch) }()
		for info := range ch {
			if f, ok := folderOf(info); ok {
				out = append(out, f)
			}
		}
		return <-done
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool { return rank(out[i].Role) < rank(out[j].Role) })
	return out, nil
}

// Recent lists the newest messages of a folder, newest first, without bodies.
func (c *Client) Recent(folderID string, limit int) ([]domain.Message, error) {
	var out []domain.Message
	err := c.run(func(conn *client.Client) error {
		status, err := conn.Select(folderID, true)
		if err != nil {
			return err
		}
		if status.Messages == 0 || limit <= 0 {
			return nil
		}
		from := uint32(1)
		if status.Messages > uint32(limit) {
			from = status.Messages - uint32(limit) + 1
		}
		set := new(imap.SeqSet)
		set.AddRange(from, status.Messages)
		raw, err := fetch(conn, false, set, []imap.FetchItem{imap.FetchUid, imap.FetchFlags, headerSection.FetchItem()})
		if err != nil {
			return err
		}
		for _, m := range raw {
			out = append(out, messageOf(m, folderID, false))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date.After(out[j].Date) })
	return out, nil
}

var (
	headerSection = &imap.BodySectionName{BodyPartName: imap.BodyPartName{Specifier: imap.HeaderSpecifier}, Peek: true}
	fullSection   = &imap.BodySectionName{Peek: true}
	errNoMessage  = errors.New("imap: the message is not on the server any more")
)

// fetch runs a FETCH (by UID or by sequence number) and collects the answers.
func fetch(conn *client.Client, byUID bool, set *imap.SeqSet, items []imap.FetchItem) ([]*imap.Message, error) {
	ch := make(chan *imap.Message, 16)
	done := make(chan error, 1)
	go func() {
		if byUID {
			done <- conn.UidFetch(set, items, ch)
		} else {
			done <- conn.Fetch(set, items, ch)
		}
	}()
	var out []*imap.Message
	for m := range ch {
		out = append(out, m)
	}
	return out, <-done
}

// messageOf turns what the server sent into a Message. withBody keeps the text.
func messageOf(m *imap.Message, folderID string, withBody bool) domain.Message {
	section := headerSection
	if withBody {
		section = fullSection
	}
	var parsed domain.Message
	if lit := m.GetBody(section); lit != nil {
		if p, err := mime.Parse(io.LimitReader(lit, maxMessage)); err == nil {
			parsed = p
		}
	}
	parsed.FolderID = folderID
	parsed.RemoteID = strconv.FormatUint(uint64(m.Uid), 10)
	parsed.Flags = flagsOf(m.Flags)
	return parsed
}

func flagsOf(flags []string) domain.Flag {
	var f domain.Flag
	for _, s := range flags {
		switch s {
		case imap.SeenFlag:
			f |= domain.Seen
		case imap.FlaggedFlag:
			f |= domain.Flagged
		case imap.AnsweredFlag:
			f |= domain.Answered
		case imap.DraftFlag:
			f |= domain.DraftFlag
		}
	}
	return f
}

func uidSet(remoteID string) (*imap.SeqSet, error) {
	uid, err := strconv.ParseUint(remoteID, 10, 32)
	if err != nil || uid == 0 {
		return nil, fmt.Errorf("imap: bad message id %q", remoteID)
	}
	set := new(imap.SeqSet)
	set.AddNum(uint32(uid))
	return set, nil
}

// Fetch returns one message with its text.
func (c *Client) Fetch(folderID, remoteID string) (domain.Message, error) {
	set, err := uidSet(remoteID)
	if err != nil {
		return domain.Message{}, err
	}
	var out domain.Message
	err = c.run(func(conn *client.Client) error {
		if _, err := conn.Select(folderID, true); err != nil {
			return err
		}
		raw, err := fetch(conn, true, set, []imap.FetchItem{imap.FetchUid, imap.FetchFlags, imap.FetchRFC822Size, fullSection.FetchItem()})
		if err != nil {
			return err
		}
		if len(raw) == 0 {
			return errNoMessage
		}
		if raw[0].Size > maxMessage {
			return fmt.Errorf("imap: the message has %d MB: too big to open on the reader", raw[0].Size>>20)
		}
		out = messageOf(raw[0], folderID, true)
		return nil
	})
	return out, err
}

// Send sends a draft through the account's SMTP server.
func (c *Client) Send(d domain.Draft) error {
	if c.Sender == nil {
		return errors.New("imap: this account has no SMTP server")
	}
	if err := d.Validate(); err != nil {
		return err
	}
	now := time.Now()
	raw, err := mime.Compose(d, now, mime.NewMessageID(d.From))
	if err != nil {
		return err
	}
	rcpts := d.Recipients()
	to := make([]string, 0, len(rcpts))
	for _, r := range rcpts {
		to = append(to, r.Email)
	}
	if err := c.Sender.Send(d.From.Email, to, raw); err != nil {
		return err
	}
	if !c.SaveSent {
		return nil
	}
	// The message is gone: a failed copy must not look like a failed send.
	_ = c.appendTo(domain.Sent, raw, now, []string{imap.SeenFlag})
	return nil
}

// appendTo keeps a copy of a message in the folder with the role.
func (c *Client) appendTo(role domain.FolderRole, raw []byte, when time.Time, flags []string) error {
	name, err := c.folderWithRole(role)
	if err != nil || name == "" {
		return err
	}
	return c.run(func(conn *client.Client) error {
		return conn.Append(name, flags, when, bytes.NewBuffer(raw))
	})
}

func (c *Client) folderWithRole(role domain.FolderRole) (string, error) {
	folders, err := c.Folders()
	if err != nil {
		return "", err
	}
	for _, f := range folders {
		if f.Role == role {
			return f.ID, nil
		}
	}
	return "", nil
}

// SetRead marks a message as read or unread.
func (c *Client) SetRead(folderID, remoteID string, read bool) error {
	var op imap.FlagsOp = imap.RemoveFlags
	if read {
		op = imap.AddFlags
	}
	return c.store(folderID, remoteID, op, imap.SeenFlag)
}

// SetFlagged flags a message for follow-up or removes the flag.
func (c *Client) SetFlagged(folderID, remoteID string, flagged bool) error {
	var op imap.FlagsOp = imap.RemoveFlags
	if flagged {
		op = imap.AddFlags
	}
	return c.store(folderID, remoteID, op, imap.FlaggedFlag)
}

func (c *Client) store(folderID, remoteID string, op imap.FlagsOp, flag string) error {
	set, err := uidSet(remoteID)
	if err != nil {
		return err
	}
	return c.run(func(conn *client.Client) error {
		if _, err := conn.Select(folderID, false); err != nil {
			return err
		}
		return conn.UidStore(set, imap.FormatFlagsOp(op, true), []interface{}{flag}, nil)
	})
}

// Delete moves a message to the Trash; a message already in the Trash, or an account
// with no Trash, is deleted for good.
func (c *Client) Delete(folderID, remoteID string) error {
	set, err := uidSet(remoteID)
	if err != nil {
		return err
	}
	trash, err := c.folderWithRole(domain.Trash)
	if err != nil {
		return err
	}
	return c.run(func(conn *client.Client) error {
		if _, err := conn.Select(folderID, false); err != nil {
			return err
		}
		if trash != "" && trash != folderID {
			if err := conn.UidCopy(set, trash); err != nil {
				return err
			}
		}
		if err := conn.UidStore(set, imap.FormatFlagsOp(imap.AddFlags, true), []interface{}{imap.DeletedFlag}, nil); err != nil {
			return err
		}
		return conn.Expunge(nil)
	})
}
