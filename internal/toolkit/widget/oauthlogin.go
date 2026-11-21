package widget

import (
	"fmt"

	"go-pocketbook-base/internal/toolkit/gfx"
)

// OAuthChallenge is what a provider asks the person to do to sign in on another device (a phone
// or a computer: the reader has no browser).
type OAuthChallenge struct {
	// URL is the address to open on the other device.
	URL string
	// Code is the code to type there (the "device code" sign-in). When it is empty the address is
	// shown as a QR code and the person pastes back the code the provider gives at the end (the
	// "authorization code" sign-in).
	Code string
}

// OAuthProvider is one way to sign in: "Google", "Microsoft"... The widget knows nothing about
// OAuth itself; the app gives it the two steps.
type OAuthProvider struct {
	ID    string // identifies the provider in OnSuccess
	Label string // the text of its button
	// Start asks the provider for the challenge. Nil: the provider is not configured and its
	// button is disabled.
	Start func() (OAuthChallenge, error)
	// Complete finishes the sign-in. typed is what the person pasted (empty for a device code).
	// It answers pending when the person has not approved yet, or an error.
	Complete func(ch OAuthChallenge, typed string) (pending bool, err error)
}

// OAuthTexts are the words of the widget, for the app to translate.
type OAuthTexts struct {
	Intro, Cancel, Check, Finish, Checking, Pending, Done string
	// Steps explains the sign-in and has one %s: the address to open.
	Steps string
	// Scan explains the QR sign-in: an address is too long to print, so it is only in the code.
	Scan string
	// Pasted is the placeholder of the field for the code pasted back.
	Pasted string
	// Failed has one %s: the reason.
	Failed string
}

// DefaultOAuthTexts are the English words.
func DefaultOAuthTexts() OAuthTexts {
	return OAuthTexts{
		Intro: "Sign in with", Cancel: "Cancel", Check: "I approved it", Finish: "Finish",
		Checking: "Checking…", Pending: "Not approved yet. Try again in a moment.", Done: "Signed in.",
		Steps:  "On your phone or computer, open %s and sign in.",
		Scan:   "Scan the code with your phone, sign in and type here the code the page shows.",
		Pasted: "Code from the page", Failed: "It did not work: %s",
	}
}

// OAuthLogin is the sign-in with an online account (Google, Microsoft, any provider) for a device
// without a browser: a button per provider, then the two ways the providers do it on a screen like
// this. Either a code to type at an address (device code) or an address shown as a QR code with a
// field for the code pasted back (authorization code). It is built from existing widgets, so add
// Root to the document, like Pager.
//
// Start and Complete run on the event loop and block on the network: an app that wants the screen
// to stay alive wraps them with its own background work.
type OAuthLogin struct {
	// Root is the element to add to the document.
	Root *Panel
	// Texts are the words shown (English by default).
	Texts OAuthTexts
	// OnSuccess is called with the ID of the provider after Complete answered without error.
	OnSuccess func(providerID string)

	providers []OAuthProvider
	current   *OAuthProvider
	challenge OAuthChallenge
	input     *Input
	note      *Label
}

// NewOAuthLogin creates the sign-in with the given providers.
func NewOAuthLogin(providers ...OAuthProvider) *OAuthLogin {
	o := &OAuthLogin{Texts: DefaultOAuthTexts(), providers: providers}
	o.Root = NewColumn(16)
	o.Reset()
	return o
}

const oauthQRSize = 420

func (o *OAuthLogin) button(id, label string, fn func()) *Button {
	b := NewButton(label)
	b.ID = id
	b.OnClick = func(*Event) { fn() }
	return b
}

func (o *OAuthLogin) show(children ...Widget) {
	o.Root.ClearChildren()
	for _, c := range children {
		o.Root.AppendChild(c)
	}
	o.Root.Invalidate()
}

func (o *OAuthLogin) say(text string) {
	o.note.Text = text
	o.Root.Invalidate()
}

// Reset goes back to the choice of provider.
func (o *OAuthLogin) Reset() {
	o.current, o.input = nil, nil
	o.note = NewLabel("")
	o.note.ID = "oauth-note"
	o.note.Style.Bold = true
	intro := NewLabel(o.Texts.Intro)
	intro.Style.Color = Muted
	items := []Widget{intro}
	for i := range o.providers {
		p := &o.providers[i]
		b := o.button("oauth-"+p.ID, p.Label, func() { o.choose(p) })
		b.Disabled = p.Start == nil
		items = append(items, b)
	}
	o.show(append(items, o.note)...)
}

// choose asks the provider for its challenge and shows what the person has to do.
func (o *OAuthLogin) choose(p *OAuthProvider) {
	ch, err := p.Start()
	if err != nil {
		o.say(fmt.Sprintf(o.Texts.Failed, err.Error()))
		return
	}
	o.current, o.challenge = p, ch
	o.note = NewLabel("")
	o.note.ID = "oauth-note"
	o.note.Style.Bold = true
	var steps *Label
	action := o.Texts.Check
	if ch.Code != "" {
		steps = NewLabel(fmt.Sprintf(o.Texts.Steps, ch.URL))
	} else {
		steps = NewLabel(o.Texts.Scan)
	}
	items := []Widget{steps}
	if ch.Code != "" {
		code := NewLabel(ch.Code)
		code.ID = "oauth-code"
		code.Style.FontSize, code.Style.Bold, code.Style.TextAlign = FontTitle+12, true, gfx.AlignCenter
		items = append(items, code)
	} else {
		if qr, err := NewQRCode(ch.URL, oauthQRSize); err == nil {
			qr.ID = "oauth-qr"
			items = append(items, qr)
		}
		o.input = NewInput(o.Texts.Pasted)
		o.input.ID = "oauth-input"
		items = append(items, o.input)
		action = o.Texts.Finish
	}
	row := NewRow(12, o.button("oauth-complete", action, o.complete), o.button("oauth-cancel", o.Texts.Cancel, o.Reset))
	o.show(append(items, o.note, row)...)
}

// complete asks the provider whether the sign-in is done.
func (o *OAuthLogin) complete() {
	if o.current == nil || o.current.Complete == nil {
		return
	}
	o.say(o.Texts.Checking)
	typed := ""
	if o.input != nil {
		typed = o.input.Value
	}
	pending, err := o.current.Complete(o.challenge, typed)
	switch {
	case pending:
		o.say(o.Texts.Pending)
	case err != nil:
		o.say(fmt.Sprintf(o.Texts.Failed, err.Error()))
	default:
		id := o.current.ID
		o.Reset()
		o.say(o.Texts.Done)
		if o.OnSuccess != nil {
			o.OnSuccess(id)
		}
	}
}
