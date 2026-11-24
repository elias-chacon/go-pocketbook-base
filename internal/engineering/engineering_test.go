package engineering

import (
	"bytes"
	"encoding/json"
	"image"
	"strings"
	"testing"
	"time"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/widget"
)

// fakePanel records what the tests draw.
type fakePanel struct {
	draws   []image.Rectangle
	modes   []Mode
	set     []Mode
	settles int
	waits   int
}

func (f *fakePanel) Info() Info {
	return Info{Model: "PocketBook 741", Depth: 24, Waveform: "a.wbf", Canvas: "canvas"}
}
func (f *fakePanel) DrawGray(_ *image.Gray, r image.Rectangle, m Mode)  { f.note(r, m) }
func (f *fakePanel) DrawColor(_ *image.RGBA, r image.Rectangle, m Mode) { f.note(r, m) }
func (f *fakePanel) note(r image.Rectangle, m Mode) {
	f.draws, f.modes = append(f.draws, r), append(f.modes, m)
}
func (f *fakePanel) Wait()          { f.waits++ }
func (f *fakePanel) SetMode(m Mode) { f.set = append(f.set, m) }
func (f *fakePanel) Settle()        { f.settles++ }

// rig is a screen with a fake panel, a log in memory, a clock that moves 5 ms per reading and a
// queue for what the screen asks to run later.
type rig struct {
	s     *Screen
	panel *fakePanel
	log   *bytes.Buffer
	queue []func()
	clock time.Time
	watch func(Paint) // what the screen asked to hear about every repaint
}

func newRig(t *testing.T, withPanel bool) *rig {
	t.Helper()
	r := &rig{log: &bytes.Buffer{}, clock: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)}
	now := func() time.Time { r.clock = r.clock.Add(5 * time.Millisecond); return r.clock }
	cfg := Config{
		Width: 1404, Height: 1872, Tr: NewTranslator("pt-BR"), Rec: NewRecorder(r.log, now), Now: now,
		After: func(_ time.Duration, f func()) { r.queue = append(r.queue, f) },
		Watch: func(f func(Paint)) { r.watch = f },
		Flush: func() {}, LogPath: "/mnt/ext1/applications/.pbgo/eng.log",
		Facts:    func() []Fact { return []Fact{{"version", "1.2.3"}} },
		LogLines: func(n int) []string { return strings.Split(strings.TrimSpace(r.log.String()), "\n") },
		ClearLog: func() { r.log.Reset() },
	}
	if withPanel {
		r.panel = &fakePanel{}
		cfg.Panel = r.panel
	}
	r.s = NewScreen(cfg)
	r.s.doc.Layout()
	return r
}

// drain runs what was asked to run later, until nothing is left.
func (r *rig) drain(t *testing.T) {
	t.Helper()
	for i := 0; len(r.queue) > 0; i++ {
		if i > 10000 {
			t.Fatal("the queue does not end")
		}
		f := r.queue[0]
		r.queue = r.queue[1:]
		f()
		r.s.doc.Layout()
	}
}

func (r *rig) tap(t *testing.T, id string) {
	t.Helper()
	w := r.s.doc.GetElementByID(id)
	if w == nil {
		t.Fatalf("no element %q on the page %q", id, r.s.page)
	}
	c := w.Elem().Bounds.Center()
	r.s.doc.HandlePointer(widget.PointerDown, c.X, c.Y)
	r.s.doc.HandlePointer(widget.PointerUp, c.X, c.Y)
	r.s.doc.Layout()
}

// events are the lines of the log, decoded.
func (r *rig) events(t *testing.T) []map[string]interface{} {
	t.Helper()
	r.s.cfg.Rec.Flush()
	var out []map[string]interface{}
	for _, line := range strings.Split(strings.TrimSpace(r.log.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]interface{}
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("a log line is not JSON: %q: %v", line, err)
		}
		out = append(out, m)
	}
	return out
}

func (r *rig) kinds(t *testing.T, kind string) []map[string]interface{} {
	var out []map[string]interface{}
	for _, e := range r.events(t) {
		if e["kind"] == kind {
			out = append(out, e)
		}
	}
	return out
}

