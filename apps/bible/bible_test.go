package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"go-pocketbook-base/internal/platform"
	"go-pocketbook-base/internal/platform/infrastructure/jsonfiles"
	"go-pocketbook-base/internal/toolkit/gfx"
	"go-pocketbook-base/internal/toolkit/widget"
)

// api is a tiny Midvash: two books, two versions, 12 verses of 200 letters in each chapter.
func api(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/books":
			fmt.Fprint(w, `{"data":[
				{"id":1,"name":{"en":"Genesis","pt-br":"Gênesis"},"slug":{"en":"genesis"},"chapters":2,"testament":"old"},
				{"id":43,"name":{"en":"John","pt-br":"João"},"slug":{"en":"john"},"chapters":2,"testament":"new"}]}`)
		case "/v1/versions":
			fmt.Fprint(w, `{"data":[
				{"slug":"almeida-livre","name":"Bíblia Livre","shortName":"BL","language":"pt-br","totalBooks":2,"totalChapters":4},
				{"slug":"kjv","name":"King James","shortName":"KJV","language":"en","totalBooks":2,"totalChapters":4}]}`)
		case "/v1/passages":
			version := r.URL.Query().Get("version")
			var items []string
			for _, ref := range strings.Split(r.URL.Query().Get("refs"), ",") {
				var slug string
				var chapter int
				fmt.Sscanf(ref, "%s %d", &slug, &chapter)
				var verses []string
				for v := 1; v <= 12; v++ {
					verses = append(verses, fmt.Sprintf("%q", fmt.Sprintf("%s %s %d:%d %s", version, slug, chapter, v, strings.Repeat("texto ", 30))))
				}
				items = append(items, fmt.Sprintf(`{"book":%q,"chapter":%d,"verse":1,"verses":[%s]}`, slug, chapter, strings.Join(verses, ",")))
			}
			fmt.Fprintf(w, `{"data":[%s]}`, strings.Join(items, ","))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func testEnv(t *testing.T, srv *httptest.Server) platform.Env {
	t.Helper()
	t.Setenv("LC_ALL", "pt_BR.UTF-8")
	t.Cleanup(func() { gfx.SetFamily(gfx.FamilyGo) })
	m, err := platform.ParseManifest(manifest)
	if err != nil {
		t.Fatal(err)
	}
	env := platform.Env{
		Width: 1404, Height: 1872, Manifest: m, Lang: "pt-BR", Version: "1.0.0",
		Settings: jsonfiles.NewSettings(t.TempDir()), Keyboard: &widget.FakeKeyboard{Reply: "x"}, Exit: func() {},
	}
	if srv != nil {
		catalogBase = srv.URL + "/v1"
		env.HTTP = srv.Client()
	}
	return env
}

type noSpeech struct{}

func (noSpeech) Available() bool    { return false }
func (noSpeech) Speak(string) error { return nil }
func (noSpeech) Stop()              {}

func find(d *widget.Document, id string) widget.Widget {
	var found widget.Widget
	d.Walk(func(w widget.Widget) bool {
		if w.Elem().ID == id {
			found = w
		}
		return true
	})
	return found
}

func tap(t *testing.T, d *widget.Document, id string) {
	t.Helper()
	d.Layout()
	w := find(d, id)
	if w == nil {
		t.Fatalf("no element %q", id)
	}
	c := w.Elem().Bounds.Center()
	d.HandlePointer(widget.PointerDown, c.X*d.Scale, c.Y*d.Scale)
	d.HandlePointer(widget.PointerUp, c.X*d.Scale, c.Y*d.Scale)
	d.Layout()
}

func TestTheManifestFollowsTheContract(t *testing.T) {
	m, err := platform.ParseManifest(manifest)
	if err != nil || m.ID != "bible" {
		t.Fatalf("%+v %v", m, err)
	}
	if m.TitleIn("pt-BR") != "Bíblia" || m.TitleIn("en") != "Bible" || m.DescriptionIn("en") == "" {
		t.Errorf("title = %q / %q", m.TitleIn("pt-BR"), m.TitleIn("en"))
	}
}

func TestDownloadReadAndCloseThenEverythingIsStillThereWhenReopened(t *testing.T) {
	srv := api(t)
	dir := t.TempDir()
	a := newBibleAt(testEnv(t, srv), dir, noSpeech{})
	if a.screen == nil {
		t.Fatal("no screen")
	}
	d := a.Document()
	d.Layout()
	if find(d, "get-bible") == nil {
		t.Fatal("the first run asks to download")
	}
	tap(t, d, "get-bible") // lists the versions and downloads the Portuguese one through the (local) API
	if find(d, "progress") == nil || find(d, "votd") == nil {
		t.Fatal("downloaded: the Home shows the progress and the verse of the day")
	}
	tap(t, d, "votd-read")
	a.screen.Show(1)
	tap(t, d, "book-43")
	tap(t, d, "chapter-2")
	tap(t, d, "verse-5")
	prefs := a.screen.Prefs()
	prefs.FontSize, prefs.Invert, prefs.Font = 44, true, "mono"
	a.screen.ApplyPreferences(prefs)
	a.close()

	// the next run: the same text, progress, place and settings, and no network at all
	catalogBase = "http://127.0.0.1:1/never"
	env := testEnv(t, nil)
	b := newBibleAt(env, dir, noSpeech{})
	t.Cleanup(b.close)
	if b.screen == nil {
		t.Fatal("no screen after reopening")
	}
	d = b.Document()
	d.Layout()
	got := b.screen.Prefs()
	if got.FontSize != 44 || !got.Invert || got.Font != "mono" || got.Version != "almeida-livre" {
		t.Errorf("preferences: %+v", got)
	}
	if !d.Invert || gfx.CurrentFamily() != gfx.FamilyMono {
		t.Error("the look is applied at start")
	}
	if find(d, "get-bible") != nil || find(d, "votd-done") == nil {
		t.Error("the version stays downloaded and the verse of the day stays read")
	}
	bar := find(d, "progress").(*widget.Progress)
	if bar.Value <= 0 {
		t.Errorf("the progress is kept: %v", bar.Value)
	}
	tap(t, d, "continue")
	if find(d, "reader") == nil || !strings.Contains(find(d, "reader-title").(*widget.Label).Text, "João 2") {
		t.Error("the reading resumes at John 2")
	}
	if !b.Key(platform.KeyNext) || b.Key(platform.KeyOther) {
		t.Error("the page keys are the app's, the other keys are not")
	}
}

func TestWithoutANetworkTheAppStillStartsAndSaysWhatFailed(t *testing.T) {
	a := newBibleAt(testEnv(t, nil), t.TempDir(), noSpeech{})
	t.Cleanup(a.close)
	d := a.Document()
	d.Layout()
	tap(t, d, "get-bible")
	if a.screen.Page() != 0 || find(d, "get-bible") == nil {
		t.Error("there is no way to download: the Home still asks")
	}
}

func TestAFolderThatCannotBeCreatedShowsTheError(t *testing.T) {
	file := t.TempDir() + "/file"
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := newBibleAt(testEnv(t, nil), file+"/sub", noSpeech{})
	if a.screen != nil || a.Key(platform.KeyNext) {
		t.Error("no screen without a database")
	}
	a.close()
	a.Document().Layout()
}

func TestTheLanguageOfTheReaderIsRecognized(t *testing.T) {
	for tag, want := range map[string]string{"pt_BR.UTF-8": "pt-BR", "pt": "pt-BR", "Portuguese": "pt-BR", "en_US.UTF-8": "en", "English": "en", "de_DE": "", "C": ""} {
		if got := fromTag(tag); got != want {
			t.Errorf("fromTag(%q) = %q, want %q", tag, got, want)
		}
	}
	for _, k := range posixVariables {
		t.Setenv(k, "")
	}
	if got := deviceLanguage("pt-BR"); got != "pt-BR" {
		t.Errorf("nothing known: the language of the shared settings: %q", got)
	}
	t.Setenv("LANG", "en_GB.UTF-8")
	if got := deviceLanguage("pt-BR"); got != "en" {
		t.Errorf("LANG: %q", got)
	}
	if fromConfig("/does/not/exist") != "" {
		t.Error("no file, no language")
	}
}

func TestTheSpeechProgramIsOptional(t *testing.T) {
	s := &commandSpeaker{}
	if s.Available() {
		t.Error("no program, no speech")
	}
	s.Stop()
	if environSpeaker() == nil {
		t.Error("a speaker is always given (Available says whether it works)")
	}
}
