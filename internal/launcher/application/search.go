package application

import (
	"go-pocketbook-base/internal/launcher/domain"
	platform "go-pocketbook-base/internal/platform/domain"
)

// Search is the use case "search the apps": the installed ones that match and
// the ones of the catalog that match and are not installed.
type Search struct {
	Apps    platform.AppRegistry
	Catalog domain.Catalog // nil when the Launcher has no catalog
	SelfID  string
}

// Run searches both lists for query, in lang. A catalog that fails does not
// fail the search: the installed apps are still answered, with the reason.
func (s Search) Run(query, lang string) domain.SearchResult {
	installed := domain.TilesFrom(s.Apps.List(), lang, s.SelfID)
	res := domain.SearchResult{Installed: domain.FilterTiles(installed, query)}

	if s.Catalog == nil {
		res.CatalogErr = domain.ErrNoCatalog
		return res
	}
	list, err := s.Catalog.List(lang)
	if err != nil {
		res.CatalogErr = err
		return res
	}
	ids := []string{s.SelfID}
	for _, t := range installed {
		ids = append(ids, t.ID)
	}
	res.Available = domain.FilterListings(list, query, ids)
	return res
}
