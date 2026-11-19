package main

import "net/smtp"

// smtpAuth signs in with the account's login and password (PLAIN over the protected
// connection: net/smtp refuses to send it over an unprotected one).
func smtpAuth(login, password, host string) smtp.Auth {
	return smtp.PlainAuth("", login, password, host)
}
