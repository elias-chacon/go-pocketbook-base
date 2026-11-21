package presentation

import (
	"errors"
	"strings"

	"go-pocketbook-base/internal/accounts/application"
	"go-pocketbook-base/internal/accounts/domain"
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/localization/i18n"
	"go-pocketbook-base/internal/toolkit/widget"
)

// ConnectConfig is everything the "connect an online account" screen needs.
type ConnectConfig struct {
	Width, Height int
	Doc           *widget.Document
	Chrome        widget.AppChrome
	Tr            *i18n.Bundle
	Accounts      Registrar
	// Microsoft signs in by device code (nil: not configured, the screen says how).
	Microsoft domain.Authorizer
	// Google signs in by authorization code, shown as a QR code (nil: not configured).
	Google domain.CodeAuthorizer
	// OAuthSetup tells where the client ids go, shown when a provider is not configured.
	OAuthSetup string
	OnDone     func(domain.Account)
	OnCancel   func()
}

// keyCancel is the translation key of the Cancel button.
const keyCancel = "signin.cancel"

// Connect is the screen that signs in to a Microsoft or Google account (the toolkit's OAuthLogin)
// and adds it to the accounts the Mail, Tasks and Calendar apps share.
type Connect struct {
	Doc *widget.Document

	cfg         ConnectConfig
	email, name *widget.Input
	login       *widget.OAuthLogin
}

// NewConnect builds the screen.
func NewConnect(cfg ConnectConfig) *Connect {
	c := &Connect{cfg: cfg, Doc: cfg.Doc}
	if c.Doc == nil {
		c.Doc = widget.NewDocument(cfg.Width, cfg.Height)
	}
	c.show()
	return c
}

func (c *Connect) t(key string, args ...interface{}) string { return c.cfg.Tr.T(key, args...) }

func (c *Connect) field(id, placeholder string) *widget.Input {
	in := widget.NewInput(placeholder)
	in.ID = id
	return in
}

// show draws the address and name to give the account, and the sign-in buttons.
func (c *Connect) show() {
	c.email = c.field("connect-email", c.t("signin.email"))
	c.name = c.field("connect-name", c.t("signin.name"))
	providers := OAuthProviders(c.cfg.Tr, OAuthSources{Microsoft: c.cfg.Microsoft, Google: c.cfg.Google}, c.register)
	for i := range providers {
		if start := providers[i].Start; start != nil {
			providers[i].Start = func() (widget.OAuthChallenge, error) {
				if !strings.Contains(c.email.Value, "@") {
					return widget.OAuthChallenge{}, errors.New(c.t("signin.invalid"))
				}
				return start()
			}
		}
	}
	c.login = widget.NewOAuthLogin(providers...)
	c.login.Texts = OAuthTexts(c.cfg.Tr)
	c.login.Reset() // the words were set after the first build

	head := widget.NewLabel(c.t("connect.title"))
	head.Style.FontSize, head.Style.Bold = widget.FontTitle, true
	items := []widget.Widget{head, c.email, c.name, c.login.Root}
	if c.cfg.Microsoft == nil || c.cfg.Google == nil {
		hint := widget.NewLabel(c.t("connect.setup", c.cfg.OAuthSetup))
		hint.ID = "connect-setup"
		hint.Style.FontSize, hint.Style.Color = widget.FontSmall, widget.Muted
		items = append(items, hint)
	}
	if c.cfg.OnCancel != nil {
		cancel := widget.NewButton(c.t(keyCancel))
		cancel.ID = "connect-cancel"
		cancel.OnClick = func(*widget.Event) { c.cfg.OnCancel() }
		items = append(items, cancel)
	}
	form := widget.NewColumn(16, items...)
	form.Style.Padding = geom.XY(28, 28)
	form.Style.FlexGrow = 1
	root := widget.NewColumn(0, form)
	root.Style.Background = widget.Paper
	c.Doc.SetRoot(c.cfg.Chrome.Wrap(c.Doc, root))
	c.Doc.ResetFrame()
	c.Doc.Invalidate()
}

// register adds the account with the tokens the provider gave.
func (c *Connect) register(providerID string, secret domain.Secret) error {
	acc, err := c.cfg.Accounts.RegisterOAuth(application.Registration{
		Name: c.name.Value, Email: c.email.Value, Provider: PresetOf(providerID),
	}, secret)
	if err != nil {
		return explainRegistration(c.cfg.Tr, err)
	}
	if c.cfg.OnDone != nil {
		c.cfg.OnDone(acc)
	}
	return nil
}