func TestTheRecorderWritesOneJSONObjectPerLineAndNilRecordsNothing(t *testing.T) {
	var buf bytes.Buffer
	rec := NewRecorder(&buf, func() time.Time { return time.Date(2026, 5, 6, 7, 8, 9, 0, time.UTC) })
	rec.Event("draw", "mode", "a2", "ms", 12.5, "odd")
	rec.Event("start")
	rec.Flush()
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines: %q", buf.String())
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &m); err != nil || m["kind"] != "draw" || m["mode"] != "a2" || m["ms"] != 12.5 {
		t.Errorf("decoded %v (%v)", m, err)
	}
	if !strings.HasPrefix(lines[0], `{"t":"2026-05-06T07:08:09Z","kind":"draw","mode":"a2","ms":12.5}`[:40]) {
		t.Errorf("the time and the kind come first: %s", lines[0])
	}
	var late bytes.Buffer
	rec2 := NewRecorder(&late, nil)
	rec2.Event("bench", "ms", 1)
	if !strings.Contains(late.String(), `"kind":"bench"`) {
		t.Error("an ordinary event is written at once, so a crash does not lose it")
	}
	late.Reset()
	rec2.Event("touch", "x", 1)
	if late.Len() != 0 {
		t.Error("a touch event waits in the buffer")
	}
	rec2.Flush()
	if late.Len() == 0 {
		t.Error("and a flush writes it")
	}
	var nilRec *Recorder
	nilRec.Event("x")
	nilRec.Flush() // must not panic
}

func TestSummarizeGivesTheMeanTheWorstCaseAndTheRate(t *testing.T) {
	ms := time.Millisecond
	st := Summarize([]time.Duration{10 * ms, 20 * ms, 30 * ms, 40 * ms, 100 * ms})
	if st.Count != 5 || st.Min != 10*ms || st.Max != 100*ms || st.Mean != 40*ms || st.P95 != 100*ms {
		t.Errorf("%+v", st)
	}
	if got := st.PerSecond(); got < 24.9 || got > 25.1 {
		t.Errorf("rate %v", got)
	}
	if (Summarize(nil) != Stats{}) || Summarize(nil).PerSecond() != 0 {
		t.Error("no samples, no statistics")
	}
	if !strings.Contains(st.String(), "mean=40.0ms") {
		t.Errorf("summary %q", st.String())
	}
}

// opaque reports whether every pixel of the picture is fully opaque.
func opaque(img *image.RGBA) bool {
	for i := 3; i < len(img.Pix); i += 4 {
		if img.Pix[i] != 255 {
			return false
		}
	}
	return true
}

// varied reports whether the picture has more than one color.
func varied(img *image.RGBA) bool {
	first := img.RGBAAt(0, 0)
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			if img.RGBAAt(x, y) != first {
				return true
			}
		}
	}
	return false
}

func TestEveryPatternFillsTheScreenAndDiffersFromAFlatOne(t *testing.T) {
	const w, h = 140, 187
	flat := map[string]bool{"white": true, "black": true, "gray50": true}
	for _, p := range append(ColorPatterns(), ScreenPatterns()...) {
		img := p.Draw(w, h)
		if img.Bounds().Dx() != w || img.Bounds().Dy() != h || !opaque(img) {
			t.Errorf("%s: %v is not a full opaque picture", p.ID, img.Bounds())
		}
		if varied(img) == flat[p.ID] {
			t.Errorf("%s: varied=%v", p.ID, varied(img))
		}
	}
	bar := bars(w, h)
	if c := bar.RGBAAt(w/16, 10); c.R != 255 || c.G != 255 || c.B != 255 {
		t.Errorf("the first bar is white: %v", c)
	}
	if c := bar.RGBAAt(w*5/8+2, 10); c.R != 255 || c.G != 0 || c.B != 0 {
		t.Errorf("the sixth bar is red: %v", c)
	}
	if c := hsv(120, 1, 1); c.G != 255 || c.R != 0 || c.B != 0 {
		t.Errorf("hue 120 is green: %v", c)
	}
}

