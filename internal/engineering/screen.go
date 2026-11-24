package engineering

import (
	"embed"
	"fmt"
	"image"
	"image/color"
	"strings"
	"time"

	"go-pocketbook-base/internal/localization/i18n"
	"go-pocketbook-base/internal/toolkit/gfx"
	"go-pocketbook-base/internal/toolkit/widget"
)

//go:embed locales/*.json
var locales embed.FS

// NewTranslator loads the engineering app's texts (pt-BR first, English fallback) in lang.
func NewTranslator(lang string) *i18n.Bundle {
	tr := i18n.New()
	if err := tr.AddFS(locales, "locales"); err != nil {
		panic(err) // embedded files: an error is a programming mistake
	}
	tr.SetLang(lang)
	return tr
}

// Config is everything the screen needs.
type Config struct {
	Width, Height int
	// Doc is the document to draw on (a new one when nil).
	Doc *widget.Document
	// Chrome is the standard top bar (the app name with About and Exit) over the menu.
	Chrome widget.AppChrome
	Tr     *i18n.Bundle
	// Panel is the reader's display (nil: the panel tests only say so).
	Panel Panel
	// Rec receives every measure (nil: nothing is recorded).
	Rec *Recorder
	// After runs f once d has passed, on the event loop (a long test goes in small steps).
	After func(d time.Duration, f func())
	// Flush paints the pending changes of the document now.
	Flush func()
	// Watch registers the function to call after every repaint of the platform (nil: none).
	Watch func(f func(Paint))
	// Now is the clock (time.Now when nil).
	Now func() time.Time
	// Facts are the lines of the system page that only the reader knows.
	Facts func() []Fact
	// LogLines returns the last n lines of the log; ClearLog empties it; LogPath is where it is.
	LogLines func(n int) []string
	ClearLog func()
	LogPath  string
}

// The pages of the app.
const (
	pageMenu     = "menu"
	pagePatterns = "patterns"
	pageTouch    = "touch"
	pageUpdate   = "update"
	pageScroll   = "scroll"
	pageSystem   = "system"
	pageLog      = "log"
)

// Screen is the engineering app: a menu of tests.
type Screen struct {
	cfg  Config
	doc  *widget.Document
	page string
	mode Mode // how the panel is refreshed by the tests that let it be chosen

	view *patternView // the full-screen picture being shown (nil: none)

	pad    *TouchPad
	scroll *widget.Scroll
	note   *widget.Label // the result line of the scroll and the update tests
	table  *widget.Table // the results of the update benchmark
	auto   *autoScroll
	bench  *benchmark

	paint Paint // the last repaint of the platform
}

