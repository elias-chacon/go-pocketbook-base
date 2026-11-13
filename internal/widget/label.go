package widget

import (
	"strings"

	"go-pocketbook-base/internal/geom"
	"go-pocketbook-base/internal/gfx"
)

// Label is static text (a <span>/<p>) that wraps to the available width.
type Label struct {
	Base
	Text string // JS: textContent
}

// NewLabel creates a label.
func NewLabel(text string) *Label {
	l := &Label{Text: text}
	l.Init(l)
	return l
}

func (l *Label) textStyle() gfx.TextStyle { return l.Style.text(FontBody) }

// lines wraps the text to width (design px).
func (l *Label) lines(width float64) []string {
	ts := l.textStyle()
	return wrapText(l.Text, width, ts.Size, ts.Bold)
}

// Measure implements Widget: the wrapped text block.
func (l *Label) Measure(avail float64) geom.Size {
	ts := l.textStyle()
	lines := l.lines(avail)
	w := 0.0
	for _, ln := range lines {
		w = geom.Max(w, gfx.TextWidth(ln, ts.Size, ts.Bold))
	}
	return geom.Size{W: w, H: lineHeight(ts.Size) * float64(len(lines))}
}

// Paint implements Widget.
func (l *Label) Paint(c *gfx.Canvas) {
	ts := l.textStyle()
	if l.Disabled {
		ts.Color = Disabled
	}
	ts.Ellipsis = false
	r := l.ContentRect()
	lh := lineHeight(ts.Size)
	for i, ln := range l.lines(r.W) {
		c.Text(ln, geom.Rect{X: r.X, Y: r.Y + float64(i)*lh, W: r.W, H: lh}, ts)
	}
}

// lineHeight is the height of one text line for a font size.
func lineHeight(size float64) float64 { return size * 1.35 }

// wrapText breaks text into lines no wider than width, honoring "\n".
func wrapText(text string, width, size float64, bold bool) []string {
	var out []string
	for _, para := range strings.Split(text, "\n") {
		line := ""
		for _, word := range strings.Fields(para) {
			cand := word
			if line != "" {
				cand = line + " " + word
			}
			if line != "" && gfx.TextWidth(cand, size, bold) > width {
				out = append(out, line)
				line = word
				continue
			}
			line = cand
		}
		out = append(out, line)
	}
	return out
}
