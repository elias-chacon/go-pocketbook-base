package showcase

import (
	"image"
	"testing"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/widget"
)

// edgeTolerance is how far (gray levels, of 255) a pixel of the partially
// repainted frame may be from the full render. Shapes drawn in a window are
// offset by whole pixels, but the rasterizer works in 1/64 px fixed point, so an
// anti-aliased corner can land one step away (1-2 levels). Anything bigger is a
// real difference (text or a shape in the wrong place).
const edgeTolerance = 8

// samePixels fails the test, naming the first real difference, when the frame
// patched by Update does not match a fresh full render.
func samePixels(t *testing.T, what string, got *image.Gray, want *image.Gray) {
	t.Helper()
	for i := range got.Pix {
		d := int(got.Pix[i]) - int(want.Pix[i])
		if d > edgeTolerance || d < -edgeTolerance {
			t.Fatalf("%s: partial repaint differs from a full render at pixel (%d,%d): %d != %d",
				what, i%got.Stride, i/got.Stride, got.Pix[i], want.Pix[i])
		}
	}
}

func TestPartialRepaintMatchesFullRender(t *testing.T) {
	app, _ := newApp(nil)
	partial, full := 0, 0
	for page, title := range app.Pages() {
		app.ShowPage(page)
		f := app.Doc.Update()
		samePixels(t, title+" (after showing)", f.Image, app.Doc.Render())

		// Tap the center of every interactive element, one after the other, and
		// check the screen after each tap.
		var targets []geom.Point
		walkVisible(app, func(w widget.Widget) {
			if _, ok := w.(widget.PointerHandler); ok {
				targets = append(targets, w.Elem().Bounds.Center())
			}
		})
		for i, p := range targets {
			tapPoint(app, p)
			f := app.Doc.Update()
			if f.Full {
				full++
			} else {
				partial++
			}
			samePixels(t, title+" tap "+string(rune('A'+i%26)), f.Image, app.Doc.Render())
			app.Doc.ClosePopups() // an open popup would swallow the next tap
			samePixels(t, title+" after closing popups", app.Doc.Update().Image, app.Doc.Render())
		}
		app.ShowPage(page) // start the next round from a known state
		app.Doc.Update()
	}
	if partial == 0 {
		t.Fatal("no tap was repainted partially: the optimization never kicks in")
	}
	t.Logf("%d taps repainted partially, %d fully", partial, full)
}

func TestNothingChangedMeansNothingToPaint(t *testing.T) {
	app, _ := newApp(nil)
	if f := app.Doc.Update(); !f.Full {
		t.Fatal("the first frame is always a full repaint")
	}
	if f := app.Doc.Update(); f.Full || len(f.Rects) != 0 {
		t.Errorf("an unchanged screen must not repaint: full=%v rects=%v", f.Full, f.Rects)
	}
}

func TestOneButtonRepaintsOnlyItsArea(t *testing.T) {
	app, _ := newApp(nil)
	app.ShowPage(2) // Dados: the "advance" button changes the progress bar
	app.Doc.Update()
	tapPoint(app, byID(t, app, "button-advance").Elem().Bounds.Center())
	f := app.Doc.Update()
	if f.Full || len(f.Rects) == 0 {
		t.Fatalf("expected a partial repaint, got full=%v rects=%v", f.Full, f.Rects)
	}
	total := 0
	for _, r := range f.Rects {
		total += r.Dx() * r.Dy()
	}
	if screen := app.Doc.DeviceW * app.Doc.DeviceH; total*5 > screen {
		t.Errorf("a progress bar and a footer label repainted %d%% of the screen", 100*total/screen)
	}
	samePixels(t, "progress", f.Image, app.Doc.Render())
}

func TestPageSwitchAndKeyboardRepaintEverything(t *testing.T) {
	app, _ := newApp(nil)
	app.Doc.Update()
	app.ShowPage(1)
	if f := app.Doc.Update(); !f.Full {
		t.Error("switching page must repaint the whole screen")
	}
	app.Doc.InvalidateAll()
	if f := app.Doc.Update(); !f.Full {
		t.Error("InvalidateAll must repaint the whole screen")
	}
}

func TestSliderDragRepaintsPartially(t *testing.T) {
	app, _ := newApp(nil)
	app.ShowPage(1)
	app.Doc.Update()
	s := byID(t, app, "slider-basic").Elem().Bounds
	y := s.Center().Y * app.Doc.Scale
	app.Doc.HandlePointer(widget.PointerDown, (s.X+40)*app.Doc.Scale, y)
	for _, f := range []float64{0.3, 0.5, 0.8} {
		app.Doc.HandlePointer(widget.PointerMove, (s.X+s.W*f)*app.Doc.Scale, y)
		fr := app.Doc.Update()
		if fr.Full {
			t.Errorf("dragging to %.0f%% repainted the whole screen", f*100)
		}
		samePixels(t, "slider drag", fr.Image, app.Doc.Render())
	}
}
