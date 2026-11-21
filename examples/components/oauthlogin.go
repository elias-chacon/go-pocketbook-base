package components

import (
	"errors"
	"fmt"

	"go-pocketbook-base/internal/toolkit/widget"
)

// OAuthLoginExample is the sign-in with a Google or Microsoft account for a screen without a browser:
// a button per provider, then a code to type at an address (Microsoft) or an address shown as a QR
// code with a field for the code pasted back (Google). The widget only asks; the app gives it the
// two steps of its OAuth client.
func OAuthLoginExample() widget.Widget {
	login := widget.NewOAuthLogin(
		widget.OAuthProvider{ID: "microsoft", Label: "Microsoft",
			Start: func() (widget.OAuthChallenge, error) {
				return widget.OAuthChallenge{URL: "https://microsoft.com/devicelogin", Code: "AB12-CD34"}, nil
			},
			Complete: func(widget.OAuthChallenge, string) (bool, error) {
				return false, nil // true: the person has not approved yet
			}},
		widget.OAuthProvider{ID: "google", Label: "Google",
			Start: func() (widget.OAuthChallenge, error) {
				return widget.OAuthChallenge{URL: "https://accounts.google.com/o/oauth2/v2/auth?client_id=..."}, nil
			},
			Complete: func(_ widget.OAuthChallenge, code string) (bool, error) {
				if code == "" {
					return false, errors.New("paste the code first")
				}
				return false, nil
			}},
	)
	login.OnSuccess = func(provider string) { fmt.Println("signed in with", provider) }
	return login.Root
}
