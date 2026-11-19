package main

import (
	"errors"
	"net"
	"strconv"

	accounts "go-pocketbook-base/internal/accounts/domain"
	"go-pocketbook-base/internal/mail/domain"
	"go-pocketbook-base/internal/mail/infrastructure/imapmail"
	"go-pocketbook-base/internal/mail/infrastructure/smtpsend"
)

var errNoMailServer = errors.New("this account has no IMAP server")

// backendFor connects an account to its mail server. Only password accounts (an app
// password) are supported for now: OAuth2 comes with the sign-in by device code.
func backendFor(acc accounts.Account, secret accounts.Secret) (domain.Backend, error) {
	if acc.Auth != accounts.Password {
		return nil, errors.New("this account signs in with OAuth2, which is not available yet")
	}
	in, ok := acc.Endpoint(accounts.Mail, accounts.IMAP)
	if !ok {
		return nil, errNoMailServer
	}
	client := &imapmail.Client{
		Addr:     address(in),
		Security: imapSecurity(in.Security),
		SignIn:   imapmail.Password(acc.Login(), secret.Password),
		// Gmail keeps a copy of what is sent by itself; the others need the app to.
		SaveSent: acc.Provider != "gmail",
	}
	if out, ok := acc.Endpoint(accounts.Mail, accounts.SMTP); ok {
		client.Sender = smtpsend.Sender{
			Addr:     address(out),
			Security: smtpSecurity(out.Security),
			Auth:     smtpAuth(acc.Login(), secret.Password, out.Host),
		}
	}
	return client, nil
}

func address(e accounts.Endpoint) string { return net.JoinHostPort(e.Host, strconv.Itoa(e.Port)) }

func imapSecurity(s accounts.Security) imapmail.Security {
	if s == accounts.StartTLS {
		return imapmail.StartTLS
	}
	return imapmail.TLS
}

func smtpSecurity(s accounts.Security) smtpsend.Security {
	if s == accounts.TLS {
		return smtpsend.TLS
	}
	return smtpsend.StartTLS
}
