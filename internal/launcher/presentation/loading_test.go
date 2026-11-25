package presentation

import (
	"errors"
	"image"
	"strings"
	"testing"
	"time"

	"go-pocketbook-base/internal/launcher/application"
	"go-pocketbook-base/internal/launcher/domain"
	"go-pocketbook-base/internal/toolkit/widget"
)

// slowCatalog answers when the test lets it.
type slowCatalog struct {
	release chan struct{}
	list    []domain.Listing
	err     error
}

func (c slowCatalog) List(string) ([]domain.Listing, error) {
	<-c.release
	return c.list, c.err
}

// catalogStartup is the startup of a test: only the catalog.
func catalogStartup(c domain.Catalog) application.Startup {
	if c == nil {
		return application.Startup{}
	}
	return application.Startup{Tasks: []application.Task{application.FetchCatalog(c)}}
}

// loop is the event loop of a test: the timers the screen asks for wait until the test runs them.
type loop struct{ queue []func() }

func (l *loop) after(_ time.Duration, f func()) { l.queue = append(l.queue, f) }

// tick runs what was waiting (and what that asks for goes to the next tick).
func (l *loop) tick() {
	q := l.queue
	l.queue = nil
	for _, f := range q {
		f()
	}
}

func loadingHome(n int, cat domain.Catalog, p *domain.Progress, l *loop) *Home {
	return NewHome(Config{
		Width: 1404, Height: 1872, Tr: NewTranslator("pt-BR"), Startup: catalogStartup(cat), Progress: p, After: l.after,
		Home:    application.Home{Apps: registry{apps: apps(n)}, Launcher: &launcher{}, SelfID: "launcher"},
		IconFor: func(domain.Tile) image.Image { return image.NewGray(image.Rect(0, 0, 128, 128)) },
	})
}

func TestTheBarShowsFromTheFirstFrameAndFollowsTheDownloadUntilTheAppsCome(t *testing.T) {
	cat := slowCatalog{release: make(chan struct{}), list: []domain.Listing{{ID: "web00", Title: "Web 00", Version: "1", AppSHA256: "x"}}}
	var p domain.Progress
	var lp loop
	h := loadingHome(0, cat, &p, &lp)
	h.Doc.Layout()

	bar, ok := find(h, "home-loading").(*widget.Progress)
	if !ok {
		t.Fatal("the bar must be on the screen as soon as the launcher opens")
	}
	if strings.Contains(texts(h), "Copie") || strings.Contains(texts(h), "Nenhum aplicativo") {
		t.Errorf("while the catalog comes the screen must not say there are no apps: %q", texts(h))
	}

	p.Begin(4)
	p.Step()
	p.Step()
	lp.tick()
	if bar = find(h, "home-loading").(*widget.Progress); bar.Value != 50 || !strings.Contains(bar.Text, "50") {
		t.Errorf("the bar must show 2 of 4: value=%v text=%q", bar.Value, bar.Text)
	}

	close(cat.release)
	deadline := time.Now().Add(2 * time.Second)
	for find(h, "home-loading") != nil && time.Now().Before(deadline) {
		lp.tick()
		time.Sleep(5 * time.Millisecond)
	}
	if find(h, "home-loading") != nil {
		t.Fatal("the bar must go when the catalog arrives")
	}
	if len(tiles(h)) != 1 {
		t.Errorf("the app of the catalog must show: %d tiles", len(tiles(h)))
	}
}

func TestWithNoAppsAndNoNetworkTheScreenSaysSoWithoutAskingToCopyFiles(t *testing.T) {
	cat := slowCatalog{release: make(chan struct{}), err: errors.New("offline")}
	var lp loop
	h := loadingHome(0, cat, &domain.Progress{}, &lp)
	close(cat.release)
	deadline := time.Now().Add(2 * time.Second)
	for find(h, "home-loading") != nil && time.Now().Before(deadline) {
		lp.tick()
		time.Sleep(5 * time.Millisecond)
	}
	got := texts(h)
	if !strings.Contains(got, "Wi-Fi") || strings.Contains(got, "Copie") {
		t.Errorf("empty text = %q", got)
	}
}

