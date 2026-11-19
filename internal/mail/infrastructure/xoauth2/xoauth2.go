// Package xoauth2 is the XOAUTH2 sign-in that Gmail and Outlook accept on IMAP and
// SMTP: the user and the OAuth2 access token, in one initial response. The token
// comes from a function, so the Accounts context can renew it, and it never leaves
// this package in an error.
package xoauth2

import (
	"errors"
	"net/smtp"

	"github.com/emersion/go-sasl"
)

// Mechanism is the name of the SASL mechanism.
const Mechanism = "XOAUTH2"

// ErrNoToken is returned when the token function gives an empty token.
var ErrNoToken = errors.New("xoauth2: no access token")

// Auth is the credentials of one mailbox.
type Auth struct {
	User  string
	Token func() (string, error)
}

func (a Auth) initial() ([]byte, error) {
	token, err := a.Token()
	if err != nil {
		return nil, err
	}
	if token == "" {
		return nil, ErrNoToken
	}
	return []byte("user=" + a.User + "\x01auth=Bearer " + token + "\x01\x01"), nil
}

// SASL returns the client for IMAP (go-sasl).
func (a Auth) SASL() sasl.Client { return saslClient{a} }

// SMTP returns the client for net/smtp.
func (a Auth) SMTP() smtp.Auth { return smtpAuth{a} }

type saslClient struct{ a Auth }

func (c saslClient) Start() (string, []byte, error) {
	ir, err := c.a.initial()
	return Mechanism, ir, err
}

// Next answers the error the server sends back with an empty line, as the protocol
// asks, so the server can finish with its own failure status.
func (c saslClient) Next([]byte) ([]byte, error) { return []byte{}, nil }

type smtpAuth struct{ a Auth }

func (s smtpAuth) Start(*smtp.ServerInfo) (string, []byte, error) {
	ir, err := s.a.initial()
	return Mechanism, ir, err
}

func (s smtpAuth) Next(_ []byte, more bool) ([]byte, error) {
	if more {
		return []byte{}, nil
	}
	return nil, nil
}
