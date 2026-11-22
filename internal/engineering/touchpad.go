package engineering

import (
	"fmt"
	"math"
	"time"

	"go-pocketbook-base/internal/kernel/geom"
	"go-pocketbook-base/internal/toolkit/gfx"
	"go-pocketbook-base/internal/toolkit/widget"
)

// trailLength is how many points of the finger's path the pad keeps and draws.
const trailLength = 400

// TouchPad is a blank area that draws the path of the finger and counts what the touch sends: the
// events, the time between them and their rate. Each event goes to OnEvent (the log).
type TouchPad struct {
	widget.Base
	// OnEvent receives every touch event: the phase, the position and the time since the last one.
	OnEvent func(phase string, x, y float64, gap time.Duration)
	// Now is the clock (time.Now when nil).
	Now func() time.Time
	// Label is the text drawn on the empty pad.
	Label string

	trail     []geom.Point
	last      time.Time
	down      time.Time
	events    int
	maxGap    time.Duration
	moves     int
	strokeLen float64
}

// NewTouchPad creates a pad.
func NewTouchPad(label string) *TouchPad {
	p := &TouchPad{Label: label}
	p.Init(p)
	p.Style.BorderColor, p.Style.BorderWidth, p.Style.BorderRadius = widget.Ink, widget.BorderThin, widget.Radius
	p.Style.FlexGrow = 1
	p.Style.MinHeight = 400
	return p
}

func (p *TouchPad) now() time.Time {
	if p.Now != nil {
		return p.Now()
	}
	return time.Now()
}

// Measure implements widget.Widget: the pad takes the room it is given.
func (p *TouchPad) Measure(avail float64) geom.Size { return geom.Size{W: avail, H: 400} }

// HandlePointer implements widget.PointerHandler.
func (p *TouchPad) HandlePointer(e *widget.PointerEvent) {
	var phase string
	switch e.Type {
	case widget.PointerDown:
		phase = "down"
		p.down, p.events, p.moves, p.maxGap, p.strokeLen = p.now(), 0, 0, 0, 0
		p.trail = p.trail[:0]
		p.last = time.Time{}
	case widget.PointerMove:
		phase = "move"
		p.moves++
	case widget.PointerUp:
		phase = "up"
	default:
		return
	}
	now := p.now()
	var gap time.Duration
	if !p.last.IsZero() {
		gap = now.Sub(p.last)
		if gap > p.maxGap {
			p.maxGap = gap
		}
	}
	p.last = now
	p.events++
	pt := geom.Point{X: e.X, Y: e.Y}
	if n := len(p.trail); n > 0 {
		p.strokeLen += distance(p.trail[n-1], pt)
	}
	p.trail = append(p.trail, pt)
	if len(p.trail) > trailLength {
		p.trail = p.trail[len(p.trail)-trailLength:]
	}
	if p.OnEvent != nil {
		p.OnEvent(phase, e.X, e.Y, gap)
	}
	p.Invalidate()
}

func distance(a, b geom.Point) float64 {
	dx, dy := a.X-b.X, a.Y-b.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// Rate is how many events per second the last stroke sent (0 before there is a stroke).
func (p *TouchPad) Rate() float64 {
	if p.events < 2 || p.last.IsZero() {
		return 0
	}
	span := p.last.Sub(p.down).Seconds()
	if span <= 0 {
		return 0
	}
	return float64(p.events-1) / span
}

// Events is how many events the last stroke sent, and MaxGap the longest time between two of them.
func (p *TouchPad) Events() int { return p.events }

// MaxGap is the longest time between two events of the last stroke.
func (p *TouchPad) MaxGap() time.Duration { return p.maxGap }

// Reset forgets the path and the counters.
func (p *TouchPad) Reset() {
	p.trail, p.events, p.moves, p.maxGap = nil, 0, 0, 0
	p.last, p.down = time.Time{}, time.Time{}
	p.Invalidate()
}

// Paint implements widget.Widget: the path, and the counters in the corner.
func (p *TouchPad) Paint(c *gfx.Canvas) {
	r := p.ContentRect()
	for i := 1; i < len(p.trail); i++ {
		c.Line(p.trail[i-1], p.trail[i], 6, widget.Ink)
	}
	if n := len(p.trail); n > 0 {
		c.StrokeCircle(p.trail[n-1], 28, 3, widget.Ink)
	}
	text := p.Label
	if p.events > 0 {
		text = fmt.Sprintf("%d ev  %.0f ev/s  gap max %.0f ms  %.0f px", p.events, p.Rate(),
			Millis(p.maxGap), p.strokeLen)
	}
	c.Text(text, geom.R(r.X+8, r.Y+8, r.W-16, widget.FontSmall*1.5), gfx.TextStyle{
		Size: widget.FontSmall, Color: widget.Muted, Align: gfx.AlignStart,
	})
}
