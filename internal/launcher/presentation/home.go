// Package presentation draws the Launcher: the screens, built from the toolkit's
// widgets, over the launcher's use cases. It holds no rules of its own.
package presentation

import (
	"embed"
	"image"
	"strings"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/launcher/application"
	"go-pocketbook-base/internal/launcher/domain"
	"go-pocketbook-base/internal/localization/i18n"
	"go-pocketbook-base/internal/toolkit/gfx"
	"go-pocketbook-base/internal/toolkit/widget"
)

//go:embed locales/*.json
var locales embed.FS

// NewTranslator loads the Launcher's texts (pt-BR first, English fallback) in lang.
func NewTranslator(lang string) *i18n.Bundle {
	tr := i18n.New()
	if err := tr.AddFS(locales, "locales"); err != nil {
		panic(err) // embedded files: an error is a programming mistake
	}
	tr.SetLang(lang)
	return tr
}

// Config is everything the home screen needs.
type Config struct {
	Width, Height int
	Keyboard      widget.Keyboard
	Tr            *i18n.Bundle
	Home          application.Home
	// Prefs are the Launcher's preferences: the grid, the icon size, the background.
	Prefs domain.Preferences
	// Languages are the choices of the language setting (DefaultLanguages when nil).
	Languages []Language
	// SavePrefs and SaveLang keep what the settings screen changes (nil: it is not kept).
	SavePrefs func(domain.Preferences) error
	SaveLang  func(lang string) error
	// Backgrounds lists the pictures a background can be chosen from and
	// LoadBackground opens one by name (nil: there are no backgrounds).
	Backgrounds    func() []string
	LoadBackground func(name string) (image.Image, error)
	// Catalog lists the apps that can be installed (nil: the search only knows the installed ones).
	Catalog domain.Catalog
	// Installer installs an app of the catalog when its row is tapped (nil: the rows only show).
	Installer domain.Installer
	// Flush paints pending changes now; it is called before a download blocks the
	// screen so the message is seen (nil: no flush).
	Flush func()
	// ListingIcon is the icon of an app of the catalog, shown in its row (nil, or a
	// nil result: the row has no icon).
	ListingIcon func(l domain.Listing) image.Image
	// IconFor loads the icon of a tile (nil results draw nothing); the composition
	// root decides where icons come from and what to show when one is missing.
	IconFor func(t domain.Tile) image.Image
	Notice  string
}

// While searching, the installed results use fewer rows so the "Available" section fits the same page.
const (
	searchRows   = 3
	availableMax = 4
	// listingIconSize is the edge of the icon in a row of the catalog, in design pixels.
	listingIconSize = 72
)

// Home is the main screen: a paged grid of the installed apps and a search field
// that splits the answer into "Installed" and "Available"; the settings screen
// takes its place when it is opened.
type Home struct {
	Doc *widget.Document

	cfg    Config
	tiles  []domain.Tile // what the grid pages through (the installed apps, filtered by the query)
	result domain.SearchResult
	query  string
	page   int

	root            *widget.Panel
	settings        *settingsForm // non-nil while the settings screen is showing
	search          *widget.Input
	body            *widget.Panel
	indicator, note *widget.Label
	prev, next      *widget.Button
}

// NewHome builds the screen and loads the tiles.
func NewHome(cfg Config) *Home {
	cfg.Prefs = cfg.Prefs.Normalized()
	if cfg.Languages == nil {
		cfg.Languages = DefaultLanguages
	}
	h := &Home{cfg: cfg}
	h.Doc = widget.NewDocument(cfg.Width, cfg.Height)
	h.Doc.Keyboard = cfg.Keyboard
	h.build()
	return h
}

// Apply puts new preferences (and a language) on the screen: the grid, the icon
// size, the background and the texts are rebuilt, and the search is cleared.
func (h *Home) Apply(prefs domain.Preferences, lang string) {
	h.cfg.Prefs = prefs.Normalized()
	if lang != "" && lang != h.cfg.Tr.Lang() {
		h.cfg.Tr = NewTranslator(lang)
	}
	h.cfg.Notice = ""
	h.page, h.query, h.settings = 0, "", nil
	h.build()
	h.Doc.ResetFrame()
	h.Doc.Invalidate()
}

// paper gives a widget the page color, so it stays readable over a background picture.
func paper(w widget.Widget) widget.Widget {
	w.Elem().Style.Background = widget.Paper
	return w
}

// background is the picture chosen for the background (nil when none or unreadable).
func (h *Home) background() image.Image {
	name := h.cfg.Prefs.Image
	if name == "" || h.cfg.LoadBackground == nil {
		return nil
	}
	img, err := h.cfg.LoadBackground(name)
	if err != nil {
		return nil
	}
	return img
}

