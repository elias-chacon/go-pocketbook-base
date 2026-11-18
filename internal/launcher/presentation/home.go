// Package presentation draws the Launcher: the screens, built from the toolkit's
// widgets, over the launcher's use cases. It holds no rules of its own.
package presentation

import (
	"embed"
	"image"

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

// DefaultIconSize is the edge of an app icon, in design pixels.
const DefaultIconSize = 150.0

// Config is everything the home screen needs.
type Config struct {
	Width, Height int
	Keyboard      widget.Keyboard
	Tr            *i18n.Bundle
	Home          application.Home
	Grid          domain.Grid
	IconSize      float64 // 0 = DefaultIconSize
	// IconFor loads the icon of a tile (nil results draw nothing); the composition
	// root decides where icons come from and what to show when one is missing.
	IconFor func(t domain.Tile) image.Image
	Notice  string
}

// Home is the main screen: a paged grid of the installed apps.
type Home struct {
	Doc *widget.Document

	cfg   Config
	tiles []domain.Tile
	page  int

	body            *widget.Panel
	indicator, note *widget.Label
	prev, next      *widget.Button
}

// NewHome builds the screen and loads the tiles.
func NewHome(cfg Config) *Home {
	cfg.Grid = cfg.Grid.Normalized()
	if cfg.IconSize == 0 {
		cfg.IconSize = DefaultIconSize
	}
	h := &Home{cfg: cfg}
	h.Doc = widget.NewDocument(cfg.Width, cfg.Height)
	h.Doc.Keyboard = cfg.Keyboard

	title := widget.NewLabel(cfg.Tr.T("home.title"))
	title.Style.FontSize, title.Style.Bold = widget.FontTitle, true
	title.Style.Padding = geom.XY(28, 28)
	title.ID = "home-title"

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

	footer := widget.NewRow(12, h.note, h.prev, h.indicator, h.next)
	footer.Style.Padding = geom.XY(14, 28)

	root := widget.NewColumn(0, title, h.body, footer)
	root.Style.Background = widget.Paper
	h.Doc.SetRoot(root)

	h.Refresh()
	return h
}

// Refresh reloads the tiles from the registry and shows the current page.
func (h *Home) Refresh() {
	h.tiles = h.cfg.Home.Tiles(h.cfg.Tr.Lang())
	h.show()
}

// Page is the current page (0-based) and Pages how many there are.
func (h *Home) Page() int  { return h.page }
func (h *Home) Pages() int { return h.cfg.Grid.Pages(len(h.tiles)) }

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

	tiles := h.cfg.Grid.Page(h.tiles, h.page)
	if len(tiles) == 0 {
		empty := widget.NewLabel(h.cfg.Tr.T("home.empty"))
		empty.Style.Color = widget.Muted
		empty.ID = "home-empty"
		h.body.AppendChild(empty)
	} else {
		cols := make([]widget.Track, h.cfg.Grid.Columns)
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
	if h.cfg.IconFor != nil {
		icon = h.cfg.IconFor(t)
	}
	img := widget.NewImage(icon)
	img.Style.Height = widget.Px(h.cfg.IconSize)

	title := widget.NewLabel(t.Title)
	title.Style.FontSize, title.Style.TextAlign = widget.FontSmall, gfx.AlignCenter

	cell := widget.NewColumn(8, img, title)
	cell.ID = "app-" + t.ID
	cell.Style.Padding = geom.All(12)
	cell.Style.BorderColor, cell.Style.BorderWidth, cell.Style.BorderRadius = widget.Mid, 2, widget.Radius
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
