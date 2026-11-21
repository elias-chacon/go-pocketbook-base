// Package presentation draws the screens of the Accounts context that every app of
// the productivity suite shares: adding an account. It holds no rules: the Register
// use cases decide what is valid.
//
// The first screen asks for the provider (Outlook.com / Microsoft 365, Gmail, iCloud
// or another server). Outlook signs in by code on another device (OAuth2), the others
// with the address and a password, with the servers typed only for an unknown one.
package presentation

import (
	"embed"
	"errors"
	"strings"

	"go-pocketbook-base/internal/accounts/application"
	"go-pocketbook-base/internal/accounts/domain"
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/localization/i18n"
	"go-pocketbook-base/internal/toolkit/widget"
)

//go:embed locales/*.json
var locales embed.FS

// NewTranslator loads the texts of these screens (pt-BR first, English fallback) in lang.
func NewTranslator(lang string) *i18n.Bundle {
	tr := i18n.New()
	if err := tr.AddFS(locales, "locales"); err != nil {
		panic(err) // embedded files: an error is a programming mistake
	}
	tr.SetLang(lang)
	return tr
}

// Registrar is what the screen needs of the Accounts context.
type Registrar interface {
	Register(application.Registration) (domain.Account, error)
	RegisterOAuth(application.Registration, domain.Secret) (domain.Account, error)
}

// SignInConfig is everything the sign-in screen needs.
type SignInConfig struct {
	Width, Height int
	// Doc is the document to draw on (a new one when nil): an app has one document for
	// its whole life, so the screens it switches between share it.
	Doc *widget.Document
	// Chrome is the standard top bar (the app name with About and Exit), above every screen.
	Chrome   widget.AppChrome
	Keyboard widget.Keyboard
	Tr       *i18n.Bundle
	Accounts Registrar
	// Authorizer signs in to Outlook.com by code (nil: it is not configured, and the
	// screen says how to).
	Authorizer domain.Authorizer
	// OAuthSetup tells where the client id of the app registration goes, shown when
	// Authorizer is nil.
	OAuthSetup string
	// OnDone is called with the account that was added.
	OnDone func(domain.Account)
	// OnCancel is called when the person gives up (nil: no Cancel button).
	OnCancel func()
}

// Texts and ids used in several screens.
const (
	keyOutlook = "signin.outlook"
	keyFailed  = "signin.failed"
	idBack     = "signin-back"
)

// The providers the first screen offers.
const (
	providerOutlook = "outlook"
	providerGmail   = "gmail"
	providerICloud  = "icloud"
	providerOther   = "other"
)

// SignIn is the "add an account" screen.
type SignIn struct {
	Doc *widget.Document

	cfg                   SignInConfig
	provider              string
	name, email, password *widget.Input
	imap, smtp            *widget.Input
	hint, note            *widget.Label
}

// NewSignIn builds the screen on the choice of provider.
func NewSignIn(cfg SignInConfig) *SignIn {
	s := &SignIn{cfg: cfg}
	s.Doc = cfg.Doc
	if s.Doc == nil {
		s.Doc = widget.NewDocument(cfg.Width, cfg.Height)
		s.Doc.Keyboard = cfg.Keyboard
	}
	s.showChooser()
	return s
}

func (s *SignIn) t(key string, args ...interface{}) string { return s.cfg.Tr.T(key, args...) }

// Provider is the provider chosen ("" on the first screen).
func (s *SignIn) Provider() string { return s.provider }

func (s *SignIn) button(id, label string, fn func()) *widget.Button {
	b := widget.NewButton(label)
	b.ID = id
	b.OnClick = func(*widget.Event) { fn() }
	return b
}

func (s *SignIn) heading(key string) widget.Widget {
	title := widget.NewLabel(s.t(key))
	title.Style.FontSize, title.Style.Bold = widget.FontTitle, true
	title.Style.Padding = geom.XY(28, 28)
	return title
}

func (s *SignIn) show(children ...widget.Widget) {
	form := widget.NewColumn(16, children...)
	form.Style.Padding = geom.XY(28, 28)
	form.Style.FlexGrow = 1
	root := widget.NewColumn(0, form)
	root.Style.Background = widget.Paper
	s.Doc.SetRoot(s.cfg.Chrome.Wrap(s.Doc, root))
	s.Doc.ResetFrame()
	s.Doc.Invalidate()
}

func (s *SignIn) input(id, placeholder string, kind widget.InputType) *widget.Input {
	in := widget.NewInput(placeholder)
	in.ID, in.Type = id, kind
	return in
}

// ----- the choice of provider -----

func (s *SignIn) showChooser() {
	s.provider = ""
	intro := widget.NewLabel(s.t("signin.choose"))
	intro.Style.Color = widget.Muted
	items := []widget.Widget{
		s.heading("signin.title"), intro,
		s.button("signin-outlook", s.t(keyOutlook), func() { s.choose(providerOutlook) }),
		s.button("signin-gmail", s.t("signin.gmail"), func() { s.choose(providerGmail) }),
		s.button("signin-icloud", s.t("signin.icloud"), func() { s.choose(providerICloud) }),
		s.button("signin-other", s.t("signin.other"), func() { s.choose(providerOther) }),
	}
	if s.cfg.OnCancel != nil {
		items = append(items, s.button("signin-cancel", s.t("signin.cancel"), s.cfg.OnCancel))
	}
	s.show(items...)
}