// NewScreen builds the app on its menu and records the start.
func NewScreen(cfg Config) *Screen {
	if cfg.Doc == nil {
		cfg.Doc = widget.NewDocument(cfg.Width, cfg.Height)
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	s := &Screen{cfg: cfg, doc: cfg.Doc, mode: Partial}
	if cfg.Watch != nil {
		cfg.Watch(s.onPaint)
	}
	s.cfg.Rec.Event("start", s.factFields()...)
	s.showMenu()
	return s
}

func (s *Screen) now() time.Time { return s.cfg.Now() }

func (s *Screen) t(key string, args ...interface{}) string { return s.cfg.Tr.T(key, args...) }

// factFields are the system facts as fields of a log event.
func (s *Screen) factFields() []interface{} {
	var out []interface{}
	if s.cfg.Panel != nil {
		in := s.cfg.Panel.Info()
		out = append(out, "model", in.Model, "depth", in.Depth, "waveform", in.Waveform, "canvas", in.Canvas)
	}
	out = append(out, "width", s.cfg.Width, "height", s.cfg.Height)
	if s.cfg.Facts != nil {
		for _, f := range s.cfg.Facts() {
			out = append(out, f.Key, f.Value)
		}
	}
	return out
}

func (s *Screen) button(id, text string, onClick func()) *widget.Button {
	b := widget.NewButton(text)
	b.ID = id
	b.OnClick = func(*widget.Event) { onClick() }
	return b
}

// show puts the page on the document: the title, the body and a Menu button, under the chrome.
func (s *Screen) show(page, title string, body ...widget.Widget) {
	s.leave()
	s.page = page
	head := widget.NewLabel(title)
	head.Style.Bold, head.Style.FontSize = true, widget.FontTitle
	items := []widget.Widget{head}
	items = append(items, body...)
	if page != pageMenu {
		items = append(items, s.button("eng-menu", s.t("eng.menu"), s.showMenu))
	}
	root := widget.NewColumn(16, items...)
	root.Style.Padding = widgetPad()
	root.Style.AlignItems = widget.AlignStretch
	s.doc.SetRoot(s.cfg.Chrome.Wrap(s.doc, root))
	s.doc.ResetFrame()
	s.doc.Invalidate()
}

// leave undoes what the page that is closing did to the panel and the log.
func (s *Screen) leave() {
	if s.auto != nil {
		s.auto.stop()
		s.auto = nil
	}
	s.bench = nil
	if s.view != nil || s.mode != Partial {
		s.mode, s.view = Partial, nil
		if s.cfg.Panel != nil {
			s.cfg.Panel.SetMode(Partial)
			s.cfg.Panel.Settle()
		}
	}
	s.cfg.Rec.Flush()
}

// Back leaves the page for the menu; false when the menu is already there (the app may exit).
func (s *Screen) Back() bool {
	if s.page == pageMenu {
		return false
	}
	s.showMenu()
	return true
}

func (s *Screen) showMenu() {
	sub := widget.NewLabel(s.t("eng.subtitle"))
	sub.Style.Color = widget.Muted
	entries := []struct {
		id, key string
		open    func()
	}{
		{"eng-colors", "eng.colors", func() { s.showPatterns("colors", ColorPatterns()) }},
		{"eng-screen", "eng.screen", func() { s.showPatterns("screen", ScreenPatterns()) }},
		{"eng-touch", "eng.touch", s.showTouch},
		{"eng-update", "eng.update", s.showUpdate},
		{"eng-scroll", "eng.scroll", s.showScroll},
		{"eng-system", "eng.system", s.showSystem},
		{"eng-log", "eng.log", s.showLog},
	}
	body := []widget.Widget{sub}
	for _, e := range entries {
		body = append(body, s.button(e.id, s.t(e.key), e.open))
	}
	s.show(pageMenu, s.t("eng.title"), body...)
}

// ---- full-screen patterns ----

// patternView is a full-screen picture and where it came from. The app paints it itself (Raw), so
// the screen's document only holds the areas that answer touches.
type patternView struct {
	group string
	list  []Pattern
	idx   int
	img   *image.RGBA // the picture drawn for idx, with the caption stamped on it
}

// showPatterns shows the pictures of a group full screen: a touch on the top left goes back, on the
// top right changes the update mode, on the left third goes to the previous picture and on the rest
// to the next one.
func (s *Screen) showPatterns(group string, list []Pattern) {
	s.leave()
	s.page, s.view = pagePatterns, &patternView{group: group, list: list}
	zone := func(id string, grow float64, f func()) *widget.Panel {
		z := widget.NewPanel()
		z.ID, z.Style.FlexGrow = id, grow
		z.OnClick = func(*widget.Event) { f() }
		return z
	}
	top := widget.NewRow(0, zone("pat-back", 1, func() { s.showMenu() }), zone("pat-mode", 1, s.cycleMode))
	top.Style.Height = widget.Px(200)
	top.Style.AlignItems = widget.AlignStretch
	bottom := widget.NewRow(0, zone("pat-prev", 1, func() { s.step(-1) }), zone("pat-next", 2, func() { s.step(1) }))
	bottom.Style.FlexGrow = 1
	bottom.Style.AlignItems = widget.AlignStretch
	root := widget.NewColumn(0, top, bottom)
	root.Style.AlignItems = widget.AlignStretch
	s.doc.SetRoot(root)
	s.doc.Layout() // the platform does not lay out a screen the app paints itself, and the zones need their place to take touches
	s.doc.ResetFrame()
	s.doc.Invalidate()
}

func (s *Screen) step(d int) {
	v := s.view
	if v == nil {
		return
	}
	v.idx = (v.idx + d + len(v.list)) % len(v.list)
	v.img = nil
	s.doc.Invalidate()
}

func (s *Screen) cycleMode() {
	s.mode = s.mode.Next()
	if s.view != nil {
		s.doc.Invalidate()
	}
}

// Raw paints the panel when a picture is being shown: the platform then leaves the screen alone.
func (s *Screen) Raw(p Panel) bool {
	v := s.view
	if v == nil {
		return false
	}
	pat := v.list[v.idx]
	gen := time.Duration(0)
	if v.img == nil {
		t0 := s.now()
		v.img = pat.Draw(s.cfg.Width, s.cfg.Height)
		gen = s.now().Sub(t0)
	}
	s.stamp(v.img, fmt.Sprintf("%d/%d  %s  [%s]", v.idx+1, len(v.list), s.t("pat."+pat.ID), s.mode))
	t1 := s.now()
	p.DrawColor(v.img, image.Rect(0, 0, s.cfg.Width, s.cfg.Height), s.mode)
	draw := s.now().Sub(t1)
	t2 := s.now()
	p.Settle()
	s.cfg.Rec.Event("pattern", "group", v.group, "id", pat.ID, "mode", s.mode.String(),
		"gen_ms", Millis(gen), "draw_ms", Millis(draw), "settle_ms", Millis(s.now().Sub(t2)))
	s.cfg.Rec.Flush()
	return true
}

// stamp writes a caption in a white box at the bottom left of the picture.
func (s *Screen) stamp(img *image.RGBA, text string) {
	const w, h = 760, 80
	c := gfx.NewCanvas(w, h, 1)
	c.Clear(widget.Paper)
	c.StrokeRoundRect(geomRect(2, 2, w-4, h-4), 10, 3, widget.Ink)
	c.Text(text, geomRect(16, 12, w-32, h-24), gfx.TextStyle{Size: widget.FontBody, Bold: true, Color: widget.Ink, Align: gfx.AlignStart})
	g := c.Image()
	x0, y0 := 40, s.cfg.Height-h-40
	if y0 < 0 || x0+w > s.cfg.Width {
		return
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := g.GrayAt(x, y).Y
			img.SetRGBA(x0+x, y0+y, color.RGBA{v, v, v, 255})
		}
	}
}