func TestTheMenuListsEveryTestAndTheModesCycle(t *testing.T) {
	r := newRig(t, true)
	for _, id := range []string{"eng-colors", "eng-screen", "eng-touch", "eng-update", "eng-scroll", "eng-system", "eng-log"} {
		if r.s.doc.GetElementByID(id) == nil {
			t.Errorf("the menu lacks %s", id)
		}
	}
	if Partial.Next() != Dynamic || Full.Next() != Partial || A2.String() != "a2" || Mode(99).String() != "unknown" {
		t.Error("the modes follow each other in a circle and have names")
	}
	if len(r.kinds(t, "start")) != 1 || r.kinds(t, "start")[0]["model"] != "PocketBook 741" {
		t.Errorf("the start is logged with the device: %v", r.kinds(t, "start"))
	}
}

func TestAPatternIsPaintedByTheAppAndTheTouchZonesNavigate(t *testing.T) {
	r := newRig(t, true)
	r.tap(t, "eng-colors")
	if r.s.view == nil || r.s.page != pagePatterns {
		t.Fatal("a picture is on")
	}
	if !r.s.Raw(r.panel) || len(r.panel.draws) != 1 || r.panel.draws[0] != image.Rect(0, 0, 1404, 1872) || r.panel.modes[0] != Partial {
		t.Fatalf("the whole screen is drawn once: %v %v", r.panel.draws, r.panel.modes)
	}
	r.tap(t, "pat-next")
	r.s.Raw(r.panel)
	r.tap(t, "pat-mode")
	r.s.Raw(r.panel)
	if r.s.view.idx != 1 || r.panel.modes[len(r.panel.modes)-1] != Dynamic {
		t.Errorf("next picture and next mode: idx %d, modes %v", r.s.view.idx, r.panel.modes)
	}
	r.tap(t, "pat-prev")
	r.tap(t, "pat-prev")
	if r.s.view.idx != len(ColorPatterns())-1 {
		t.Errorf("previous goes around: %d", r.s.view.idx)
	}
	pats := r.kinds(t, "pattern")
	if len(pats) != 3 || pats[0]["group"] != "colors" || pats[0]["id"] != "bars" || pats[1]["id"] != "primaries" || pats[2]["mode"] != "dynamic" {
		t.Errorf("every picture drawn is logged: %v", pats)
	}
	r.tap(t, "pat-back")
	if r.s.view != nil || r.s.page != pageMenu || r.panel.settles == 0 {
		t.Error("back leaves the picture, settles the panel and shows the menu")
	}
	if r.s.Raw(r.panel) {
		t.Error("on the menu the platform paints")
	}
	if got := r.panel.set[len(r.panel.set)-1]; got != Partial {
		t.Errorf("the platform's mode is back to the default: %v", got)
	}
}

func TestThePatternZonesAreLaidOutWithoutHelpFromThePlatform(t *testing.T) {
	r := newRig(t, true)
	r.s.showPatterns("colors", ColorPatterns()) // no Layout here: the platform does not lay out a screen the app paints
	for _, id := range []string{"pat-back", "pat-mode", "pat-prev", "pat-next"} {
		if w := r.s.doc.GetElementByID(id); w == nil || w.Elem().Bounds.W <= 0 || w.Elem().Bounds.H <= 0 {
			t.Errorf("%s has no place on the screen", id)
		}
	}
	if next := r.s.doc.GetElementByID("pat-next").Elem().Bounds; r.s.doc.TargetAt(next.Center()) != r.s.doc.GetElementByID("pat-next") {
		t.Error("a touch in the middle of the next zone reaches it")
	}
}

