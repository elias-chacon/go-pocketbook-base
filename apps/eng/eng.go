package main

import (
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go-pocketbook-base/internal/engineering"
	"go-pocketbook-base/internal/platform"
	"go-pocketbook-base/internal/toolkit/widget"
)

// logName is the log file, in applications/.pbgo (the folder every app of the ecosystem keeps its
// files in); it holds one JSON object per line.
const logName = "eng.log"

// maxLog is the size past which the log starts again when the app opens: it lives on the reader's
// small flash and is meant to be read after a test session, not kept for good.
const maxLog = 4 << 20

type eng struct {
	doc    *widget.Document
	screen *engineering.Screen
	rec    *engineering.Recorder
	file   io.Closer // the log file (nil when it could not be opened)
}

// newEng builds the app: the document, the log file and the screen of tests.
func newEng(env platform.Env) platform.App {
	a := &eng{doc: widget.NewDocument(env.Width, env.Height)}
	path := logPath(env.Root)
	w, closer := openLog(path)
	a.rec, a.file = engineering.NewRecorder(w, nil), closer
	exit := env.Exit
	chrome := platform.Chrome(env)
	chrome.Exit = func() {
		a.rec.Flush()
		if exit != nil {
			exit()
		}
	}
	cfg := engineering.Config{
		Width: env.Width, Height: env.Height, Doc: a.doc, Chrome: chrome,
		Tr: engineering.NewTranslator(env.Lang), Rec: a.rec, After: env.After, Flush: env.Flush,
		Facts: func() []engineering.Fact { return facts(env, path) }, LogPath: path,
		LogLines: func(n int) []string { return tailLines(path, n) },
		ClearLog: func() { _ = os.Truncate(path, 0) },
	}
	if env.Panel != nil {
		cfg.Panel = panel{env.Panel}
	}
	if env.Watch != nil {
		cfg.Watch = func(f func(engineering.Paint)) {
			env.Watch(func(p platform.PaintStats) { f(paintOf(p)) })
		}
	}
	a.screen = engineering.NewScreen(cfg)
	return a
}

// Document implements platform.App.
func (a *eng) Document() *widget.Document { return a.doc }

// Key implements platform.KeyHandler: a back key leaves the test for the menu.
func (a *eng) Key(k platform.Key) bool { return k.IsBack() && a.screen.Back() }

// Raw implements platform.Raw: a full-screen picture is painted by the screen itself.
func (a *eng) Raw(p platform.Panel) bool { return a.screen.Raw(panel{p}) }

// logPath is where the log is, under the storage root.
func logPath(root string) string {
	if root == "" {
		root = platform.DefaultRoot
	}
	return filepath.Join(root, "applications", ".pbgo", logName)
}

// openLog opens the log for appending (starting it again when it is too big); a log that cannot be
// opened discards what is written, so a read-only card never stops the tests.
func openLog(path string) (io.Writer, io.Closer) {
	if st, err := os.Stat(path); err == nil && st.Size() > maxLog {
		_ = os.Truncate(path, 0)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return io.Discard, nil
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return io.Discard, nil
	}
	return f, f
}

// close writes what is buffered and closes the log file.
func (a *eng) close() {
	a.rec.Flush()
	if a.file != nil {
		_ = a.file.Close()
	}
}

// tailLines are the last n lines of a file.
func tailLines(path string, n int) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	const window = 64 << 10
	st, err := f.Stat()
	if err != nil {
		return nil
	}
	start := st.Size() - window
	if start < 0 {
		start = 0
	}
	buf := make([]byte, st.Size()-start)
	if _, err := f.ReadAt(buf, start); err != nil && err != io.EOF {
		return nil
	}
	lines := strings.Split(strings.TrimRight(string(buf), "\n"), "\n")
	if start > 0 && len(lines) > 0 {
		lines = lines[1:] // the first one is cut in the middle
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}

// facts are the lines of the system page that only the platform knows.
func facts(env platform.Env, path string) []engineering.Fact {
	out := []engineering.Fact{{Key: "version", Value: env.Version}}
	if total, free, ok := platform.DiskSpace(platform.DefaultRoot); ok {
		out = append(out, engineering.Fact{Key: "storage", Value: fmt.Sprintf("%s / %s", bytes(free), bytes(total))})
	}
	return append(out, engineering.Fact{Key: "folder", Value: filepath.Dir(path)})
}

// bytes writes a size in the biggest unit that keeps it above one.
func bytes(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := uint64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGT"[exp])
}

// paintOf is the cost of a repaint, as the tests of the engineering app see it.
func paintOf(p platform.PaintStats) engineering.Paint {
	return engineering.Paint{
		Layout: p.Layout, Snapshot: p.Snapshot, Diff: p.Diff, Paint: p.UpdateStats.Paint, Show: p.Show,
		Full: p.Full, Slid: p.Slid, Painted: p.Painted, Sent: p.Sent, Areas: p.Areas,
	}
}

// panel adapts the platform's panel to the one the tests use.
type panel struct{ p platform.Panel }

func (a panel) Info() engineering.Info {
	i := a.p.Info()
	return engineering.Info{Model: i.Model, Depth: i.Depth, Waveform: i.Waveform, Canvas: i.Canvas}
}

func (a panel) DrawGray(img *image.Gray, r image.Rectangle, m engineering.Mode) {
	a.p.DrawGray(img, r, platform.UpdateMode(m))
}

func (a panel) DrawColor(img *image.RGBA, r image.Rectangle, m engineering.Mode) {
	a.p.DrawColor(img, r, platform.UpdateMode(m))
}

func (a panel) SetMode(m engineering.Mode) { a.p.SetMode(platform.UpdateMode(m)) }
func (a panel) Wait()                      { a.p.Wait() }
func (a panel) Settle()                    { a.p.Settle() }
