package widget

import "go-pocketbook-base/internal/kernel/geom"

// unbounded stands for "no limit" along an axis that is sized by its content.
const unbounded = 1e9

// resolve turns a Length into pixels; ok is false for auto.
func resolve(l Length, avail float64) (v float64, ok bool) {
	switch l.Unit {
	case UnitPx:
		return l.V, true
	case UnitPct:
		return l.V / 100 * avail, true
	}
	return 0, false
}

func visibleChildren(w Widget) []Widget {
	var out []Widget
	for _, c := range w.Elem().children {
		if c.Elem().Visible() {
			out = append(out, c)
		}
	}
	return out
}

// measure returns the border-box size of w when it may use at most maxW pixels
// of width: explicit Width/Height win, otherwise the content decides.
func measure(w Widget, maxW float64) geom.Size { return measureForced(w, maxW, -1) }

// measureForced is measure with a width imposed by the parent (stretch / grid
// cell) when the element has no explicit width; forceW < 0 means none.
func measureForced(w Widget, maxW, forceW float64) geom.Size {
	st := &w.Elem().Style
	ex := st.boxInsets()
	outerW, wDef := resolve(st.Width, maxW)
	if !wDef && forceW >= 0 {
		outerW, wDef = forceW, true
	}
	availContent := geom.Max(0, maxW-ex.Horizontal())
	if wDef {
		availContent = geom.Max(0, outerW-ex.Horizontal())
	}

	nat := contentSize(w, availContent, wDef)
	if !wDef {
		outerW = geom.Min(nat.W, availContent) + ex.Horizontal()
	}
	outerH := nat.H + ex.Vertical()
	if st.Height.Unit == UnitPx {
		outerH = st.Height.V
	}
	return geom.Size{W: geom.Max(outerW, st.MinWidth), H: geom.Max(outerH, st.MinHeight)}
}

// contentSize is the natural size of the content box.
func contentSize(w Widget, availW float64, definiteW bool) geom.Size {
	if len(w.Elem().children) == 0 {
		return w.Measure(availW)
	}
	size := arrange(w, geom.Rect{W: availW, H: unbounded}, definiteW, false, false)
	if _, window := w.(clipper); window {
		size.H = 0 // a window onto its content: its height is not the content's (Height or FlexGrow give it one)
	}
	return size
}

// layoutWidget places w in r (its border box) and lays out its subtree.
func layoutWidget(w Widget, r geom.Rect) {
	b := w.Elem()
	b.Bounds = r
	if len(b.children) > 0 {
		arrange(w, r.Inset(b.Style.boxInsets()), true, true, true)
		if a, ok := w.(afterLayouter); ok {
			a.afterLayout()
		}
	}
}

// arrange lays out the children of w inside content and returns the content
// size they use. With apply=false nothing is positioned (measuring only).
func arrange(w Widget, content geom.Rect, defW, defH, apply bool) geom.Size {
	if w.Elem().Style.Display == DisplayGrid {
		return arrangeGrid(w, content, defW, apply)
	}
	return arrangeFlex(w, content, defW, defH, apply)
}

// ---- flex ----

// flexSpec is a flex container's configuration; Block is a vertical flex that stretches.
type flexSpec struct {
	vertical bool
	wrap     bool
	justify  Justify
	align    Align
	gap      float64
}

func specOf(st *Style) flexSpec {
	if st.Display == DisplayFlex {
		return flexSpec{st.FlexDirection == Column, st.FlexWrap, st.JustifyContent, st.AlignItems, st.Gap}
	}
	return flexSpec{vertical: true, gap: st.Gap}
}

func (s flexSpec) main(z geom.Size) float64 {
	if s.vertical {
		return z.H
	}
	return z.W
}

func (s flexSpec) cross(z geom.Size) float64 {
	if s.vertical {
		return z.W
	}
	return z.H
}

// size builds a Size from a main-axis and a cross-axis length.
func (s flexSpec) size(main, cross float64) geom.Size {
	if s.vertical {
		return geom.Size{W: cross, H: main}
	}
	return geom.Size{W: main, H: cross}
}

func (s flexSpec) marginMain(m geom.Insets) float64 {
	if s.vertical {
		return m.Vertical()
	}
	return m.Horizontal()
}

func (s flexSpec) marginCross(m geom.Insets) float64 {
	if s.vertical {
		return m.Horizontal()
	}
	return m.Vertical()
}