func TestTheTouchPadLogsEveryEventAndTheStroke(t *testing.T) {
	r := newRig(t, true)
	r.tap(t, "eng-touch")
	pad := r.s.pad
	r.s.doc.Layout()
	b := pad.Bounds
	x, y := b.X+50, b.Y+50
	r.s.doc.HandlePointer(widget.PointerDown, x, y)
	for i := 1; i <= 5; i++ {
		r.s.doc.HandlePointer(widget.PointerMove, x+float64(i*20), y+float64(i*10))
	}
	r.s.doc.HandlePointer(widget.PointerUp, x+100, y+50)
	if pad.Events() != 7 || pad.Rate() <= 0 || pad.MaxGap() <= 0 {
		t.Errorf("events %d rate %v gap %v", pad.Events(), pad.Rate(), pad.MaxGap())
	}
	phases := ""
	for _, e := range r.kinds(t, "touch") {
		phases += e["phase"].(string)[:1]
	}
	if phases != "dmmmmmu" {
		t.Errorf("phases %q", phases)
	}
	if st := r.kinds(t, "touch_stroke"); len(st) != 1 || st[0]["events"].(float64) != 7 {
		t.Errorf("the stroke is summarized: %v", st)
	}
	if distance(geom.Point{}, geom.Point{X: 3, Y: 4}) != 5 {
		t.Error("distance")
	}
	r.tap(t, "touch-clear")
	if pad.Events() != 0 {
		t.Error("clear forgets the stroke")
	}
	if r.s.Back() != true || r.s.Back() != false {
		t.Error("Back leaves the page once and then says the menu is already there")
	}
}

func TestTheBenchmarkTimesEveryModeAndFillsTheTable(t *testing.T) {
	r := newRig(t, true)
	r.tap(t, "eng-update")
	r.tap(t, "update-run")
	for _, img := range r.s.bench.imgs {
		if !r.s.bench.rect.In(img.Bounds()) {
			t.Fatalf("the rectangle %v is not inside its picture %v: the panel would read past the end", r.s.bench.rect, img.Bounds())
		}
	}
	r.drain(t)
	if len(r.s.table.Rows) != Modes || r.s.table.Rows[0][0] != "partial" || r.s.table.Rows[Modes-1][0] != "full" {
		t.Fatalf("one row per mode: %v", r.s.table.Rows)
	}
	if len(r.panel.draws) != Modes*benchIterations || r.panel.waits < Modes*benchIterations {
		t.Errorf("draws: %d, waits %d (each draw is waited for)", len(r.panel.draws), r.panel.waits)
	}
	half := image.Rect(0, 1872-936, 1404, 1872)
	if r.panel.draws[0] != half {
		t.Errorf("the lower half is redrawn: %v", r.panel.draws[0])
	}
	if got := len(r.kinds(t, "bench_mode")); got != Modes {
		t.Errorf("a summary per mode: %d", got)
	}
	if m := r.kinds(t, "bench_mode")[0]; m["mean_ms"].(float64) <= 0 || m["total_mean_ms"].(float64) < m["mean_ms"].(float64) || m["n"].(float64) != benchIterations {
		t.Errorf("summary: %v", m)
	}
	if len(r.kinds(t, "bench_start")) != 1 || len(r.kinds(t, "bench_end")) != 1 {
		t.Error("the run is delimited in the log")
	}
	r.tap(t, "update-samples")
	if r.s.view == nil || r.s.view.group != "sample" {
		t.Error("the samples show the pictures")
	}
}

func TestTheBenchmarkWithoutAPanelSaysSo(t *testing.T) {
	r := newRig(t, false)
	r.tap(t, "eng-update")
	r.tap(t, "update-run")
	if !strings.Contains(r.s.note.Text, "não tem") || r.s.bench != nil {
		t.Errorf("note %q", r.s.note.Text)
	}
}