// ---- touch ----

func (s *Screen) showTouch() {
	s.pad = NewTouchPad(s.t("touch.hint"))
	s.pad.ID = "touch-pad"
	s.pad.Now = s.cfg.Now
	s.pad.OnEvent = func(phase string, x, y float64, gap time.Duration) {
		s.cfg.Rec.Event("touch", "phase", phase, "x", int(x), "y", int(y), "gap_ms", Millis(gap))
		if phase == "up" {
			s.cfg.Rec.Event("touch_stroke", "events", s.pad.Events(), "rate", s.pad.Rate(), "max_gap_ms", Millis(s.pad.MaxGap()))
			s.cfg.Rec.Flush()
		}
	}
	clear := s.button("touch-clear", s.t("touch.clear"), func() { s.pad.Reset() })
	s.show(pageTouch, s.t("eng.touch"), s.pad, clear)
}

// ---- system ----

func (s *Screen) showSystem() {
	facts := s.systemFacts()
	table := widget.NewTable(len(facts)+1,
		widget.TableColumn{Title: s.t("sys.col.item"), Flex: 2},
		widget.TableColumn{Title: s.t("sys.col.value"), Flex: 3})
	table.ID = "system-table"
	for _, f := range facts {
		table.Rows = append(table.Rows, []string{s.t("sys." + f.Key), f.Value})
	}
	s.cfg.Rec.Event("system", s.factFields()...)
	s.show(pageSystem, s.t("eng.system"), table)
}

func (s *Screen) systemFacts() []Fact {
	var out []Fact
	if s.cfg.Panel != nil {
		in := s.cfg.Panel.Info()
		out = append(out, Fact{"model", in.Model}, Fact{"depth", fmt.Sprintf("%d bit", in.Depth)},
			Fact{"waveform", in.Waveform}, Fact{"canvas", in.Canvas})
	}
	out = append(out, Fact{"screen", fmt.Sprintf("%d x %d", s.cfg.Width, s.cfg.Height)})
	if s.cfg.Facts != nil {
		out = append(out, s.cfg.Facts()...)
	}
	return out
}

// ---- log ----

func (s *Screen) showLog() {
	s.cfg.Rec.Flush()
	var lines []string
	if s.cfg.LogLines != nil {
		lines = s.cfg.LogLines(40)
	}
	text := strings.Join(lines, "\n")
	if text == "" {
		text = s.t("log.empty")
	}
	path := widget.NewLabel(s.t("log.path", s.cfg.LogPath))
	path.Style.FontSize, path.Style.Color = widget.FontSmall, widget.Muted
	body := widget.NewLabel(text)
	body.ID = "log-text"
	body.Style.FontSize = 20
	scroll := widget.NewScroll(body)
	scroll.ID = "log-scroll"
	scroll.Style.FlexGrow = 1
	clear := s.button("log-clear", s.t("log.clear"), func() {
		if s.cfg.ClearLog != nil {
			s.cfg.ClearLog()
		}
		s.showLog()
	})
	s.show(pageLog, s.t("eng.log"), path, scroll, clear)
}
