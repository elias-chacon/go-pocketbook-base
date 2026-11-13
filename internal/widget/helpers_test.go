package widget

import "go-pocketbook-base/internal/geom"

// testDoc puts w at the top-left of a 1404x1872 document and lays it out.
func testDoc(w Widget) *Document {
	d := NewDocument(1404, 1872)
	root := NewColumn(0, w)
	root.Style.AlignItems = AlignStart
	d.SetRoot(root)
	d.Layout()
	return d
}

// tapAt simulates a touch down and up at p (design pixels).
func tapAt(d *Document, p geom.Point) {
	d.HandlePointer(PointerDown, p.X*d.Scale, p.Y*d.Scale)
	d.HandlePointer(PointerUp, p.X*d.Scale, p.Y*d.Scale)
}

// tap touches the center of w.
func tap(d *Document, w Widget) { tapAt(d, w.Elem().Bounds.Center()) }

// counter counts handler invocations.
type counter struct{ n int }

func (c *counter) handler() Handler { return func(*Event) { c.n++ } }

func geomPt(x, y float64) geom.Point { return geom.Point{X: x, Y: y} }