func TestTheAutoScrollTimesEveryStepAndBouncesAtTheEnds(t *testing.T) {
	r := newRig(t, true)
	r.tap(t, "eng-scroll")
	if r.s.scroll.MaxOffset() <= 0 {
		t.Fatal("the list is taller than the window")
	}
	r.tap(t, "scroll-mode")
	if r.s.mode != Dynamic || r.panel.set[len(r.panel.set)-1] != Dynamic {
		t.Error("the button chooses the mode of the platform's repaints")
	}
	r.tap(t, "scroll-auto")
	r.drain(t)
	if steps := r.kinds(t, "scroll_step"); len(steps) != autoSteps {
		t.Errorf("steps: %d", len(steps))
	}
	run := r.kinds(t, "scroll_run")
	if len(run) != 1 || run[0]["mode"] != "dynamic" || run[0]["fps"].(float64) <= 0 {
		t.Fatalf("run: %v", run)
	}
	if !strings.Contains(r.s.note.Text, "n=40") || r.s.auto != nil {
		t.Errorf("result %q", r.s.note.Text)
	}
	r.s.scroll.ScrollTo(300)
	if len(r.kinds(t, "scroll")) == 0 {
		t.Error("a manual scroll is logged with its offset")
	}
}

func TestTheSystemAndLogPages(t *testing.T) {
	r := newRig(t, true)
	r.tap(t, "eng-system")
	rows := r.s.doc.GetElementByID("system-table").(*widget.Table).Rows
	joined := ""
	for _, row := range rows {
		joined += strings.Join(row, "|") + ";"
	}
	for _, want := range []string{"Modelo|PocketBook 741", "24 bit", "1404 x 1872", "1.2.3"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the system page lacks %q: %s", want, joined)
		}
	}
	r.tap(t, "eng-menu")
	r.tap(t, "eng-log")
	if !strings.Contains(r.s.doc.GetElementByID("log-text").(*widget.Label).Text, `"kind":"system"`) {
		t.Error("the log page shows the last lines")
	}
	r.tap(t, "log-clear")
	if got := r.s.doc.GetElementByID("log-text").(*widget.Label).Text; strings.Contains(got, "system") {
		t.Errorf("cleared: %q", got)
	}
}

func TestEveryEnglishAndPortugueseTextIsThere(t *testing.T) {
	for _, lang := range []string{"pt-BR", "en"} {
		tr := NewTranslator(lang)
		if missing := tr.Missing("pt-BR"); len(missing) > 0 {
			t.Errorf("pt-BR lacks %v", missing)
		}
		if missing := tr.Missing("en"); len(missing) > 0 {
			t.Errorf("en lacks %v", missing)
		}
	}
	tr := NewTranslator("en")
	for _, p := range append(ColorPatterns(), append(ScreenPatterns(), SamplePatterns()...)...) {
		if tr.T("pat."+p.ID) == "pat."+p.ID {
			t.Errorf("no text for the pattern %s", p.ID)
		}
	}
}

func TestEveryRepaintOfAScrollingPageIsLoggedStepByStep(t *testing.T) {
	r := newRig(t, true)
	if r.watch == nil {
		t.Fatal("the screen must ask to hear about the repaints")
	}
	r.tap(t, "eng-scroll")
	r.watch(Paint{Layout: 3 * time.Millisecond, Snapshot: 4 * time.Millisecond, Paint: 5 * time.Millisecond,
		Show: 6 * time.Millisecond, Slid: true, Painted: 1000, Sent: 5000, Areas: 2})
	r.s.cfg.Rec.Flush()
	paints := r.kinds(t, "paint")
	if len(paints) != 1 {
		t.Fatalf("paints: %v", paints)
	}
	p := paints[0]
	if p["page"] != "scroll" || p["layout_ms"].(float64) != 3 || p["show_ms"].(float64) != 6 || p["slid"] != true || p["sent_px"].(float64) != 5000 {
		t.Errorf("paint: %v", p)
	}
	r.tap(t, "scroll-auto")
	r.drain(t)
	if steps := r.kinds(t, "scroll_step"); len(steps) == 0 || steps[0]["layout_ms"] == nil || steps[0]["show_ms"] == nil {
		t.Errorf("each auto step carries the cost of the last repaint: %v", steps)
	}
	r.tap(t, "eng-menu")
	r.watch(Paint{Layout: time.Millisecond})
	r.s.cfg.Rec.Flush()
	if got := len(r.kinds(t, "paint")); got != 1 {
		t.Errorf("a repaint outside the scroll and touch pages is not logged (%d paints)", got)
	}
}