type flexItem struct {
	w         Widget
	margin    geom.Insets
	size      geom.Size // border box
	grow      float64
	autoCross bool    // cross size is not explicit, so it may stretch
	main      float64 // outer main size, including margins (grows with flex-grow)
	cross     float64 // outer cross size, including margins
}

func newFlexItem(k Widget, spec flexSpec, availW float64) *flexItem {
	st := &k.Elem().Style
	m := st.Margin
	maxW := geom.Max(0, availW-m.Horizontal())
	var sz geom.Size
	if spec.vertical && spec.align == AlignStretch {
		sz = measureForced(k, maxW, maxW)
	} else {
		sz = measure(k, maxW)
	}
	explicit := st.Height.Unit == UnitPx
	if spec.vertical {
		explicit = st.Width.Unit != UnitAuto
	}
	return &flexItem{
		w: k, margin: m, size: sz, grow: st.FlexGrow, autoCross: !explicit,
		main:  spec.main(sz) + spec.marginMain(m),
		cross: spec.cross(sz) + spec.marginCross(m),
	}
}

func splitLines(items []*flexItem, spec flexSpec, mainAvail float64) [][]*flexItem {
	if !spec.wrap {
		return [][]*flexItem{items}
	}
	var lines [][]*flexItem
	var cur []*flexItem
	sum := 0.0
	for _, it := range items {
		next := sum + it.main
		if len(cur) > 0 {
			next += spec.gap
		}
		if len(cur) > 0 && next > mainAvail {
			lines = append(lines, cur)
			cur, next = nil, it.main
		}
		cur = append(cur, it)
		sum = next
	}
	return append(lines, cur)
}

// flexAxis is the room along one axis: a definite size, or only an upper bound.
type flexAxis struct {
	avail    float64
	definite bool
}

// used is the size the container takes: its definite size, else what the content needs.
func (a flexAxis) used(natural float64) float64 {
	if a.definite {
		return a.avail
	}
	return natural
}

// axesOf returns the main and cross axes of the container content box.
func axesOf(spec flexSpec, content geom.Rect, defW, defH bool) (main, cross flexAxis) {
	main, cross = flexAxis{content.W, defW}, flexAxis{content.H, defH}
	if spec.vertical {
		main, cross = cross, main
		if !main.definite {
			main.avail = unbounded
		}
	}
	return main, cross
}

// lineExtent is the main-axis length of a line (items and gaps) and its cross size.
func lineExtent(line []*flexItem, spec flexSpec) (main, cross float64) {
	main = spec.gap * float64(len(line)-1)
	for _, it := range line {
		main += it.main
		cross = geom.Max(cross, it.cross)
	}
	return main, cross
}

// arrangeFlex lays the children out as a flex container (a block is a vertical one).
func arrangeFlex(w Widget, content geom.Rect, defW, defH, apply bool) geom.Size {
	spec := specOf(&w.Elem().Style)
	kids := visibleChildren(w)
	if len(kids) == 0 {
		return geom.Size{}
	}
	main, cross := axesOf(spec, content, defW, defH)

	items := make([]*flexItem, len(kids))
	for i, k := range kids {
		items[i] = newFlexItem(k, spec, content.W)
	}

	lines := splitLines(items, spec, main.avail)
	usedMain, crossCursor := 0.0, 0.0
	for li, line := range lines {
		sum, lineCross := lineExtent(line, spec)
		usedMain = geom.Max(usedMain, sum)
		if len(lines) == 1 && cross.definite {
			lineCross = cross.avail
		}
		if apply {
			offset, between := spacing(line, spec, main, sum)
			placeLine(line, spec, content, offset, between, crossCursor, lineCross)
		}
		crossCursor += lineCross
		if li < len(lines)-1 {
			crossCursor += spec.gap
		}
	}
	return spec.size(main.used(usedMain), cross.used(crossCursor))
}

// spacing is the start offset and the gap between items of a line: free space
// on a definite main axis is distributed, otherwise items just keep the gap.
func spacing(line []*flexItem, spec flexSpec, main flexAxis, sum float64) (offset, between float64) {
	if main.definite && main.avail > sum {
		return distribute(line, spec, main.avail-sum)
	}
	return 0, spec.gap
}

