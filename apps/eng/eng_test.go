package main

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go-pocketbook-base/internal/platform"
)

// recordingPanel is a platform.Panel that remembers the calls.
type recordingPanel struct {
	draws, settles int
	mode           platform.UpdateMode
}

func (p *recordingPanel) Info() platform.PanelInfo {
	return platform.PanelInfo{Model: "PocketBook 741", Depth: 24}
}
func (p *recordingPanel) DrawGray(*image.Gray, image.Rectangle, platform.UpdateMode) { p.draws++ }
func (p *recordingPanel) DrawColor(_ *image.RGBA, _ image.Rectangle, m platform.UpdateMode) {
	p.draws, p.mode = p.draws+1, m
}
func (p *recordingPanel) SetMode(platform.UpdateMode) {}
func (p *recordingPanel) Wait()                       {}
func (p *recordingPanel) Settle()                     { p.settles++ }

func TestTheAppLogsToTheSharedFolderAndPaintsPicturesItself(t *testing.T) {
	root := t.TempDir()
	panel := &recordingPanel{}
	app := newEng(platform.Env{Width: 1404, Height: 1872, Root: root, Lang: "pt-BR", Version: "9.9.9", Panel: panel}).(*eng)
	t.Cleanup(app.close)
	app.doc.Layout()
	if app.Raw(panel) {
		t.Fatal("on the menu the platform paints")
	}
	if app.Key(platform.KeyBack) {
		t.Error("on the menu a back key has nothing to leave")
	}
	app.rec.Flush()
	data, err := os.ReadFile(filepath.Join(root, "applications", ".pbgo", logName))
	if err != nil || !strings.Contains(string(data), `"kind":"start"`) || !strings.Contains(string(data), `"model":"PocketBook 741"`) {
		t.Fatalf("the start is in the log: %v %q", err, data)
	}
	lines := tailLines(filepath.Join(root, "applications", ".pbgo", logName), 5)
	if len(lines) != 1 || !strings.HasPrefix(lines[0], "{") {
		t.Errorf("tail: %q", lines)
	}
}

func TestTheAppWorksWithoutAPanelAndAnUnwritableLog(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "applications"), []byte("x"), 0o644); err != nil {
		t.Fatal(err) // a file where the folder should be: the log cannot be opened
	}
	app := newEng(platform.Env{Width: 1404, Height: 1872, Root: root, Lang: "en"}).(*eng)
	t.Cleanup(app.close)
	if app.Document() == nil {
		t.Error("the app has a document")
	}
	if got := tailLines(filepath.Join(root, "none.log"), 3); got != nil {
		t.Errorf("no file, no lines: %q", got)
	}
}

func TestTheTailKeepsTheLastLinesOfABigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.log")
	var b strings.Builder
	for i := 0; i < 5000; i++ {
		b.WriteString("line of the log number " + strings.Repeat("x", 20) + "\n")
	}
	b.WriteString("the last\n")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	got := tailLines(path, 3)
	if len(got) != 3 || got[2] != "the last" || strings.HasPrefix(got[0], "x") {
		t.Errorf("tail: %q", got)
	}
}

func TestSizesAreWrittenInTheBiggestUnit(t *testing.T) {
	for in, want := range map[uint64]string{512: "512 B", 2048: "2.0 KB", 5 << 20: "5.0 MB", 3 << 30: "3.0 GB"} {
		if got := bytes(in); got != want {
			t.Errorf("%d: %q want %q", in, got, want)
		}
	}
}
