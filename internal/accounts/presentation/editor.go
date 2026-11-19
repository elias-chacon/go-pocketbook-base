package presentation

import (
	"strconv"
	"strings"

	"go-pocketbook-base/internal/accounts/domain"
	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/localization/i18n"
	"go-pocketbook-base/internal/toolkit/widget"
)

// Manager is what the account editor needs of the Accounts context.
type Manager interface {
	Update(domain.Account) error
	Refresh(id string, s domain.Secret) error
	Remove(id string) error
}

// EditorConfig is everything the account editor needs.
type EditorConfig struct {
	Width, Height int
	Doc           *widget.Document
	Chrome        widget.AppChrome
	Tr            *i18n.Bundle
	Account       domain.Account
	Manager       Manager
	// Test connects with the settings and the password typed (the stored one when
	// blank) and says what went wrong; nil hides the button.
	Test func(acc domain.Account, password string) error
	// OnSaved is called with the saved account, OnRemoved after Remove, OnBack when the
	// person leaves without saving.
	OnSaved   func(domain.Account)
	OnRemoved func()
	OnBack    func()
}

const (
	signatureRows = 4
	idEditNote    = "acct-note"
	keyEditFailed = "acct.failed"
)

// endpointForm is the fields of one server.
type endpointForm struct {
	kind         domain.Kind
	host, port   *widget.Input
	security     domain.Security
	securityMenu *widget.Dropdown
}

// Editor is the "edit account" screen: name, login, password, the incoming (IMAP) and
// outgoing (SMTP) servers with their port and protection, and the signature.
type Editor struct {
	Doc *widget.Document

	cfg                   EditorConfig
	name, login, password *widget.Input
	signature             *widget.TextArea
	incoming, outgoing    *endpointForm
	note                  *widget.Label
	remove                *widget.Button
	confirming            bool
	oauth                 bool
}

// NewEditor builds the screen for an account.
func NewEditor(cfg EditorConfig) *Editor {
	e := &Editor{cfg: cfg, Doc: cfg.Doc, oauth: cfg.Account.Auth == domain.OAuth2}
	if e.Doc == nil {
		e.Doc = widget.NewDocument(cfg.Width, cfg.Height)
	}
	e.build()
	return e
}

func (e *Editor) t(key string, args ...interface{}) string { return e.cfg.Tr.T(key, args...) }

func (e *Editor) field(id, placeholder, value string, kind widget.InputType) *widget.Input {
	in := widget.NewInput(placeholder)
	in.ID, in.Type, in.Value = id, kind, value
	return in
}

func (e *Editor) button(id, label string, fn func()) *widget.Button {
	b := widget.NewButton(label)
	b.ID = id
	b.OnClick = func(*widget.Event) { fn() }
	return b
}

// server builds the fields of one endpoint of the account (empty when it has none).
func (e *Editor) server(kind domain.Kind, prefix string, fallback domain.Security) *endpointForm {
	ep, ok := e.cfg.Account.Endpoint(domain.Mail, kind)
	f := &endpointForm{kind: kind, security: ep.Security}
	if !ok {
		f.security = fallback
	}
	f.host = e.field(prefix+"-host", e.t("acct."+prefix+".host"), ep.Host, widget.TypeText)
	port := ""
	if ep.Port > 0 {
		port = strconv.Itoa(ep.Port)
	}
	f.host.Style.FlexGrow = 1
	f.port = e.field(prefix+"-port", e.t("acct.port"), port, widget.TypeText)
	f.port.Style.Width = widget.Px(180)
	f.securityMenu = widget.NewDropdown(e.securityLabel(f.security),
		widget.MenuItem{Label: e.securityLabel(domain.TLS), OnSelect: func() { f.setSecurity(e, domain.TLS) }},
		widget.MenuItem{Label: e.securityLabel(domain.StartTLS), OnSelect: func() { f.setSecurity(e, domain.StartTLS) }},
	)
	f.securityMenu.ID = prefix + "-security"
	return f
}

func (f *endpointForm) setSecurity(e *Editor, s domain.Security) {
	f.security = s
	f.securityMenu.Text = e.securityLabel(s)
}

func (e *Editor) securityLabel(s domain.Security) string {
	if s == domain.StartTLS {
		return e.t("acct.starttls")
	}
	return e.t("acct.tls")
}

func (e *Editor) build() {
	acc := e.cfg.Account
	title := widget.NewLabel(e.t("acct.title", acc.Email))
	title.Style.FontSize, title.Style.Bold = widget.FontTitle, true

	e.name = e.field("acct-name", e.t("acct.name"), acc.Name, widget.TypeText)
	e.signature = widget.NewTextArea(e.t("acct.signature"), signatureRows)
	e.signature.ID, e.signature.Value = "acct-signature", acc.Signature
	e.signature.Style.Width = widget.Pct(100)
	e.note = widget.NewLabel("")
	e.note.ID = idEditNote
	e.note.Style.Bold = true

	children := []widget.Widget{title, e.name}
	if e.oauth {
		hint := widget.NewLabel(e.t("acct.oauth"))
		hint.Style.FontSize, hint.Style.Color = widget.FontSmall, widget.Muted
		children = append(children, hint)
	} else {
		e.login = e.field("acct-login", e.t("acct.login"), acc.Username, widget.TypeText)
		e.password = e.field("acct-password", e.t("acct.password"), "", widget.TypePassword)
		e.incoming = e.server(domain.IMAP, "imap", domain.TLS)
		e.outgoing = e.server(domain.SMTP, "smtp", domain.StartTLS)
		children = append(children, e.login, e.password,
			e.heading("acct.incoming"), widget.NewRow(12, e.incoming.host, e.incoming.port, e.incoming.securityMenu),
			e.heading("acct.outgoing"), widget.NewRow(12, e.outgoing.host, e.outgoing.port, e.outgoing.securityMenu))
	}
	children = append(children, e.heading("acct.signature.title"), e.signature, e.note, e.actions())

	form := widget.NewColumn(14, children...)
	form.Style.Padding = geom.XY(28, 28)
	form.Style.FlexGrow = 1
	root := widget.NewColumn(0, form)
	root.Style.Background = widget.Paper
	e.Doc.SetRoot(e.cfg.Chrome.Wrap(e.Doc, root))
	e.Doc.ResetFrame()
	e.Doc.Invalidate()
}

