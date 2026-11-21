package showcase

import (
	"errors"

	"go-pocketbook-base/internal/toolkit/widget"
)

// ---- Conta: the sign-in with Google and Microsoft (and the long list of languages, on Escolhas) ----

// languages are sample data (not translated): enough of them that the combo box needs its search and
// its pages.
var languages = []string{"Afrikaans", "Alemão", "Árabe", "Armênio", "Basco", "Búlgaro", "Catalão", "Chinês", "Coreano", "Croata", "Dinamarquês", "Eslovaco", "Esloveno", "Espanhol", "Estoniano", "Finlandês", "Francês", "Galego", "Georgiano", "Grego", "Hebraico", "Hindi", "Holandês", "Húngaro", "Indonésio", "Inglês", "Irlandês", "Islandês", "Italiano", "Japonês", "Latim", "Letão", "Lituano", "Norueguês", "Persa", "Polonês", "Português", "Romeno", "Russo", "Sérvio", "Sueco", "Tailandês", "Tcheco", "Turco", "Ucraniano", "Vietnamita"}

func (a *App) accountPage() widget.Widget {
	return twoColPage(
		widget.NewCard(a.t("account.card.login.title"), a.t("account.card.login.subtitle"), a.signIn()),
	)
}

// longListCard is a combo box with 46 languages: it gets its search field and its pages by itself.
func (a *App) longListCard() widget.Widget {
	var options []widget.Option
	for _, name := range languages {
		options = append(options, widget.Option{Label: name, Value: name})
	}
	lang := widget.NewComboBox(a.t("account.language.placeholder"), options...)
	lang.SearchPlaceholder, lang.NoMatchText = a.t("account.language.search"), a.t("account.language.none")
	lang.SetValue("Português")
	lang.OnChange = func(e *widget.Event) { a.say(a.t("say.language", lang.Value())) }
	return widget.NewCard(a.t("account.card.combo.title"), a.t("account.card.combo.subtitle"),
		field(a.t("account.field.language"), withID(lang, "combo-language")))
}

// signIn is the OAuthLogin: with the providers the app gave (the real sign-ins when oauth.json has the
// client ids; the demo keeps no token) or, without any, two simulated ones that send nothing.
func (a *App) signIn() widget.Widget {
	address := widget.NewInput(a.t("account.login.address"))
	address.ID = "oauth-address"
	var providers []widget.OAuthProvider
	if a.cfg.OAuth != nil {
		providers = a.cfg.OAuth(func() string { return address.Value })
	}
	if len(providers) == 0 {
		providers = a.simulatedProviders()
	}
	login := widget.NewOAuthLogin(providers...)
	login.Texts = widget.OAuthTexts{
		Intro: a.t("account.login.intro"), Cancel: a.t("account.login.cancel"), Check: a.t("account.login.check"),
		Finish: a.t("account.login.finish"), Checking: a.t("account.login.checking"), Pending: a.t("account.login.pending"),
		Done: a.t("account.login.done"), Steps: a.t("account.login.steps"), Scan: a.t("account.login.scan"),
		Pasted: a.t("account.login.pasted"), Failed: a.t("account.login.failed"),
	}
	login.Reset() // the words were set after the first build
	login.OnSuccess = func(id string) { a.say(a.t("say.signedin", id)) }
	items := []widget.Widget{address, login.Root}
	for _, p := range providers {
		if p.Start == nil {
			items = append(items, hint(a.t("account.login.setup", a.cfg.OAuthSetup)))
			break
		}
	}
	return widget.NewColumn(16, items...)
}

// simulatedProviders are two providers that answer by themselves, for the preview and the tests.
func (a *App) simulatedProviders() []widget.OAuthProvider {
	checks := 0
	return []widget.OAuthProvider{
		{ID: "google", Label: a.t("account.login.google"),
			Start: func() (widget.OAuthChallenge, error) {
				return widget.OAuthChallenge{URL: "https://accounts.example.com/demo-google"}, nil
			},
			Complete: func(_ widget.OAuthChallenge, typed string) (bool, error) {
				if typed == "" {
					return false, errors.New(a.t("account.login.empty"))
				}
				return false, nil
			}},
		{ID: "microsoft", Label: a.t("account.login.microsoft"),
			Start: func() (widget.OAuthChallenge, error) {
				checks = 0
				return widget.OAuthChallenge{URL: "https://login.example.com/demo-device", Code: "HELLO-2024"}, nil
			},
			// The first answer is "not approved yet", like a person who is still typing the code.
			Complete: func(widget.OAuthChallenge, string) (bool, error) { checks++; return checks < 2, nil }},
	}
}
