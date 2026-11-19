// Package smtpsend sends a composed message over SMTP (standard library net/smtp).
// It knows nothing of drafts: the caller hands it the sender, the recipients and the
// bytes of the message, so the same code serves any account.
package smtpsend

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"time"
)

// Security is how the connection is protected.
type Security int

// The ways to protect the connection. NoTLS is for tests and a trusted local relay:
// net/smtp refuses to send a password over it anyway, except to localhost.
const (
	StartTLS Security = iota // port 587: starts plain, upgrades before signing in
	TLS                      // port 465: encrypted from the first byte
	NoTLS
)

// DefaultTimeout bounds the whole conversation with the server.
const DefaultTimeout = 60 * time.Second

// Sender is one SMTP server and the way to sign in to it.
type Sender struct {
	Addr      string // host:port
	Security  Security
	Auth      smtp.Auth   // nil: no sign-in
	TLSConfig *tls.Config // nil: defaults, verifying the host of Addr
	Timeout   time.Duration
}

// Send sends the message from one address to the recipients (To, Cc and Bcc
// together: the envelope, not the header).
func (s Sender) Send(from string, to []string, raw []byte) error {
	if from == "" || len(to) == 0 {
		return errors.New("smtp: a sender and a recipient are needed")
	}
	c, err := s.connect()
	if err != nil {
		return err
	}
	defer c.Close()

	if s.Auth != nil {
		if err := c.Auth(s.Auth); err != nil {
			return fmt.Errorf("smtp: sign-in refused: %w", err)
		}
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("smtp: sender refused: %w", err)
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			return fmt.Errorf("smtp: recipient %s refused: %w", rcpt, err)
		}
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("smtp: %w", err)
	}
	if _, err := w.Write(raw); err != nil {
		return fmt.Errorf("smtp: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp: message refused: %w", err)
	}
	return c.Quit()
}

// connect dials, upgrades the connection when asked and says hello.
func (s Sender) connect() (*smtp.Client, error) {
	host, _, err := net.SplitHostPort(s.Addr)
	if err != nil {
		return nil, fmt.Errorf("smtp: bad address %q: %w", s.Addr, err)
	}
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	cfg := s.TLSConfig
	if cfg == nil {
		cfg = &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	}
	dialer := &net.Dialer{Timeout: timeout}
	var conn net.Conn
	if s.Security == TLS {
		conn, err = tls.DialWithDialer(dialer, "tcp", s.Addr, cfg)
	} else {
		conn, err = dialer.Dial("tcp", s.Addr)
	}
	if err != nil {
		return nil, fmt.Errorf("smtp: cannot reach %s: %w", s.Addr, err)
	}
	_ = conn.SetDeadline(time.Now().Add(timeout))
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("smtp: %w", err)
	}
	if s.Security == StartTLS {
		if err := c.StartTLS(cfg); err != nil {
			c.Close()
			return nil, fmt.Errorf("smtp: cannot start TLS: %w", err)
		}
	}
	return c, nil
}