func TestTheBarWaitsAreShortSoItMovesButTheTotalWaitIsTheSame(t *testing.T) {
	var p domain.Progress
	h := &Home{cfg: Config{Progress: &p}}
	if h.pollEvery() != barPoll || barPoll*time.Duration(catalogPolls*int(catalogPoll/barPoll)) > catalogPoll*catalogPolls {
		t.Errorf("poll = %v", h.pollEvery())
	}
	if (&Home{}).pollEvery() != catalogPoll {
		t.Error("without a progress the poll stays as it was")
	}
}

type fakeHome struct{ removed []string }

func (f fakeHome) RemoveOrphans(report func(string)) ([]string, error) {
	for _, id := range f.removed {
		report(id)
	}
	return f.removed, nil
}

func TestTheHomeSaysWhenTheStartupTookEntriesOutOfTheHomeScreen(t *testing.T) {
	var lp loop
	h := NewHome(Config{
		Width: 1404, Height: 1872, Tr: NewTranslator("pt-BR"), After: lp.after,
		Startup: application.Startup{Tasks: []application.Task{application.PruneHomeScreen(fakeHome{removed: []string{"a", "b"}})}},
		Home:    application.Home{Apps: registry{apps: apps(1)}, Launcher: &launcher{}, SelfID: "launcher"},
		IconFor: func(domain.Tile) image.Image { return image.NewGray(image.Rect(0, 0, 128, 128)) },
	})
	if !settle(&lp, func() bool { return find(h, "home-loading") == nil }) {
		t.Fatal("the startup must end")
	}
	if !strings.Contains(texts(h), "tela inicial: 2") {
		t.Errorf("the note says how many entries went: %q", texts(h))
	}
}

func TestALongMessageWrapsAndLeavesTheButtonsOnTheScreen(t *testing.T) {
	h := newHome(3, domain.Grid{Columns: 3, Rows: 3}, &launcher{})
	h.say(strings.Repeat("uma mensagem muito comprida que não cabe numa linha só ", 6))
	h.Doc.Layout()
	for _, id := range []string{"home-settings", "home-prev", "home-next"} {
		b := find(h, id).Elem().Bounds
		if b.X < 0 || b.X+b.W > 1404 {
			t.Errorf("%s is out of the screen: %+v", id, b)
		}
	}
	if note := h.note.Bounds; note.X+note.W > 1404 || note.H < 60 {
		t.Errorf("the message must wrap inside the screen: %+v", note)
	}
}

func TestTheDownloadEndsQuietlyWhileTheSettingsAreOpenAndTheBarIsNotLeftBehind(t *testing.T) {
	cat := slowCatalog{release: make(chan struct{}), list: []domain.Listing{{ID: "web00", Title: "Web 00", Version: "1", AppSHA256: "x"}}}
	var p domain.Progress
	var lp loop
	h := loadingHome(1, cat, &p, &lp)
	h.Doc.Layout()
	if find(h, "home-loading") == nil {
		t.Fatal("the bar is up while the catalog comes")
	}
	h.OpenSettings()
	close(cat.release) // the download ends with the settings open
	settle(&lp, func() bool { return false })
	h.CloseSettings()
	h.Doc.Layout()
	if find(h, "home-loading") != nil {
		t.Error("the bar must not stay on the home after the download ended")
	}
	if len(tiles(h)) != 2 {
		t.Errorf("the catalog that came meanwhile shows: %d tiles", len(tiles(h)))
	}
}

func TestComingBackFromTheSettingsInTheMiddleOfTheDownloadKeepsTheBarMoving(t *testing.T) {
	cat := slowCatalog{release: make(chan struct{}), list: []domain.Listing{{ID: "web00", Title: "Web 00", Version: "1", AppSHA256: "x"}}}
	var p domain.Progress
	var lp loop
	h := loadingHome(1, cat, &p, &lp)
	for _, on := p.State(); !on; _, on = p.State() { // the startup began its job
		time.Sleep(time.Millisecond)
	}
	p.Add(1) // 3 steps in all
	p.Step()
	p.Step()
	h.OpenSettings()
	h.CloseSettings()
	h.Doc.Layout()
	bar, ok := find(h, "home-loading").(*widget.Progress)
	if !ok || bar.Value != 67 {
		t.Fatalf("the bar comes back where the download is: %v %v", ok, bar)
	}
	close(cat.release)
	if !settle(&lp, func() bool { return find(h, "home-loading") == nil }) {
		t.Error("and goes when the download ends")
	}
}
