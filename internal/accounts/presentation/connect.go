package presentation

import (
	"errors"

	"go-pocketbook-base/internal/accounts/application"
	"go-pocketbook-base/internal/accounts/domain"
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/localization/i18n"
	"go-pocketbook-base/internal/toolkit/gfx"
	"go-pocketbook-base/internal/toolkit/widget"
)

// ConnectConfig is everything the "connect a to-do service" screen needs.
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

// The providers of the screen, as account presets.
const (
	keyCancel       = "signin.cancel"
	idConnectBack   = "connect-back"
	presetMicrosoft = "outlook"
	presetGoogle    = "google"
	qrSize          = 520
)

// Connect is the screen that signs in to Microsoft To Do or Google Tasks: the account the
// Tasks (and Calendar) apps synchronize with.
type Connect struct {
	Doc *widget.Document

	cfg                ConnectConfig
	preset             string
	email, name, typed *widget.Input
	note               *widget.Label
	challenge          *domain.Challenge
}

// NewConnect builds the screen on the choice of provider.
func NewConnect(cfg ConnectConfig) *Connect {
	c := &Connect{cfg: cfg, Doc: cfg.Doc}
	if c.Doc == nil {
		c.Doc = widget.NewDocument(cfg.Width, cfg.Height)
	}
	c.showChooser()
	return c
}

func (c *Connect) t(key string, args ...interface{}) string { return c.cfg.Tr.T(key, args...) }

func (c *Connect) button(id, label string, fn func()) *widget.Button {
	b := widget.NewButton(label)
	b.ID = id
	b.OnClick = func(*widget.Event) { fn() }
	return b
}

func (c *Connect) field(id, placeholder string) *widget.Input {
	in := widget.NewInput(placeholder)
	in.ID = id
	return in
}

func (c *Connect) show(title string, children ...widget.Widget) {
	head := widget.NewLabel(title)
	head.Style.FontSize, head.Style.Bold = widget.FontTitle, true
	form := widget.NewColumn(16, append([]widget.Widget{head}, children...)...)
	form.Style.Padding = geom.XY(28, 28)
	form.Style.FlexGrow = 1
	root := widget.NewColumn(0, form)
	root.Style.Background = widget.Paper
	c.Doc.SetRoot(c.cfg.Chrome.Wrap(c.Doc, root))
	c.Doc.ResetFrame()
	c.Doc.Invalidate()
}

func (c *Connect) say(text string) {
	c.note.Text = text
	c.Doc.Invalidate()
}

func (c *Connect) showChooser() {
	c.preset, c.challenge = "", nil
	intro := widget.NewLabel(c.t("connect.choose"))
	intro.Style.Color = widget.Muted
	items := []widget.Widget{
		intro,
		c.button("connect-microsoft", c.t("connect.microsoft"), func() { c.choose(presetMicrosoft) }),
		c.button("connect-google", c.t("connect.google"), func() { c.choose(presetGoogle) }),
	}
	if c.cfg.OnCancel != nil {
		items = append(items, c.button("connect-cancel", c.t(keyCancel), c.cfg.OnCancel))
	}
	c.show(c.t("connect.title"), items...)
}

func (c *Connect) configured() bool {
	if c.preset == presetGoogle {
		return c.cfg.Google != nil
	}
	return c.cfg.Microsoft != nil
}

// choose opens the form of a provider.
func (c *Connect) choose(preset string) {
	c.preset = preset
	c.note = widget.NewLabel("")
	c.note.Style.Bold = true
	back := c.button(idConnectBack, c.t("signin.back"), c.showChooser)
	if !c.configured() {
		hint := widget.NewLabel(c.t("connect.setup", c.cfg.OAuthSetup))
		hint.Style.Color = widget.Muted
		c.show(c.providerName(), hint, back)
		return
	}
	c.email = c.field("connect-email", c.t("signin.email"))
	c.name = c.field("connect-name", c.t("signin.name"))
	hint := widget.NewLabel(c.t("connect." + preset + ".hint"))
	hint.Style.FontSize, hint.Style.Color = widget.FontSmall, widget.Muted
	start := c.button("connect-start", c.t("signin.oauth.start"), c.Start)
	c.show(c.providerName(), c.email, c.name, hint, c.note, widget.NewRow(12, start, back))
}

