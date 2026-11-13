// examples/login.go
package examples

import "go-pocketbook-base/internal/widget"

// LoginForm is a sign-in screen with validation.
type LoginForm struct {
	Root     widget.Widget
	User     *widget.Input
	Password *widget.Input
	Remember *widget.Toggle
	Message  *widget.Label
}

// NewLoginForm builds the form. onLogin runs only when the data is valid.
func NewLoginForm(onLogin func(user string, remember bool)) *LoginForm {
	f := &LoginForm{
		User:     widget.NewInput("User name"),
		Password: widget.NewInput("Password"),
		Remember: widget.NewCheckbox("Remember me", true),
		Message:  widget.NewLabel(""),
	}
	f.Password.Type = widget.TypePassword
	f.Message.Hidden = true // shown only when there is an error
	f.Message.Style.Bold = true

	submit := widget.NewButton("Sign in")
	submit.SetVariant(widget.VariantPrimary)
	submit.OnClick = func(e *widget.Event) {
		switch {
		case f.User.Value == "":
			f.fail("Type your user name")
		case len(f.Password.Value) < 4:
			f.fail("The password needs at least 4 characters")
		default:
			f.Message.Hidden = true
			onLogin(f.User.Value, f.Remember.Checked)
		}
	}

	f.Root = widget.NewCard("Sign in", "Touch a field to type",
		f.User, f.Password, f.Remember, f.Message, submit)
	return f
}

func (f *LoginForm) fail(text string) {
	f.Message.Text = text
	f.Message.Hidden = false
}