// build creates the widgets of the home screen from the current configuration.
func (h *Home) build() {
	cfg := h.cfg

	title := widget.NewLabel(cfg.Tr.T("home.title"))
	title.Style.FontSize, title.Style.Bold = widget.FontTitle, true
	title.Style.Padding = geom.XY(28, 28)
	title.ID = "home-title"
	paper(title)

	h.search = widget.NewSearchInput(cfg.Tr.T("search.placeholder"))
	h.search.ID = "home-search"
	h.search.Style.FlexGrow = 1
	h.search.OnChange = func(*widget.Event) { h.Search(h.search.Value) }
	h.search.OnSubmit = h.search.OnChange
	searchBar := widget.NewRow(0, h.search)
	searchBar.Style.Padding = geom.XY(0, 28)

	h.body = widget.NewColumn(0)
	h.body.Style.FlexGrow, h.body.Style.Padding = 1, geom.XY(12, 28)
	h.body.ID = "home-body"

	h.note = widget.NewLabel(cfg.Notice)
	h.note.Style.FontSize, h.note.Style.Color, h.note.Style.FlexGrow = widget.FontSmall, widget.Muted, 1
	h.indicator = widget.NewLabel("")
	h.indicator.Style.FontSize, h.indicator.Style.Bold = widget.FontSmall, true
	h.indicator.Style.TextAlign, h.indicator.Style.Width = gfx.AlignCenter, widget.Px(110)
	h.prev, h.next = widget.NewIconButton(gfx.IconChevronLeft), widget.NewIconButton(gfx.IconChevronRight)
	h.prev.ID, h.next.ID = "home-prev", "home-next"
	h.prev.OnClick = func(*widget.Event) { h.PrevPage() }
	h.next.OnClick = func(*widget.Event) { h.NextPage() }
	settings := widget.NewButton(cfg.Tr.T("home.settings"))
	settings.ID = "home-settings"
	settings.OnClick = func(*widget.Event) { h.OpenSettings() }

	footer := widget.NewRow(12, h.note, settings, h.prev, h.indicator, h.next)
	footer.Style.Padding = geom.XY(14, 28)
	paper(footer)

	root := widget.NewColumn(0, title, searchBar, h.body, footer)
	root.Style.Background = widget.Paper
	if bg := h.background(); bg != nil {
		root.Style.BackgroundImage = bg
	}
	h.root = root
	h.Doc.SetRoot(root)

	h.Refresh()
}

// Refresh reloads the tiles from the registry and shows the current page.
func (h *Home) Refresh() {
	if h.query == "" {
		h.tiles, h.result = h.cfg.Home.Tiles(h.cfg.Tr.Lang()), domain.SearchResult{}
	} else {
		h.result = application.Search{
			Apps: h.cfg.Home.Apps, Catalog: h.cfg.Catalog, SelfID: h.cfg.Home.SelfID,
		}.Run(h.query, h.cfg.Tr.Lang())
		h.tiles = h.result.Installed
	}
	h.show()
}

// Search filters the screen by query (an empty one shows the whole home again).
func (h *Home) Search(query string) {
	query = strings.TrimSpace(query)
	if query == h.query {
		return
	}
	h.query, h.page = query, 0
	h.Refresh()
}

// Query is what the screen is filtered by ("" when it is not searching).
func (h *Home) Query() string { return h.query }

// grid is the layout of the current page: searching leaves room for "Available".
func (h *Home) grid() domain.Grid {
	g := h.cfg.Prefs.Grid
	if h.query != "" && g.Rows > searchRows {
		g.Rows = searchRows
	}
	return g
}

// Page is the current page (0-based) and Pages how many there are.
func (h *Home) Page() int  { return h.page }
func (h *Home) Pages() int { return h.grid().Pages(len(h.tiles)) }

// PrevPage and NextPage change the page (they stop at the ends).
func (h *Home) PrevPage() { h.goTo(h.page - 1) }
func (h *Home) NextPage() { h.goTo(h.page + 1) }

func (h *Home) goTo(page int) {
	page = domain.ClampPage(page, h.Pages())
	if page == h.page {
		return
	}
	h.page = page
	h.show()
}

// show builds the current page of tiles.
func (h *Home) show() {
	h.page = domain.ClampPage(h.page, h.Pages())
	h.body.ClearChildren()

	searching := h.query != ""
	if searching {
		h.body.AppendChild(h.section("search.installed"))
	}
	tiles := h.grid().Page(h.tiles, h.page)
	if len(tiles) == 0 {
		h.body.AppendChild(h.muted("home-empty", h.emptyText()))
	} else {
		cols := make([]widget.Track, h.grid().Columns)
		for i := range cols {
			cols[i] = widget.FrTrack(1)
		}
		cells := make([]widget.Widget, len(tiles))
		for i, t := range tiles {
			cells[i] = h.tile(t)
		}
		grid := widget.NewGrid(16, cols, cells...)
		grid.ID = "home-grid"
		h.body.AppendChild(grid)
	}
	if searching {
		h.showAvailable()
	}

	pages := h.Pages()
	h.indicator.Text = h.cfg.Tr.T("home.page", h.page+1, pages)
	h.prev.Disabled, h.next.Disabled = h.page == 0, h.page >= pages-1
	h.Doc.Invalidate()
}