func (c *Connect) providerName() string {
	if c.preset == presetGoogle {
		return c.t("connect.google")
	}
	return c.t("connect.microsoft")
}

// Start asks the provider for the sign-in: a code to type on another device (Microsoft),
// or an address to open on a phone, shown as a QR code (Google).
func (c *Connect) Start() {
	var ch domain.Challenge
	var err error
	if c.preset == presetGoogle {
		ch, err = c.cfg.Google.Start()
	} else {
		ch, err = c.cfg.Microsoft.Start()
	}
	if err != nil {
		c.say(c.t(keyFailed, err.Error()))
		return
	}
	c.challenge = &ch
	if c.preset == presetGoogle {
		c.showGoogle()
		return
	}
	c.showMicrosoft()
}

func (c *Connect) showMicrosoft() {
	steps := widget.NewLabel(c.t("signin.oauth.steps", c.challenge.VerificationURI))
	code := widget.NewLabel(c.challenge.UserCode)
	code.ID = "connect-code"
	code.Style.FontSize, code.Style.Bold = widget.FontTitle+12, true
	code.Style.TextAlign = gfx.AlignCenter
	c.note = widget.NewLabel("")
	c.note.Style.Bold = true
	check := c.button("connect-check", c.t("signin.oauth.check"), c.Check)
	cancel := c.button(idConnectBack, c.t(keyCancel), c.showChooser)
	c.show(c.providerName(), steps, code, c.note, widget.NewRow(12, check, cancel))
}

// Check asks whether the Microsoft code was approved; when it was, the account is added.
func (c *Connect) Check() {
	if c.challenge == nil || c.cfg.Microsoft == nil {
		return
	}
	c.say(c.t("signin.oauth.checking"))
	secret, pending, err := application.CheckOAuth(c.cfg.Microsoft, *c.challenge)
	switch {
	case pending:
		c.say(c.t("signin.oauth.pending"))
	case err != nil:
		c.say(c.explain(err))
	default:
		c.finish(secret)
	}
}

func (c *Connect) showGoogle() {
	steps := widget.NewLabel(c.t("connect.google.steps"))
	items := []widget.Widget{steps}
	if qr, err := widget.NewQRCode(c.challenge.VerificationURI, qrSize); err == nil {
		qr.ID = "connect-qr"
		items = append(items, qr)
	}
	c.typed = c.field("connect-code-input", c.t("connect.google.code"))
	c.note = widget.NewLabel("")
	c.note.Style.Bold = true
	done := c.button("connect-finish", c.t("connect.google.finish"), c.Finish)
	cancel := c.button(idConnectBack, c.t(keyCancel), c.showChooser)
	c.show(c.providerName(), append(items, c.typed, c.note, widget.NewRow(12, done, cancel))...)
}

// Finish trades the code typed (or the address it is in) for the tokens, and adds the account.
func (c *Connect) Finish() {
	if c.challenge == nil || c.cfg.Google == nil {
		return
	}
	c.say(c.t("signin.oauth.checking"))
	secret, err := c.cfg.Google.Exchange(*c.challenge, c.typed.Value)
	if err != nil {
		c.say(c.t(keyFailed, err.Error()))
		return
	}
	c.finish(secret)
}

func (c *Connect) finish(secret domain.Secret) {
	acc, err := c.cfg.Accounts.RegisterOAuth(application.Registration{Name: c.name.Value, Email: c.email.Value, Provider: c.preset}, secret)
	if err != nil {
		c.say(c.explain(err))
		return
	}
	c.challenge = nil
	if c.cfg.OnDone != nil {
		c.cfg.OnDone(acc)
	}
}

func (c *Connect) explain(err error) string {
	switch {
	case errors.Is(err, application.ErrInvalidEmail):
		return c.t("signin.invalid")
	case errors.Is(err, domain.ErrExists):
		return c.t("signin.exists")
	case errors.Is(err, domain.ErrDenied):
		return c.t("signin.oauth.denied")
	case errors.Is(err, domain.ErrExpired):
		return c.t("signin.oauth.expired")
	}
	return c.t(keyFailed, err.Error())
}
