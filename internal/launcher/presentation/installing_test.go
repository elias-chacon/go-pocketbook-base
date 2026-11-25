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

// gatedInstaller installs when the test lets it.
type gatedInstaller struct {
	gate chan struct{}
	err  error
}

func (g gatedInstaller) Install(domain.Listing) error {
	<-g.gate
	return g.err
}

func installingHome(inst domain.Installer, p *domain.Progress, lp *loop) *Home {
	return NewHome(Config{
		Width: 1404, Height: 1872, Tr: NewTranslator("pt-BR"),
		Startup: catalogStartup(catalog{list: listings(2)}), Installer: inst, InstallProgress: p, After: lp.after,
		Home:    application.Home{Apps: registry{apps: apps(1)}, Launcher: &launcher{}, SelfID: "launcher"},
		IconFor: func(domain.Tile) image.Image { return image.NewGray(image.Rect(0, 0, 128, 128)) },
	})
}

// settle runs the loop until cond holds (the goroutines of the screen answer in their own time).
func settle(lp *loop, cond func() bool) bool {
	deadline := time.Now().Add(2 * time.Second)
	for !cond() && time.Now().Before(deadline) {
		lp.tick()
		time.Sleep(5 * time.Millisecond)
	}
	return cond()
}

func TestInstallingShowsAWindowWithABarThatFollowsTheDownloadAndThenTheResult(t *testing.T) {
	inst := gatedInstaller{gate: make(chan struct{})}
	var p domain.Progress
	var lp loop
	h := installingHome(inst, &p, &lp)
	settle(&lp, func() bool { return find(h, "app-web00") != nil })
	h.Doc.Layout()
	tap(h.Doc, find(h, "app-web00"))

	if !h.Doc.HasModal() || find(h, "install-bar") == nil {
		t.Fatal("a tap that installs must open a window with a bar")
	}
	if !strings.Contains(texts(h), "Baixando “Web 00”") {
		t.Errorf("the window says what it is doing: %s", texts(h))
	}
	p.Begin(200)
	p.Advance(100)
	lp.tick()
	if bar := find(h, "install-bar").(*widget.Progress); bar.Value != 50 {
		t.Errorf("the bar follows the bytes: %v", bar.Value)
	}

	close(inst.gate)
	if !settle(&lp, func() bool { return strings.Contains(texts(h), "instalado") }) {
		t.Fatalf("the window must say it is installed: %s", texts(h))
	}
	if !h.Doc.HasModal() {
		t.Error("the result stays on the screen until it is closed")
	}
	if find(h, "install-bar") != nil && !find(h, "install-bar").Elem().Hidden {
		t.Error("the bar goes when it is over")
	}
}

func TestAFailedInstallShowsTheErrorInTheWindow(t *testing.T) {
	inst := gatedInstaller{gate: make(chan struct{}), err: errors.New("download corrupted")}
	var lp loop
	h := installingHome(inst, &domain.Progress{}, &lp)
	settle(&lp, func() bool { return find(h, "app-web00") != nil })
	h.Doc.Layout()
	tap(h.Doc, find(h, "app-web00"))
	close(inst.gate)
	if !settle(&lp, func() bool { return strings.Contains(texts(h), "Não foi possível instalar") }) {
		t.Fatalf("the error must show: %s", texts(h))
	}
	if !strings.Contains(texts(h), "download corrupted") || !h.Doc.HasModal() {
		t.Errorf("the reason stays in the window: %s", texts(h))
	}
}

func TestClosingTheResultRunsWhatWasWaiting(t *testing.T) {
	inst := &installer{}
	var lp loop
	h := installingHome(inst, nil, &lp)
	settle(&lp, func() bool { return find(h, "app-web00") != nil })
	h.Doc.Layout()
	back := 0
	h.install(listings(2)[1], false, func() { back++ })
	settle(&lp, func() bool { return strings.Contains(texts(h), "instalado") })
	h.Doc.Modal().Confirm()
	if back != 1 || h.Doc.HasModal() {
		t.Errorf("closing the window goes back (%d) and closes it", back)
	}
}