// tile is the cell of one app: its icon and title, tappable as a whole.
func (h *Home) tile(t domain.Tile) widget.Widget {
	var icon image.Image
	if h.cfg.IconFor != nil {
		icon = h.cfg.IconFor(t)
	}
	img := widget.NewImage(icon)
	img.Style.Height = widget.Px(float64(h.cfg.Prefs.IconSize))

	title := widget.NewLabel(t.Title)
	title.Style.FontSize, title.Style.TextAlign = widget.FontSmall, gfx.AlignCenter

	cell := widget.NewColumn(8, img, title)
	cell.ID = "app-" + t.ID
	cell.Style.Padding = geom.All(12)
	cell.Style.BorderColor, cell.Style.BorderWidth, cell.Style.BorderRadius = widget.Mid, 2, widget.Radius
	paper(cell)
	cell.OnClick = func(*widget.Event) { h.open(t) }
	return cell
}

// open starts the app; the message tells why when it could not be started.
func (h *Home) open(t domain.Tile) {
	if err := h.cfg.Home.Open(t); err != nil {
		h.say(h.cfg.Tr.T("home.open.failed", t.Title, err.Error()))
	}
}

func (h *Home) say(text string) {
	h.note.Text = text
	h.Doc.Invalidate()
}

// emptyText is what an empty grid says: no apps at all, or no match.
func (h *Home) emptyText() string {
	if h.query != "" {
		return h.cfg.Tr.T("search.none", h.query)
	}
	return h.cfg.Tr.T("home.empty")
}

func (h *Home) muted(id, text string) widget.Widget {
	l := widget.NewLabel(text)
	l.ID, l.Style.Color = id, widget.Muted
	return paper(l)
}

// section is the heading of a group of results ("-- Installed --").
func (h *Home) section(key string) widget.Widget {
	l := widget.NewLabel(h.cfg.Tr.T(key))
	l.ID = key
	l.Style.FontSize, l.Style.Bold, l.Style.Padding = widget.FontSmall, true, geom.XY(8, 12)
	return paper(l)
}

// showAvailable lists the apps of the catalog that match and are not installed.
func (h *Home) showAvailable() {
	h.body.AppendChild(h.section("search.available"))
	switch {
	case h.result.CatalogErr == domain.ErrNoCatalog:
		h.body.AppendChild(h.muted("search-nocatalog", h.cfg.Tr.T("search.nocatalog")))
	case h.result.CatalogErr != nil:
		h.body.AppendChild(h.muted("search-offline", h.cfg.Tr.T("search.offline", h.result.CatalogErr.Error())))
	case len(h.result.Available) == 0:
		h.body.AppendChild(h.muted("search-none-available", h.cfg.Tr.T("search.none", h.query)))
	default:
		for i, l := range h.result.Available {
			if i == availableMax {
				h.body.AppendChild(h.muted("search-more", h.cfg.Tr.T("search.more", len(h.result.Available)-availableMax)))
				break
			}
			h.body.AppendChild(h.listing(l))
		}
	}
}

// listing is the row of one app of the catalog.
func (h *Home) listing(l domain.Listing) widget.Widget {
	title := widget.NewLabel(l.Title)
	title.Style.FlexGrow = 1
	version := widget.NewLabel("v" + l.Version)
	version.Style.FontSize, version.Style.Color = widget.FontSmall, widget.Muted
	children := []widget.Widget{title, version}
	if h.cfg.ListingIcon != nil {
		if img := h.cfg.ListingIcon(l); img != nil {
			icon := widget.NewImage(img)
			icon.Style.Width, icon.Style.Height = widget.Px(listingIconSize), widget.Px(listingIconSize)
			children = append([]widget.Widget{icon}, children...)
		}
	}
	row := widget.NewRow(12, children...)
	row.ID = "avail-" + l.ID
	row.Style.Padding = geom.XY(14, 20)
	row.Style.BorderColor, row.Style.BorderWidth, row.Style.BorderRadius = widget.Mid, 2, widget.Radius
	paper(row)
	if h.cfg.Installer != nil {
		row.OnClick = func(*widget.Event) { h.install(l) }
	}
	return row
}

// install downloads and installs an app of the catalog. The download blocks, so
// the screen says so first; afterwards the lists are rebuilt and the app moves
// from Available to Installed.
func (h *Home) install(l domain.Listing) {
	h.say(h.cfg.Tr.T("install.downloading", l.Title))
	if h.cfg.Flush != nil {
		h.cfg.Flush()
	}
	if err := h.cfg.Installer.Install(l); err != nil {
		h.say(h.cfg.Tr.T("install.failed", l.Title, err.Error()))
		return
	}
	h.Refresh()
	h.say(h.cfg.Tr.T("install.done", l.Title))
}