func (e *Editor) heading(key string) widget.Widget {
	l := widget.NewLabel(e.t(key))
	l.Style.Bold = true
	return l
}

func (e *Editor) actions() widget.Widget {
	save := e.button("acct-save", e.t("acct.save"), e.Save)
	row := widget.NewRow(12, save)
	if e.cfg.Test != nil && !e.oauth {
		row.AppendChild(e.button("acct-test", e.t("acct.test"), e.Test))
	}
	e.remove = e.button("acct-remove", e.t("acct.remove"), e.Remove)
	row.AppendChild(e.remove)
	row.AppendChild(e.button("acct-back", e.t("acct.back"), func() {
		if e.cfg.OnBack != nil {
			e.cfg.OnBack()
		}
	}))
	return row
}

// read copies the form into an account; the problem it finds is returned as text.
func (e *Editor) read() (domain.Account, string) {
	acc := e.cfg.Account
	acc.Name = strings.TrimSpace(e.name.Value)
	if acc.Name == "" {
		acc.Name = acc.Email
	}
	acc.Signature = strings.TrimRight(e.signature.Value, " \n")
	if e.oauth {
		return acc, ""
	}
	acc.Username = strings.TrimSpace(e.login.Value)
	if acc.Username == acc.Email {
		acc.Username = ""
	}
	services := map[domain.Service][]domain.Endpoint{}
	for s, eps := range acc.Services {
		services[s] = eps
	}
	var mailEPs []domain.Endpoint
	for _, ep := range acc.Services[domain.Mail] {
		if ep.Kind != domain.IMAP && ep.Kind != domain.SMTP {
			mailEPs = append(mailEPs, ep)
		}
	}
	for _, f := range []*endpointForm{e.incoming, e.outgoing} {
		ep, problem := e.readServer(f)
		if problem != "" {
			return acc, problem
		}
		if ep.Host != "" {
			mailEPs = append(mailEPs, ep)
		}
	}
	services[domain.Mail] = mailEPs
	acc.Services = services
	return acc, ""
}

func (e *Editor) readServer(f *endpointForm) (domain.Endpoint, string) {
	host := strings.TrimSpace(f.host.Value)
	if host == "" {
		return domain.Endpoint{}, ""
	}
	port, err := strconv.Atoi(strings.TrimSpace(f.port.Value))
	if err != nil || port < 1 || port > 65535 {
		return domain.Endpoint{}, e.t("acct.badport", host)
	}
	return domain.Endpoint{Kind: f.kind, Host: host, Port: port, Security: f.security}, ""
}

func (e *Editor) say(text string) {
	e.note.Text = text
	e.Doc.Invalidate()
}

// Save validates the form and keeps the account (and the new password, when one was typed).
func (e *Editor) Save() {
	acc, problem := e.read()
	if problem != "" {
		e.say(problem)
		return
	}
	if err := e.cfg.Manager.Update(acc); err != nil {
		e.say(e.t(keyEditFailed, err.Error()))
		return
	}
	if !e.oauth && e.password.Value != "" {
		if err := e.cfg.Manager.Refresh(acc.ID, domain.Secret{Password: e.password.Value}); err != nil {
			e.say(e.t(keyEditFailed, err.Error()))
			return
		}
		e.password.Value = "" // the password does not stay in the widget
	}
	e.cfg.Account = acc
	if e.cfg.OnSaved != nil {
		e.cfg.OnSaved(acc)
	}
}

// Test connects with what is typed, to find a wrong password or server before saving.
func (e *Editor) Test() {
	acc, problem := e.read()
	if problem != "" {
		e.say(problem)
		return
	}
	e.say(e.t("acct.testing"))
	if err := e.cfg.Test(acc, e.password.Value); err != nil {
		e.say(e.t("acct.test.failed", err.Error()))
		return
	}
	e.say(e.t("acct.test.ok"))
}

// Remove asks to be tapped twice: the first tap asks for confirmation.
func (e *Editor) Remove() {
	if !e.confirming {
		e.confirming = true
		e.remove.Text = e.t("acct.remove.confirm")
		e.say(e.t("acct.remove.warn"))
		return
	}
	if err := e.cfg.Manager.Remove(e.cfg.Account.ID); err != nil {
		e.say(e.t(keyEditFailed, err.Error()))
		return
	}
	if e.cfg.OnRemoved != nil {
		e.cfg.OnRemoved()
	}
}
