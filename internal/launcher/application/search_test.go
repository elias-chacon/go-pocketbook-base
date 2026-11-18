package application

import (
	"errors"
	"testing"

	"go-pocketbook-base/internal/launcher/domain"
	platform "go-pocketbook-base/internal/platform/domain"
)

type fakeCatalog struct {
	list []domain.Listing
	err  error
}

func (c fakeCatalog) List(string) ([]domain.Listing, error) { return c.list, c.err }

func installedApp(id, title string) platform.InstalledApp {
	return platform.InstalledApp{Manifest: platform.Manifest{
		ID: id, Title: map[string]string{"pt-BR": title}, Description: map[string]string{"pt-BR": "d"},
	}}
}

func TestSearchSplitsInstalledFromAvailable(t *testing.T) {
	apps := registry{apps: []platform.InstalledApp{installedApp("hello", "Hello Go"), installedApp("launcher", "Launcher")}}
	cat := fakeCatalog{list: []domain.Listing{
		{ID: "hello", Title: "Hello Go"}, {ID: "launcher", Title: "Launcher"}, {ID: "helper", Title: "Helper"},
	}}
	res := Search{Apps: apps, Catalog: cat, SelfID: "launcher"}.Run("hel", "pt-BR")
	if len(res.Installed) != 1 || res.Installed[0].ID != "hello" {
		t.Errorf("installed: %+v", res.Installed)
	}
	if len(res.Available) != 1 || res.Available[0].ID != "helper" || res.CatalogErr != nil {
		t.Errorf("only what is not installed (nor the Launcher itself) is available: %+v %v", res.Available, res.CatalogErr)
	}
}

func TestSearchKeepsAnsweringWhenThereIsNoCatalogOrItFails(t *testing.T) {
	apps := registry{apps: []platform.InstalledApp{installedApp("hello", "Hello Go")}}
	res := Search{Apps: apps, SelfID: "launcher"}.Run("", "pt-BR")
	if len(res.Installed) != 1 || res.CatalogErr != domain.ErrNoCatalog {
		t.Errorf("%+v", res)
	}
	boom := errors.New("offline")
	res = Search{Apps: apps, Catalog: fakeCatalog{err: boom}}.Run("", "pt-BR")
	if len(res.Installed) != 1 || res.CatalogErr != boom || res.Available != nil {
		t.Errorf("%+v", res)
	}
}
