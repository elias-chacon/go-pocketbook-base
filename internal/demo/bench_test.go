package demo

import (
	"fmt"
	"testing"
)

// BenchmarkRender measures a full repaint of each demo page at PB741 size
// (run with -benchmem: allocations matter on a 1 GB device).
func BenchmarkRender(b *testing.B) {
	app, _ := newApp(nil)
	for i, title := range app.Pages() {
		app.ShowPage(i)
		b.Run(fmt.Sprintf("%d-%s", i, title), func(b *testing.B) {
			b.ReportAllocs()
			for n := 0; n < b.N; n++ {
				app.Doc.Render()
			}
		})
	}
}

// BenchmarkUpdate measures one ordinary tap (the Data page's "advance" button,
// which moves a progress bar and the footer): layout + snapshot + the partial
// repaint. Compare with BenchmarkRender/2-Dados, the full repaint it replaces.
func BenchmarkUpdate(b *testing.B) {
	app, _ := newApp(nil)
	app.ShowPage(2)
	app.Doc.Update()
	p := app.Doc.GetElementByID("button-advance").Elem().Bounds.Center()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		tapPoint(app, p)
		app.Doc.Update()
	}
}

// BenchmarkSnapshotOnly isolates the cost of the change detection.
func BenchmarkSnapshotOnly(b *testing.B) {
	app, _ := newApp(nil)
	app.ShowPage(2)
	app.Doc.Update()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		app.Doc.Update() // nothing changed: layout + snapshot + compare
	}
}
