package main

import (
	"errors"
	"net"
	"strconv"

	accounts "go-pocketbook-base/internal/accounts/domain"
	"go-pocketbook-base/internal/mail/domain"
	"go-pocketbook-base/internal/mail/infrastructure/graph"
	"go-pocketbook-base/internal/mail/infrastructure/imapmail"
	"go-pocketbook-base/internal/mail/infrastructure/smtpsend"
	"go-pocketbook-base/internal/mail/infrastructure/xoauth2"
)

var errNoMailServer = errors.New("this account has no IMAP server")

// tokenSource gives the current access token of an OAuth2 account, renewed when needed.
type tokenSource func() (string, error)

// backendFor connects an account to its mail server: IMAP and SMTP with an app password,
// or Microsoft Graph (Outlook.com, Microsoft 365) with an access token that tokens renews.
func backendFor(acc accounts.Account, secret accounts.Secret, tokens tokenSource, web graph.Doer) (domain.Backend, error) {
	if acc.Auth == accounts.OAuth2 {
		return oauthBackend(acc, tokens, web)
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

// oauthBackend connects an OAuth2 account: Microsoft Graph, or an IMAP server (Gmail) signed in
// with XOAUTH2 and an SMTP server that takes the same token.
func oauthBackend(acc accounts.Account, tokens tokenSource, web graph.Doer) (domain.Backend, error) {
	if tokens == nil {
		return nil, errNoMailServer
	}
	if _, ok := acc.Endpoint(accounts.Mail, accounts.Graph); ok {
		return graph.Client{HTTP: web, Token: tokens}, nil
	}
	in, ok := acc.Endpoint(accounts.Mail, accounts.IMAP)
	if !ok {
		return nil, errNoMailServer
	}
	auth := xoauth2.Auth{User: acc.Login(), Token: tokens}
	client := &imapmail.Client{
		Addr: address(in), Security: imapSecurity(in.Security),
		SignIn: imapmail.SASL(auth.SASL),
		// Gmail keeps a copy of what is sent by itself.
		SaveSent: false,
	}
	if out, ok := acc.Endpoint(accounts.Mail, accounts.SMTP); ok {
		client.Sender = smtpsend.Sender{Addr: address(out), Security: smtpSecurity(out.Security), Auth: auth.SMTP()}
	}
	return client, nil
}
