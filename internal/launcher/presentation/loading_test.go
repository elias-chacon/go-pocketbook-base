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
		Width: 1404, Height: 1872, Tr: NewTranslator("pt-BR"), Catalog: cat, Progress: p, After: l.after,
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
