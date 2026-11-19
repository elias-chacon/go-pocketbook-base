package application

import "go-pocketbook-base/internal/toolkit/widget"

// Chrome is the standard top bar of the app (its name with About and Exit), in the
// language of the reader. The screens of the app put it above each of their roots.
func Chrome(env Env) widget.AppChrome {
	tr := translator(env.Lang)
	return widget.AppChrome{
		Title:        env.Manifest.TitleIn(env.Lang),
		Description:  env.Manifest.DescriptionIn(env.Lang),
		Version:      env.Version,
		LabelAbout:   tr.T("chrome.about"),
		LabelExit:    tr.T("chrome.exit"),
		LabelVersion: tr.T("chrome.version"),
		Exit:         env.Exit,
	}
}
