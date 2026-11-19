package main

import (
	"image"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go-pocketbook-base/internal/launcher/application"
	"go-pocketbook-base/internal/launcher/domain"
	"go-pocketbook-base/internal/launcher/infrastructure/appinstaller"
	"go-pocketbook-base/internal/launcher/infrastructure/github"
	"go-pocketbook-base/internal/launcher/infrastructure/iconcache"
	"go-pocketbook-base/internal/launcher/infrastructure/processlauncher"
	"go-pocketbook-base/internal/launcher/presentation"
	"go-pocketbook-base/internal/platform"
	"go-pocketbook-base/internal/toolkit/gfx"
	"go-pocketbook-base/internal/toolkit/widget"
)

// The repository whose latest release is the catalog of apps.
const (
	repoOwner = "elias-chacon"
	repoName  = "go-pocketbook-base"
	// httpTimeout is the client's timeout when the platform gives none (tests).
	httpTimeout = 60 * time.Second
)

// iconEdge is the size of the icons the launcher asks for (they are written at 128).
const iconEdge = 256

// launcher is the app: the platform's Env wired to the launcher context.
type launcher struct{ home *presentation.Home }

// newLauncher is the platform.Factory.
func newLauncher(env platform.Env) platform.App {
	self, err := os.Executable()
	if err != nil {
		self = "" // an app started by us then cannot find its way back: it just ends
	}
	client := env.HTTP // the platform's: it works on the reader's network
	if client == nil {
		client = &http.Client{Timeout: httpTimeout}
	}
	icons := iconcache.Cache{Client: client, Dir: filepath.Join(platform.DefaultRoot, "applications", ".pbgo", "cache", "icons")}
	registrar := iconRegistrar{icons: icons}
	catalog := &loggedCatalog{inner: &github.Catalog{Client: client, Owner: repoOwner, Repo: repoName}}
	return &launcher{home: presentation.NewHome(presentation.Config{
		Width: env.Width, Height: env.Height, Keyboard: env.Keyboard,
		Tr:   presentation.NewTranslator(env.Lang),
		Grid: domain.DefaultGrid,
		Home: application.Home{
			Apps:     env.Apps,
			Launcher: processlauncher.Launcher{Self: self},
			SelfID:   env.Manifest.ID,
		},
		IconFor: iconFor,
		Notice:  env.Notice,
		Catalog: catalog, Installer: loggedInstaller{inner: appinstaller.Installer{
			Client: client, Root: platform.DefaultRoot, Register: registrar.register,
		}},
		Flush: env.Flush,
		ListingIcon: func(l domain.Listing) image.Image {
			img, err := icons.Image(l)
			if err != nil && err != iconcache.ErrNoIcon {
				platform.Logf("icon %s: %v", l.ID, err)
			}
			return img
		},
	})}
}

// loggedCatalog and loggedInstaller write what goes wrong on the network to the
// app's log, where it can be read from the reader's drive.
type loggedCatalog struct{ inner domain.Catalog }

func (c *loggedCatalog) List(lang string) ([]domain.Listing, error) {
	list, err := c.inner.List(lang)
	if err != nil {
		platform.Logf("catalog: %v", err)
	}
	return list, err
}

type loggedInstaller struct{ inner domain.Installer }

func (i loggedInstaller) Install(l domain.Listing) error {
	err := i.inner.Install(l)
	if err != nil {
		platform.Logf("install %s: %v", l.ID, err)
	}
	return err
}

// register puts a downloaded app on the home screen with the platform's own
// registration. The app is not run here, so it cannot draw its own icon: it uses
// the icon.png the release published for it.
func (i iconRegistrar) register(root string, l domain.Listing) error {
	_, err := platform.RegisterOnDrive(root, l.Manifest, i.iconOf(l), l.Version)
	return err
}

// iconRegistrar registers downloaded apps with the icon the catalog published
// (already in the cache from the search), or the default one without it.
type iconRegistrar struct{ icons iconcache.Cache }

func (i iconRegistrar) iconOf(l domain.Listing) func(size int, focused bool) image.Image {
	img, err := i.icons.Image(l)
	if err != nil {
		platform.Logf("icon %s: %v (the default icon is used)", l.ID, err)
		return nil
	}
	return platform.IconFromImage(img)
}

// Document implements platform.App.
func (l *launcher) Document() *widget.Document { return l.home.Doc }

// Key implements platform.KeyHandler. The platform already used left and Menu to
// close a popup or a modal; with nothing open, left and right change the page.
func (l *launcher) Key(key platform.Key) bool {
	switch key {
	case platform.KeyPrevious:
		l.home.PrevPage()
	case platform.KeyNext:
		l.home.NextPage()
	default:
		return false
	}
	return true
}

// iconFor loads the BMP icon an app wrote when it first ran, or draws the
// default one (the initial of the title) when it is missing or unreadable.
func iconFor(t domain.Tile) image.Image {
	if t.IconPath != "" {
		if img, err := gfx.LoadImage(t.IconPath, iconEdge, iconEdge); err == nil {
			return img
		}
	}
	return platform.DefaultIcon(t.Title)(128, false)
}