// choose opens the form of a provider.
func (s *SignIn) choose(provider string) {
	s.provider = provider
	if provider == providerOutlook {
		s.showOutlook()
		return
	}
	s.showPassword()
}

// ----- Outlook.com and Microsoft 365: sign in by code -----

func (s *SignIn) showOutlook() {
	back := s.button(idBack, s.t("signin.back"), s.showChooser)
	if s.cfg.Authorizer == nil {
		hint := widget.NewLabel(s.t("signin.oauth.setup", s.cfg.OAuthSetup))
		hint.Style.Color = widget.Muted
		s.show(s.heading(keyOutlook), hint, back)
		return
	}
	s.email = s.input("signin-email", s.t("signin.email"), widget.TypeText)
	s.name = s.input("signin-name", s.t("signin.name"), widget.TypeText)
	hint := widget.NewLabel(s.t("signin.oauth.hint"))
	hint.Style.FontSize, hint.Style.Color = widget.FontSmall, widget.Muted
	// The sign-in itself is the toolkit's OAuthLogin: the code to type on another device.
	providers := OAuthProviders(s.cfg.Tr, OAuthSources{Microsoft: s.cfg.Authorizer}, s.registerOAuth, ProviderMicrosoft)
	if start := providers[0].Start; start != nil {
		providers[0].Start = func() (widget.OAuthChallenge, error) {
			if !strings.Contains(s.email.Value, "@") {
				return widget.OAuthChallenge{}, errors.New(s.t("signin.invalid"))
			}
			return start()
		}
	}
	login := widget.NewOAuthLogin(providers...)
	login.Texts = OAuthTexts(s.cfg.Tr)
	login.Reset() // the words were set after the first build
	s.show(s.heading(keyOutlook), s.email, s.name, hint, login.Root, back)
}

// registerOAuth adds the account with the tokens the sign-in gave.
func (s *SignIn) registerOAuth(_ string, secret domain.Secret) error {
	acc, err := s.cfg.Accounts.RegisterOAuth(application.Registration{Name: s.name.Value, Email: s.email.Value}, secret)
	if err != nil {
		return explainRegistration(s.cfg.Tr, err)
	}
	if s.cfg.OnDone != nil {
		s.cfg.OnDone(acc)
	}
	return nil
}

// ----- Gmail, iCloud and others: address and password -----

func (s *SignIn) showPassword() {
	s.email = s.input("signin-email", s.t("signin.email"), widget.TypeText)
	s.name = s.input("signin-name", s.t("signin.name"), widget.TypeText)
	s.password = s.input("signin-password", s.t("signin.password"), widget.TypePassword)
	s.imap = s.input("signin-imap", s.t("signin.imap"), widget.TypeText)
	s.smtp = s.input("signin-smtp", s.t("signin.smtp"), widget.TypeText)
	s.imap.Hidden, s.smtp.Hidden = s.provider != providerOther, s.provider != providerOther
	s.email.OnChange = func(*widget.Event) { s.update() }

	s.hint = widget.NewLabel("")
	s.hint.Style.FontSize, s.hint.Style.Color = widget.FontSmall, widget.Muted
	s.note = widget.NewLabel("")
	s.note.Style.Bold = true
	add := s.button("signin-add", s.t("signin.add"), s.Submit)
	back := s.button(idBack, s.t("signin.back"), s.showChooser)
	s.update()
	s.show(s.heading("signin.title"), s.email, s.name, s.password, s.hint, s.imap, s.smtp, s.note, widget.NewRow(12, add, back))
}

// update shows what the typed address needs: the servers, or a hint about the password.
func (s *SignIn) update() {
	prov := application.ProviderOf(s.email.Value)
	custom := !prov.Known && s.email.Value != ""
	s.imap.Hidden, s.smtp.Hidden = !custom && s.provider != providerOther, !custom && s.provider != providerOther
	switch {
	case prov.OAuth():
		s.hint.Text = s.t("signin.hint.oauth", prov.Preset.Label)
	case custom || s.provider == providerOther:
		s.hint.Text = s.t("signin.hint.custom")
	default:
		s.hint.Text = s.t("signin.hint.app")
	}
	s.Doc.Invalidate()
}

// Submit adds the account with what was typed.
func (s *SignIn) Submit() {
	if s.password == nil {
		return
	}
	acc, err := s.cfg.Accounts.Register(application.Registration{
		Name: s.name.Value, Email: s.email.Value, Password: s.password.Value,
		IMAPHost: s.imap.Value, SMTPHost: s.smtp.Value,
	})
	if err != nil {
		s.note.Text = s.explain(err)
		s.Doc.Invalidate()
		return
	}
	s.password.Value = "" // the password does not stay in the widget
	if s.cfg.OnDone != nil {
		s.cfg.OnDone(acc)
	}
}

// explain turns an error into the sentence the person reads. The password is never
// part of one.
func (s *SignIn) explain(err error) string {
	switch {
	case errors.Is(err, application.ErrInvalidEmail):
		return s.t("signin.invalid")
	case errors.Is(err, application.ErrNeedPassword):
		return s.t("signin.nopassword")
	case errors.Is(err, application.ErrNeedServers):
		return s.t("signin.noservers")
	case errors.Is(err, application.ErrNeedOAuth):
		return s.t("signin.hint.oauth", "Outlook.com")
	case errors.Is(err, application.ErrNotOAuth):
		return s.t("signin.oauth.notoauth")
	case errors.Is(err, domain.ErrExists):
		return s.t("signin.exists")
	}
	return s.t(keyFailed, err.Error())
}