// distribute spends free main-axis space: flex-grow first, otherwise justify-content.
// It returns the start offset and the spacing between items.
func distribute(line []*flexItem, spec flexSpec, free float64) (offset, between float64) {
	between = spec.gap
	totalGrow := 0.0
	for _, it := range line {
		totalGrow += it.grow
	}
	if totalGrow > 0 {
		for _, it := range line {
			it.main += free * it.grow / totalGrow
		}
		return 0, between
	}
	n := float64(len(line))
	switch spec.justify {
	case JustifyCenter:
		offset = free / 2
	case JustifyEnd:
		offset = free
	case JustifySpaceBetween:
		if n > 1 {
			between += free / (n - 1)
		}
	case JustifySpaceAround:
		offset, between = free/n/2, between+free/n
	}
	return offset, between
}

func placeLine(line []*flexItem, spec flexSpec, content geom.Rect, offset, between, crossStart, lineCross float64) {
	pos := offset
	for _, it := range line {
		mainSize := it.main - spec.marginMain(it.margin)
		crossSize := spec.cross(it.size)
		free := lineCross - spec.marginCross(it.margin)
		crossOff := 0.0
		switch spec.align {
		case AlignStretch:
			if it.autoCross {
				crossSize = free
			}
		case AlignCenter:
			crossOff = (free - crossSize) / 2
		case AlignEnd:
			crossOff = free - crossSize
		}
		r := geom.Rect{
			X: content.X + it.margin.Left + pos,
			Y: content.Y + it.margin.Top + crossStart + crossOff,
			W: mainSize, H: crossSize,
		}
		if spec.vertical {
			r = geom.Rect{
				X: content.X + it.margin.Left + crossStart + crossOff,
				Y: content.Y + it.margin.Top + pos,
				W: crossSize, H: mainSize,
			}
		}
		layoutWidget(it.w, r)
		pos += it.main + between
	}
}

// ---- grid ----

type gridCell struct {
	w      Widget
	margin geom.Insets
	x      float64
	size   geom.Size
	row    int
}

func resolveTracks(tracks []Track, total, gap float64) []float64 {
	fixed, fr := 0.0, 0.0
	for _, t := range tracks {
		fixed += t.Px
		fr += t.Fr
	}
	free := geom.Max(0, total-fixed-gap*float64(len(tracks)-1))
	out := make([]float64, len(tracks))
	for i, t := range tracks {
		out[i] = t.Px
		if t.Fr > 0 && fr > 0 {
			out[i] = free * t.Fr / fr
		}
	}
	return out
}

// sumRange is the total of n consecutive tracks starting at from, with gaps between them.
func sumRange(v []float64, from, n int, gap float64) float64 {
	if n <= 0 {
		return 0
	}
	s := gap * float64(n-1)
	for _, x := range v[from : from+n] {
		s += x
	}
	return s
}

func arrangeGrid(w Widget, content geom.Rect, defW, apply bool) geom.Size {
	st := &w.Elem().Style
	kids := visibleChildren(w)
	if len(kids) == 0 {
		return geom.Size{}
	}
	tracks := st.GridColumns
	if len(tracks) == 0 {
		tracks = []Track{FrTrack(1)}
	}
	colW := resolveTracks(tracks, content.W, st.Gap)

	var cells []gridCell
	var rowH []float64
	col, row := 0, 0
	for _, k := range kids {
		ks := &k.Elem().Style
		span := int(geom.Clamp(float64(ks.GridColumnSpan), 1, float64(len(tracks))))
		if col+span > len(tracks) {
			col, row = 0, row+1
		}
		cw := sumRange(colW, col, span, st.Gap)
		inner := geom.Max(0, cw-ks.Margin.Horizontal())
		sz := measureForced(k, inner, inner)
		for len(rowH) <= row {
			rowH = append(rowH, 0)
		}
		rowH[row] = geom.Max(rowH[row], sz.H+ks.Margin.Vertical())
		cells = append(cells, gridCell{k, ks.Margin, sumRange(colW, 0, col, st.Gap) + st.Gap*boolf(col > 0), sz, row})
		if col += span; col >= len(tracks) {
			col, row = 0, row+1
		}
	}

	usedH := sumRange(rowH, 0, len(rowH), st.Gap)
	if apply {
		for _, c := range cells {
			y := sumRange(rowH, 0, c.row, st.Gap) + st.Gap*boolf(c.row > 0)
			h := c.size.H
			if c.w.Elem().Style.Height.Unit != UnitPx {
				h = rowH[c.row] - c.margin.Vertical()
			}
			layoutWidget(c.w, geom.Rect{X: content.X + c.x + c.margin.Left, Y: content.Y + y + c.margin.Top, W: c.size.W, H: h})
		}
	}
	return geom.Size{W: content.W, H: usedH}
}

func boolf(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
