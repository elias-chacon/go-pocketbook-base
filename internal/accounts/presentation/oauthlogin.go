package presentation

import (
	"errors"

	"go-pocketbook-base/internal/accounts/application"
	"go-pocketbook-base/internal/accounts/domain"
	"go-pocketbook-base/internal/localization/i18n"
	"go-pocketbook-base/internal/toolkit/widget"
)

// The ids of the providers of the OAuthLogin, and the account presets they sign in to.
const (
	ProviderMicrosoft = "microsoft"
	ProviderGoogle    = "google"
	presetMicrosoft   = "outlook"
	presetGoogle      = "google"
)

// OAuthSources are the sign-ins an app has: Microsoft by device code, Google by authorization code.
// A nil one is a provider whose client id is not set (oauth.json).
type OAuthSources struct {
	Microsoft domain.Authorizer
	Google    domain.CodeAuthorizer
}

// PresetOf is the account preset that a provider of the OAuthLogin signs in to.
func PresetOf(providerID string) string {
	if providerID == ProviderGoogle {
		return presetGoogle
	}
	return presetMicrosoft
}

// OAuthTexts are the words of the toolkit's OAuthLogin, in the language of tr.
func OAuthTexts(tr *i18n.Bundle) widget.OAuthTexts {
	return widget.OAuthTexts{
		Intro: tr.T("connect.choose"), Cancel: tr.T(keyCancel), Check: tr.T("signin.oauth.check"),
		Finish: tr.T("connect.google.finish"), Checking: tr.T("signin.oauth.checking"),
		Pending: tr.T("signin.oauth.pending"), Done: tr.T("signin.oauth.done"),
		Steps: tr.T("signin.oauth.steps"), Scan: tr.T("connect.google.steps"),
		Pasted: tr.T("connect.google.code"), Failed: tr.T(keyFailed, "%s"),
	}
}

// OAuthProviders adapts the sign-ins to the toolkit's OAuthLogin: one provider per id given
// ("microsoft", "google"; both when none), in that order. done receives the tokens once the person
// approved: what it returns (the account could not be added...) is shown by the widget. A provider
// whose source is nil is listed disabled.
func OAuthProviders(tr *i18n.Bundle, src OAuthSources, done func(providerID string, secret domain.Secret) error, ids ...string) []widget.OAuthProvider {
	if len(ids) == 0 {
		ids = []string{ProviderMicrosoft, ProviderGoogle}
	}
	var out []widget.OAuthProvider
	for _, id := range ids {
		switch id {
		case ProviderMicrosoft:
			out = append(out, microsoftProvider(tr, src.Microsoft, done))
		case ProviderGoogle:
			out = append(out, googleProvider(tr, src.Google, done))
		}
	}
	return out
}

// microsoftProvider signs in by device code: the person types the code at an address.
func microsoftProvider(tr *i18n.Bundle, auth domain.Authorizer, done func(string, domain.Secret) error) widget.OAuthProvider {
	p := widget.OAuthProvider{ID: ProviderMicrosoft, Label: tr.T("connect.microsoft")}
	if auth == nil {
		return p
	}
	var ch domain.Challenge
	p.Start = func() (widget.OAuthChallenge, error) {
		var err error
		if ch, err = auth.Start(); err != nil {
			return widget.OAuthChallenge{}, err
		}
		return widget.OAuthChallenge{URL: ch.VerificationURI, Code: ch.UserCode}, nil
	}
	p.Complete = func(widget.OAuthChallenge, string) (bool, error) {
		secret, pending, err := application.CheckOAuth(auth, ch)
		if pending || err != nil {
			return pending, explainOAuth(tr, err)
		}
		return false, done(ProviderMicrosoft, secret)
	}
	return p
}

// googleProvider signs in by authorization code: the address is read as a QR code and the code
// the page ends in is typed back.
func googleProvider(tr *i18n.Bundle, auth domain.CodeAuthorizer, done func(string, domain.Secret) error) widget.OAuthProvider {
	p := widget.OAuthProvider{ID: ProviderGoogle, Label: tr.T("connect.google")}
	if auth == nil {
		return p
	}
	var ch domain.Challenge
	p.Start = func() (widget.OAuthChallenge, error) {
		var err error
		if ch, err = auth.Start(); err != nil {
			return widget.OAuthChallenge{}, err
		}
		return widget.OAuthChallenge{URL: ch.VerificationURI}, nil
	}
	p.Complete = func(_ widget.OAuthChallenge, typed string) (bool, error) {
		secret, err := auth.Exchange(ch, typed)
		if err != nil {
			return false, explainOAuth(tr, err)
		}
		return false, done(ProviderGoogle, secret)
	}
	return p
}

// explainOAuth puts a failure of the sign-in in the words of the person.
func explainOAuth(tr *i18n.Bundle, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, domain.ErrDenied):
		return errors.New(tr.T("signin.oauth.denied"))
	case errors.Is(err, domain.ErrExpired):
		return errors.New(tr.T("signin.oauth.expired"))
	}
	return err
}

// ExplainRegistration puts a failure to add the account in the words of the person (nil stays nil).
func ExplainRegistration(tr *i18n.Bundle, err error) error { return explainRegistration(tr, err) }

// explainRegistration puts a failure to add the account in the words of the person.
func explainRegistration(tr *i18n.Bundle, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, application.ErrInvalidEmail):
		return errors.New(tr.T("signin.invalid"))
	case errors.Is(err, domain.ErrExists):
		return errors.New(tr.T("signin.exists"))
	case errors.Is(err, application.ErrNotOAuth):
		return errors.New(tr.T("signin.oauth.notoauth"))
	}
	return err
}
