package showcase

import (
	"errors"

	"go-pocketbook-base/internal/toolkit/widget"
)

// ---- Conta: long combo box and the sign-in with Google and Microsoft ----

// languages are sample data (not translated): enough of them that the combo box needs its search and
// its pages.
var languages = []string{"Afrikaans", "Alemão", "Árabe", "Armênio", "Basco", "Búlgaro", "Catalão", "Chinês", "Coreano", "Croata", "Dinamarquês", "Eslovaco", "Esloveno", "Espanhol", "Estoniano", "Finlandês", "Francês", "Galego", "Georgiano", "Grego", "Hebraico", "Hindi", "Holandês", "Húngaro", "Indonésio", "Inglês", "Irlandês", "Islandês", "Italiano", "Japonês", "Latim", "Letão", "Lituano", "Norueguês", "Persa", "Polonês", "Português", "Romeno", "Russo", "Sérvio", "Sueco", "Tailandês", "Tcheco", "Turco", "Ucraniano", "Vietnamita"}

func (a *App) accountPage() widget.Widget {
	var options []widget.Option
	for _, name := range languages {
		options = append(options, widget.Option{Label: name, Value: name})
	}
	lang := widget.NewComboBox(a.t("account.language.placeholder"), options...)
	lang.SearchPlaceholder, lang.NoMatchText = a.t("account.language.search"), a.t("account.language.none")
	lang.SetValue("Português")
	lang.OnChange = func(e *widget.Event) { a.say(a.t("say.language", lang.Value())) }

	return twoColPage(
		widget.NewCard(a.t("account.card.combo.title"), a.t("account.card.combo.subtitle"),
			field(a.t("account.field.language"), withID(lang, "combo-language"))),
		widget.NewCard(a.t("account.card.login.title"), a.t("account.card.login.subtitle"), a.signIn()),
	)
}

// signIn is the OAuthLogin with two simulated providers: nothing leaves the reader. A real app gives
// it the provider of its OAuth client (see the Tasks app).
func (a *App) signIn() widget.Widget {
	checks := 0
	login := widget.NewOAuthLogin(
		widget.OAuthProvider{ID: "google", Label: a.t("account.login.google"),
			Start: func() (widget.OAuthChallenge, error) {
				return widget.OAuthChallenge{URL: "https://accounts.example.com/demo-google"}, nil
			},
			Complete: func(_ widget.OAuthChallenge, typed string) (bool, error) {
				if typed == "" {
					return false, errors.New(a.t("account.login.empty"))
				}
				return false, nil
			}},
		widget.OAuthProvider{ID: "microsoft", Label: a.t("account.login.microsoft"),
			Start: func() (widget.OAuthChallenge, error) {
				checks = 0
				return widget.OAuthChallenge{URL: "https://login.example.com/demo-device", Code: "HELLO-2024"}, nil
			},
			// The first answer is "not approved yet", like a person who is still typing the code.
			Complete: func(widget.OAuthChallenge, string) (bool, error) { checks++; return checks < 2, nil }},
	)
	login.Texts = widget.OAuthTexts{
		Intro: a.t("account.login.intro"), Cancel: a.t("account.login.cancel"), Check: a.t("account.login.check"),
		Finish: a.t("account.login.finish"), Checking: a.t("account.login.checking"), Pending: a.t("account.login.pending"),
		Done: a.t("account.login.done"), Steps: a.t("account.login.steps"), Scan: a.t("account.login.scan"), Pasted: a.t("account.login.pasted"),
		Failed: a.t("account.login.failed"),
	}
	login.Reset() // the words were set after the first build
	login.OnSuccess = func(id string) { a.say(a.t("say.signedin", id)) }
	return login.Root
}
