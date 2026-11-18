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
