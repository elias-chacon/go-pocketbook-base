package widget

import (
	"math"

	"go-pocketbook-base/internal/geom"
	"go-pocketbook-base/internal/gfx"
)

// Slider is a draggable range control (<input type="range">). Its size is
// configurable: Style.Width / Style.Height set the box and ThumbSize the knob;
// by default it fills the available width.
type Slider struct {
	Base
	Min, Max, Step float64
	Value          float64
	ThumbSize      float64 // knob diameter in design px
	OnInput        Handler // while dragging
	OnChange       Handler // when the finger is released
}

// NewSlider creates a slider from min to max with the given step.
func NewSlider(min, max, step, value float64) *Slider {
	s := &Slider{Min: min, Max: max, Step: step, ThumbSize: 56}
	s.init(s)
	s.SetValue(value)
	return s
}

// SetValue clamps v to the range and snaps it to Step.
func (s *Slider) SetValue(v float64) {
	if s.Step > 0 {
		v = s.Min + math.Round((v-s.Min)/s.Step)*s.Step
	}
	s.Value = geom.Clamp(v, s.Min, s.Max)
}

// fraction is the value position from 0 to 1.
func (s *Slider) fraction() float64 { return fraction(s.Value, s.Min, s.Max) }

// track is the horizontal span the knob center travels along.
func (s *Slider) track() (x0, x1, y float64) {
	r := s.ContentRect()
	half := s.ThumbSize / 2
	return r.X + half, r.Right() - half, r.Y + r.H/2
}

// Measure implements Widget: a moderate natural width (it stretches in a column
// and grows with FlexGrow in a row) and a touch area of at least MinTouch, even
// when the knob is smaller.
func (s *Slider) Measure(avail float64) geom.Size {
	return geom.Size{W: geom.Min(avail, 240), H: geom.Max(s.ThumbSize, MinTouch)}
}

// Paint implements Widget.
func (s *Slider) Paint(c *gfx.Canvas) {
	x0, x1, y := s.track()
	const thick = 14.0
	knobX := x0 + (x1-x0)*s.fraction()
	line := s.fg(Ink)
	c.FillRoundRect(geom.R(x0-thick/2, y-thick/2, x1-x0+thick, thick), thick/2, Mid)
	c.FillRoundRect(geom.R(x0-thick/2, y-thick/2, knobX-x0+thick, thick), thick/2, line)
	knob := geom.Point{X: knobX, Y: y}
	c.FillCircle(knob, s.ThumbSize/2, Paper)
	c.StrokeCircle(knob, s.ThumbSize/2, BorderThin+2, line)
	c.FillCircle(knob, s.ThumbSize/2-14, line)
}

// HandlePointer implements PointerHandler: tap or drag sets the value.
func (s *Slider) HandlePointer(e *PointerEvent) {
	switch e.Type {
	case PointerDown, PointerMove:
		s.setFromX(e.X)
		fire(s.OnInput, s)
	case PointerUp:
		s.setFromX(e.X)
		fire(s.OnChange, s)
	}
}

func (s *Slider) setFromX(x float64) {
	x0, x1, _ := s.track()
	if x1 <= x0 {
		return
	}
	s.SetValue(s.Min + (s.Max-s.Min)*geom.Clamp((x-x0)/(x1-x0), 0, 1))
	s.Invalidate()
}

// fraction maps v in [min, max] to [0, 1] (0 when the range is empty).
func fraction(v, min, max float64) float64 {
	if max <= min {
		return 0
	}
	return geom.Clamp((v-min)/(max-min), 0, 1)
}
