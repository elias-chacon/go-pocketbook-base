package application

import (
	"embed"

	"go-pocketbook-base/internal/localization/i18n"
	"go-pocketbook-base/internal/platform/domain"
)

//go:embed locales/*.json
var locales embed.FS

// translator is the platform's own messages (the first-run notice).
func translator(lang string) *i18n.Bundle {
	tr := i18n.New()
	if err := tr.AddFS(locales, "locales"); err != nil {
		panic(err) // embedded files: an error is a programming mistake
	}
	tr.SetLang(lang)
	return tr
}

// FirstRun is the use case "register this app": PocketBook has no installer, so
// copying the .app is the installation and the app finishes the job the first
// time it runs (and records itself for the Launcher on every run).
type FirstRun struct {
	Registrar domain.DesktopRegistrar
	Apps      domain.AppRegistry
	// Logf is optional: where to report what happened (for the on-device log).
	Logf func(format string, args ...interface{})
}

// FirstRunInput is what the use case needs to know about the running app.
type FirstRunInput struct {
	Manifest domain.Manifest
	Lang     string
	Version  string
	Exe      string              // absolute path of the running <id>.app
	IconPath string              // where the registrar writes the icon (recorded for the Launcher)
	Icon     domain.IconRenderer // nil uses DefaultIcon
	Replace  bool                // an install: replace what an older version registered (see domain.Registration)
}

// Execute registers the app on the home screen and records it. It returns the
// message to show the user, or "" when there is nothing to say.
func (f FirstRun) Execute(in FirstRunInput) string {
	tr := translator(in.Lang)
	title := in.Manifest.TitleIn(in.Lang)
	icon := in.Icon
	if icon == nil {
		icon = DefaultIcon(title)
	}

	changed, err := f.Registrar.Register(domain.Registration{
		Exe: in.Exe, Title: title, Group: in.Manifest.DesktopGroup(), Icon: icon, Replace: in.Replace,
	})
	f.log("register: changed=%v err=%v", changed, err)
	if err != nil {
		return tr.T("install.failed", err.Error())
	}

	record := domain.InstalledApp{Manifest: in.Manifest, Version: in.Version, Exe: in.Exe, Icon: in.IconPath}
	if err := f.Apps.Save(record); err != nil {
		f.log("record: %v", err) // not being listed must not stop the app
	}
	if changed {
		return tr.T("install.registered")
	}
	return ""
}

func (f FirstRun) log(format string, args ...interface{}) {
	if f.Logf != nil {
		f.Logf(format, args...)
	}
}
