package widget

import (
	"image"
	"image/color"
	"testing"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
)

// halfImage is w x h, black on the left half and white on the right.
func halfImage(w, h int) *image.Gray {
	g := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := w / 2; x < w; x++ {
			g.SetGray(x, y, color.Gray{Y: 255})
		}
	}
	return g
}

func TestImageMeasuresItsNaturalSizeAndShrinksToFit(t *testing.T) {
	i := NewImage(halfImage(200, 100))
	if got := i.Measure(1000); got.W != 200 || got.H != 100 {
		t.Errorf("natural size = %+v", got)
	}
	if got := i.Measure(100); got.W != 100 || got.H != 50 {
		t.Errorf("shrunk to 100 wide must keep the proportion: %+v", got)
	}
	if got := NewImage(nil).Measure(500); got.W != 0 || got.H != 0 {
		t.Errorf("no source = no size: %+v", got)
	}
}

func TestImageIsDrawnInsideItsBoxAndCanBeTapped(t *testing.T) {
	img := NewImage(halfImage(100, 100))
	taps := &counter{}
	img.OnClick = taps.handler()
	d := testDoc(img)
	d.Layout()
	pic := d.Render()
	b := img.Bounds
	if pic.GrayAt(int(b.X)+20, int(b.Y)+50).Y > 10 || pic.GrayAt(int(b.X)+80, int(b.Y)+50).Y < 245 {
		t.Error("the image must be painted inside its bounds")
	}
	tap(d, img)
	if taps.n != 1 {
		t.Error("an image with OnClick is a tap target (an icon tile)")
	}
	if got := d.TargetAt(img.Bounds.Center()); got != Widget(img) {
		t.Errorf("target = %T", got)
	}
}

func TestImageFitModesAndSizedBox(t *testing.T) {
	img := NewImage(halfImage(100, 100))
	img.Style.Width, img.Style.Height = Px(200), Px(100) // a wide box for a square image
	d := testDoc(img)
	d.Layout()
	pic := d.Render()
	b := img.Bounds
	if pic.GrayAt(int(b.X)+5, int(b.Y)+50).Y != Paper.R {
		t.Error("contain leaves the sides of a wide box empty")
	}
	img.Fit = gfx.FitFill
	if d.Render().GrayAt(int(b.X)+5, int(b.Y)+50).Y > 10 {
		t.Error("fill stretches over the whole box")
	}
}

func TestBackgroundImageIsPaintedOverTheBackground(t *testing.T) {
	p := NewPanel()
	p.Style.Width, p.Style.Height = Px(200), Px(100)
	p.Style.Background = Soft
	p.Style.BackgroundImage = halfImage(200, 100)
	d := testDoc(p)
	d.Layout()
	pic := d.Render()
	b := p.Bounds
	if pic.GrayAt(int(b.X)+20, int(b.Y)+50).Y > 10 || pic.GrayAt(int(b.X)+180, int(b.Y)+50).Y < 245 {
		t.Error("the background image must cover the panel")
	}

	p.Style.BorderRadius = 30 // rounded: the picture is cut to the corners
	if d.Render().GrayAt(int(b.X)+1, int(b.Y)+1).Y != Paper.R {
		t.Error("the corner outside the rounded box must not show the image")
	}
}

func TestPartialRepaintOverABackgroundImageMatchesAFullRender(t *testing.T) {
	label := NewLabel("before")
	root := NewColumn(0, label)
	root.Style.BackgroundImage = halfImage(300, 300)
	root.Style.Padding = geom.All(40)
	d := NewDocument(1404, 1872)
	d.SetRoot(root)
	d.Update()

	label.Text = "after the change"
	f := d.Update()
	if f.Full || len(f.Rects) == 0 {
		t.Fatalf("a label change must repaint partially: full=%v rects=%v", f.Full, f.Rects)
	}
	want := d.Render()
	for i := range f.Image.Pix {
		if diff := int(f.Image.Pix[i]) - int(want.Pix[i]); diff > 8 || diff < -8 {
			t.Fatalf("pixel %d differs from a full render: %d != %d", i, f.Image.Pix[i], want.Pix[i])
		}
	}
}
