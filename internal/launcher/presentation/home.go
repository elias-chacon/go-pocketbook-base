// Package presentation draws the Launcher: the screens, built from the toolkit's
// widgets, over the launcher's use cases. It holds no rules of its own.
package presentation

import (
	"embed"
	"image"
	"strings"
	"sync"
	"time"

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
	// Version is the version of the Launcher and Repository the "owner/name" of the repository that
	// publishes the releases: both are shown in About. SystemInfo gives the lines of the system
	// information that only the reader knows (its model, its storage...); each key is translated as
	// "system.<key>". All optional.
	Version, Repository string
	SystemInfo          func() []Fact
	// Startup is what the Launcher does when it opens (clean the home screen, bring the catalog of
	// the apps of the release): run in the background when the screen opens, and again when the
	// language changes. A failure is silent: the apps already shown stay. No tasks: nothing happens.
	Startup application.Startup
	// CachedCatalog is the last catalog kept, in lang, to show the apps that are not installed at
	// once and offline (nil: they show after the first answer of Catalog).
	CachedCatalog func(lang string) []domain.Listing
	// After runs f on the event loop later (platform Env.After); the catalog is fetched in the
	// background and polled with it. Nil (tests) fetches in one go.
	After func(d time.Duration, f func())
	// Progress is how far the background fetch of the catalog (and its icons) got; the home shows a
	// bar while it runs (nil: no bar, a fetch is silent).
	Progress *domain.Progress
	// InstallProgress is how far the install running now is (the Installer advances it; nil: the
	// window of the install shows no movement).
	InstallProgress *domain.Progress
	// Installer installs (or updates) an app of the catalog (nil: it cannot be done).
	Installer domain.Installer
	// Uninstaller removes an app and all its data (nil: the Remove action is not offered).
	Uninstaller domain.Uninstaller
	// InstalledSum is the SHA-256 of the installed file of an app ("" when unknown): an app whose
	// sum differs from the catalog AppSHA256 can be updated.
	InstalledSum func(appID string) string
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

// How the screen waits for the catalog fetched in the background (see Config.After).
const (
	catalogPoll  = 2 * time.Second
	catalogPolls = 90
	// barPoll is how often the bar is looked at while the catalog comes; the wait is the same
	// catalogPoll * catalogPolls.
	barPoll = 400 * time.Millisecond
)

// Home is the main screen: a paged grid of every app of the ecosystem (the installed ones and the
// ones of the catalog that are not installed, a tap on those installs them), a search field and a
// long press that manages an app; the settings screen takes its place when it is opened.
type Home struct {
	Doc *widget.Document

	cfg      Config
	listings []domain.Listing // what the catalog said last
	all      []domain.Tile    // every app, hidden ones too
	tiles    []domain.Tile    // what the grid pages through: the visible apps that match the query
	query    string
	page     int
	loading  bool // the catalog is being fetched: the bar shows
	failed   bool // the last fetch failed

	root            *widget.Panel
	settings        *settingsForm // non-nil while the settings screen is showing
	search          *widget.Input
	body            *widget.Panel
	indicator, note *widget.Label
	bar             *widget.Progress
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
	if cfg.CachedCatalog != nil {
		h.listings = cfg.CachedCatalog(cfg.Tr.Lang())
	}
	h.build()
	h.runStartup()
	return h
}

// Apply puts new preferences (and a language) on the screen: the grid, the icon
// size, the background and the texts are rebuilt, and the search is cleared.
func (h *Home) Apply(prefs domain.Preferences, lang string) {
	h.cfg.Prefs = prefs.Normalized()
	changed := lang != "" && lang != h.cfg.Tr.Lang()
	if changed {
		h.cfg.Tr = NewTranslator(lang)
		if h.cfg.CachedCatalog != nil {
			h.listings = h.cfg.CachedCatalog(lang) // the titles of the apps are in the new language
		}
	}
	h.cfg.Notice = ""
	h.page, h.query, h.settings = 0, "", nil
	h.build()
	h.Doc.ResetFrame()
	h.Doc.Invalidate()
	if changed {
		h.runStartup()
	}
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
	h.note.Style.FontSize, h.note.Style.Color = widget.FontSmall, widget.Muted
	h.note.Hidden = cfg.Notice == ""
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

	// The message has a line of its own, wrapping: a long one must not push the buttons off the screen.
	push := widget.NewPanel()
	push.Style.FlexGrow = 1
	buttons := widget.NewRow(12, push, settings, h.prev, h.indicator, h.next)
	footer := widget.NewColumn(10, h.note, buttons)
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

// Refresh rebuilds the list of apps (the installed ones, the ones of the catalog, minus the hidden)
// and shows the current page.
func (h *Home) Refresh() {
	h.all = h.cfg.Home.All(h.listings, h.cfg.Tr.Lang())
	h.tiles = domain.FilterTiles(domain.Visible(h.all, h.cfg.Prefs.HiddenApps), h.query)
	h.show()
}

// startResult is the outcome of the startup running in the background, handed to the event loop.
type startResult struct {
	mu   sync.Mutex
	done bool
	out  application.Outcome
}

func (r *startResult) set(out application.Outcome) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.out, r.done = out, true
}

func (r *startResult) get() (application.Outcome, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.out, r.done
}

// runStartup runs the startup tasks in the background (a goroutine does the work, the event loop
// polls and draws: no widget is touched off the loop).
func (h *Home) runStartup() {
	if len(h.cfg.Startup.Tasks) == 0 {
		return
	}
	lang := h.cfg.Tr.Lang()
	if h.cfg.After == nil {
		h.startupDone(h.cfg.Startup.Execute(lang, h.cfg.Progress))
		return
	}
	res := &startResult{}
	go func() { res.set(h.cfg.Startup.Execute(lang, h.cfg.Progress)) }()
	h.loading = true
	h.Refresh()                                                        // the bar is on the screen from the first frame
	h.cfg.After(h.pollEvery(), func() { h.pollStartup(res, lang, 0) }) // the first look is on the loop too
}

// pollEvery is the wait between two looks at the startup: short when a bar is moving.
func (h *Home) pollEvery() time.Duration {
	if h.cfg.Progress != nil {
		return barPoll
	}
	return catalogPoll
}

func (h *Home) pollStartup(res *startResult, lang string, n int) {
	out, done := res.get()
	switch {
	case done && lang == h.cfg.Tr.Lang():
		h.startupDone(out)
	case !done && n < catalogPolls*int(catalogPoll/h.pollEvery()):
		h.moveBar()
		h.cfg.After(h.pollEvery(), func() { h.pollStartup(res, lang, n+1) })
	}
}

// startupDone takes what the startup found: the catalog (a failure leaves what is shown as it is)
// and the home screen entries it took out. While the settings screen is open the list is kept and
// drawn when the home comes back.
func (h *Home) startupDone(out application.Outcome) {
	h.loading = false
	err := out.Errors[application.TaskCatalog]
	h.failed = err != nil
	if err == nil && out.Listings != nil {
		h.listings = out.Listings
	}
	if n := len(out.Removed); n > 0 {
		h.say(h.cfg.Tr.T("startup.pruned", n))
	}
	if h.settings == nil {
		h.Refresh()
	}
}

// barPercent is how far the startup is, 0..100.
func (h *Home) barPercent() int {
	f, _ := h.cfg.Progress.State()
	return int(f*100 + 0.5)
}

// barText is the title of the bar: what the startup is doing now, in real time, and how far it is.
func (h *Home) barText(pct int) string {
	phase, detail := h.cfg.Progress.Phase()
	switch phase {
	case application.PhaseRemoving:
		return h.cfg.Tr.T("startup.removing", detail, pct)
	case application.PhaseCatalog:
		return h.cfg.Tr.T("startup.catalog", pct)
	}
	return h.cfg.Tr.T("startup.checking", pct)
}

// moveBar puts the progress of the fetch on the bar (and repaints it only when it moved).
func (h *Home) moveBar() {
	if !h.loading || h.bar == nil || h.settings != nil {
		return
	}
	pct := h.barPercent()
	if text := h.barText(pct); h.bar.Value == float64(pct) && h.bar.Text == text {
		return
	}
	h.bar.Value, h.bar.Text = float64(pct), h.barText(pct)
	h.Doc.Invalidate()
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

// grid is the layout of the current page.
func (h *Home) grid() domain.Grid { return h.cfg.Prefs.Grid }

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

	if h.loading {
		h.bar = widget.NewProgress(0)
		h.bar.ID, h.bar.Style.Width = "home-loading", widget.Pct(100)
		pct := h.barPercent()
		h.bar.Value, h.bar.Text = float64(pct), h.barText(pct)
		h.body.AppendChild(paper(h.bar))
	}
	tiles := h.grid().Page(h.tiles, h.page)
	if len(tiles) == 0 {
		if !h.loading {
			h.body.AppendChild(h.muted("home-empty", h.emptyText()))
		}
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
	pages := h.Pages()
	h.indicator.Text = h.cfg.Tr.T("home.page", h.page+1, pages)
	h.prev.Disabled, h.next.Disabled = h.page == 0, h.page >= pages-1
	h.Doc.Invalidate()
}

// tile is the cell of one app: its icon and title, tappable as a whole.
func (h *Home) tile(t domain.Tile) widget.Widget {
	var icon image.Image
	if !t.Installed && t.Listing != nil && h.cfg.ListingIcon != nil {
		icon = h.cfg.ListingIcon(*t.Listing)
	}
	if icon == nil && h.cfg.IconFor != nil {
		icon = h.cfg.IconFor(t)
	}
	img := widget.NewImage(icon)
	img.Style.Height = widget.Px(float64(h.cfg.Prefs.IconSize))

	title := widget.NewLabel(t.Title)
	title.Style.FontSize, title.Style.TextAlign = widget.FontSmall, gfx.AlignCenter

	cell := widget.NewColumn(8, img, title)
	if !t.Installed {
		hint := widget.NewLabel(h.cfg.Tr.T("home.tile.install"))
		hint.Style.FontSize, hint.Style.Color, hint.Style.TextAlign = widget.FontSmall, widget.Muted, gfx.AlignCenter
		cell.AppendChild(hint)
	}
	cell.ID = "app-" + t.ID
	cell.Style.Padding = geom.All(12)
	cell.Style.BorderColor, cell.Style.BorderWidth, cell.Style.BorderRadius = widget.Mid, 2, widget.Radius
	paper(cell)
	cell.OnClick = func(*widget.Event) { h.activate(t) }
	cell.OnLongPress = func(*widget.Event) { h.Manage(t, nil) }
	return cell
}

// activate is a tap on a tile: an installed app starts, one that is not installed is installed.
func (h *Home) activate(t domain.Tile) {
	if t.Installed {
		h.open(t)
		return
	}
	if t.Listing != nil && h.cfg.Installer != nil {
		h.install(*t.Listing, false, nil)
	}
}

// open starts the app; the message tells why when it could not be started.
func (h *Home) open(t domain.Tile) {
	if err := h.cfg.Home.Open(t); err != nil {
		h.say(h.cfg.Tr.T("home.open.failed", t.Title, err.Error()))
	}
}

func (h *Home) say(text string) {
	h.note.Text, h.note.Hidden = text, text == ""
	h.Doc.Invalidate()
}

// emptyText is what an empty grid says: no apps at all, or no match.
func (h *Home) emptyText() string {
	if h.query != "" {
		return h.cfg.Tr.T("search.none", h.query)
	}
	if h.failed {
		return h.cfg.Tr.T("home.failed")
	}
	return h.cfg.Tr.T("home.empty")
}

func (h *Home) muted(id, text string) widget.Widget {
	l := widget.NewLabel(text)
	l.ID, l.Style.Color = id, widget.Muted
	return paper(l)
}
