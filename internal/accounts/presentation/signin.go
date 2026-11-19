// Package presentation draws the screens of the Accounts context that every app of
// the productivity suite shares: today, adding an account. It holds no rules: the
// Register use case decides what is valid.
package presentation

import (
	"embed"
	"errors"

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
}

// SignInConfig is everything the sign-in screen needs.
type SignInConfig struct {
	Width, Height int
	// Doc is the document to draw on (a new one when nil): an app has one document for
	// its whole life, so the screens it switches between share it.
	Doc      *widget.Document
	Keyboard widget.Keyboard
	Tr       *i18n.Bundle
	Accounts Registrar
	// OnDone is called with the account that was added.
	OnDone func(domain.Account)
	// OnCancel is called when the person gives up (nil: no Cancel button).
	OnCancel func()
}

// SignIn is the "add an account" screen: the address, the password and, for a
// provider that is not known, its servers.
type SignIn struct {
	Doc *widget.Document

	cfg                   SignInConfig
	name, email, password *widget.Input
	imap, smtp            *widget.Input
	hint, note            *widget.Label
	add                   *widget.Button
}

// NewSignIn builds the screen.
func NewSignIn(cfg SignInConfig) *SignIn {
	s := &SignIn{cfg: cfg}
	s.Doc = cfg.Doc
	if s.Doc == nil {
		s.Doc = widget.NewDocument(cfg.Width, cfg.Height)
		s.Doc.Keyboard = cfg.Keyboard
	}
	s.build()
	return s
}

func (s *SignIn) t(key string, args ...interface{}) string { return s.cfg.Tr.T(key, args...) }

func (s *SignIn) input(id, placeholder string, kind widget.InputType) *widget.Input {
	in := widget.NewInput(placeholder)
	in.ID, in.Type = id, kind
	in.Style.FlexGrow = 1
	in.OnChange = func(*widget.Event) { s.update() }
	return in
}

func (s *SignIn) build() {
	title := widget.NewLabel(s.t("signin.title"))
	title.Style.FontSize, title.Style.Bold = widget.FontTitle, true
	title.Style.Padding = geom.XY(28, 28)

	s.email = s.input("signin-email", s.t("signin.email"), widget.TypeText)
	s.name = s.input("signin-name", s.t("signin.name"), widget.TypeText)
	s.password = s.input("signin-password", s.t("signin.password"), widget.TypePassword)
	s.imap = s.input("signin-imap", s.t("signin.imap"), widget.TypeText)
	s.smtp = s.input("signin-smtp", s.t("signin.smtp"), widget.TypeText)

	s.hint = widget.NewLabel("")
	s.hint.Style.FontSize, s.hint.Style.Color = widget.FontSmall, widget.Muted
	s.note = widget.NewLabel("")
	s.note.Style.Bold = true

	s.add = widget.NewButton(s.t("signin.add"))
	s.add.ID = "signin-add"
	s.add.OnClick = func(*widget.Event) { s.Submit() }
	buttons := []widget.Widget{s.add}
	if s.cfg.OnCancel != nil {
		cancel := widget.NewButton(s.t("signin.cancel"))
		cancel.ID = "signin-cancel"
		cancel.OnClick = func(*widget.Event) { s.cfg.OnCancel() }
		buttons = append(buttons, cancel)
	}

	form := widget.NewColumn(16, s.email, s.name, s.password, s.hint, s.imap, s.smtp, s.note, widget.NewRow(12, buttons...))
	form.Style.Padding = geom.XY(28, 28)
	form.Style.FlexGrow = 1
	root := widget.NewColumn(0, title, form)
	root.Style.Background = widget.Paper
	s.Doc.SetRoot(root)
	s.update()
}

// update shows what the typed address needs: the servers, or a hint about the password.
func (s *SignIn) update() {
	prov := application.ProviderOf(s.email.Value)
	custom := !prov.Known
	s.imap.Hidden, s.smtp.Hidden = !custom || s.email.Value == "", !custom || s.email.Value == ""
	switch {
	case s.email.Value == "":
		s.hint.Text = ""
	case prov.OAuth():
		s.hint.Text = s.t("signin.hint.oauth", prov.Preset.Label)
	case custom:
		s.hint.Text = s.t("signin.hint.custom")
	default:
		s.hint.Text = s.t("signin.provider", prov.Preset.Label) + "\n" + s.t("signin.hint.app")
	}
	s.add.Disabled = prov.OAuth()
	s.Doc.Invalidate()
}

// Submit adds the account with what was typed.
func (s *SignIn) Submit() {
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
		return s.t("signin.hint.oauth", "")
	case errors.Is(err, domain.ErrExists):
		return s.t("signin.exists")
	}
	return s.t("signin.failed", err.Error())
}
