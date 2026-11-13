package widget

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"go-pocketbook-base/internal/geom"
	"go-pocketbook-base/internal/gfx"
)

// InputType is the <input type=…> of an Input.
type InputType string

// Input types.
const (
	TypeText     InputType = "text"
	TypeNumber   InputType = "number"   // steppers (+/−), optional Mask
	TypeSearch   InputType = "search"   // clear (×) and magnifier buttons
	TypePassword InputType = "password" // text shown as bullets
)

// Input is a single-line field. Touching it opens the system keyboard (numeric
// for TypeNumber); the stepper and magnifier buttons act directly.
type Input struct {
	Base
	Type        InputType
	Value       string // for numbers: the number as text; with Mask: only the digits
	Placeholder string
	MaxLength   int
	ReadOnly    bool

	// TypeNumber
	Min, Max, Step   float64 // range is enforced when Max > Min
	Decimals         int
	Prefix, Suffix   string
	Mask             string // digit pattern, see ApplyMask
	NoSteppers       bool   // hide the − / + buttons
	DecimalSeparator string // shown instead of "." (e.g. "," for pt-BR)

	OnChange Handler
	OnSubmit Handler // search button / keyboard "OK" on a search field
}

func newInput(t InputType, placeholder string) *Input {
	in := &Input{Type: t, Placeholder: placeholder, Step: 1}
	in.Init(in)
	applyFieldStyle(&in.Style)
	return in
}

// NewInput creates a text field.
func NewInput(placeholder string) *Input { return newInput(TypeText, placeholder) }

// NewSearchInput creates a search field with clear and magnifier buttons.
func NewSearchInput(placeholder string) *Input { return newInput(TypeSearch, placeholder) }

// NewNumberInput creates a numeric field with − / + buttons.
func NewNumberInput(value float64, decimals int) *Input {
	in := newInput(TypeNumber, "")
	in.Decimals = decimals
	in.SetNumber(value)
	return in
}

// NewMaskedInput creates a numeric field that formats its digits with mask,
// e.g. "(00) 00000-0000" or "000.000.000-00".
func NewMaskedInput(mask, digits string) *Input {
	in := newInput(TypeNumber, "")
	in.Mask = mask
	in.Value = TrimToMask(mask, digits)
	return in
}

// Display is the text shown in the field.
func (in *Input) Display() string {
	switch {
	case in.Type == TypeNumber && in.Mask != "":
		return ApplyMask(in.Mask, in.Value)
	case in.Type == TypeNumber && in.Value != "":
		return in.Prefix + in.localized() + in.Suffix
	case in.Type == TypePassword:
		return strings.Repeat("•", utf8.RuneCountInString(in.Value))
	}
	return in.Value
}

// Number is the value as a number (0 when empty or invalid): JS valueAsNumber.
func (in *Input) Number() float64 {
	v, _ := strconv.ParseFloat(in.Value, 64)
	return v
}

// SetNumber stores v, clamped to [Min, Max] and rounded to Decimals.
func (in *Input) SetNumber(v float64) {
	if in.Max > in.Min {
		v = geom.Clamp(v, in.Min, in.Max)
	}
	if in.Mask != "" {
		in.Value = TrimToMask(in.Mask, strconv.FormatInt(int64(math.Round(math.Max(v, 0))), 10))
		return
	}
	scale := math.Pow10(in.Decimals)
	in.Value = strconv.FormatFloat(math.Round(v*scale)/scale, 'f', in.Decimals, 64)
}

// StepUp adds Step to the number (JS stepUp()).
func (in *Input) StepUp() { in.stepBy(1) }

// StepDown subtracts Step from the number (JS stepDown()).
func (in *Input) StepDown() { in.stepBy(-1) }

func (in *Input) stepBy(dir float64) {
	step := in.Step
	if step == 0 {
		step = 1
	}
	in.SetNumber(in.Number() + dir*step)
	Fire(in.OnChange, in)
	in.Invalidate()
}

// ---- side buttons ----

type adornment struct {
	icon    gfx.IconKind
	rect    geom.Rect
	primary bool // filled button (search)
	action  func()
}

// adornments are the square buttons at the right edge, right to left.
func (in *Input) adornments() []adornment {
	if in.ReadOnly || in.Disabled {
		return nil
	}
	r := in.Bounds.Inset(geom.All(BorderThin))
	x := r.Right()
	var out []adornment
	add := func(icon gfx.IconKind, primary bool, action func()) {
		x -= r.H
		out = append(out, adornment{icon, geom.R(x, r.Y, r.H, r.H), primary, action})
	}
	switch in.Type {
	case TypeNumber:
		if !in.NoSteppers {
			add(gfx.IconPlus, false, in.StepUp)
			add(gfx.IconMinus, false, in.StepDown)
		}
	case TypeSearch:
		add(gfx.IconSearch, true, func() { Fire(in.OnSubmit, in) })
		if in.Value != "" {
			add(gfx.IconClose, false, func() { in.Value = ""; Fire(in.OnChange, in) })
		}
	}
	return out
}

func (in *Input) textRect() geom.Rect {
	r := in.ContentRect()
	if ads := in.adornments(); len(ads) > 0 {
		r.W = geom.Max(0, ads[len(ads)-1].rect.X-8-r.X)
	}
	return r
}

// Measure implements Widget: a default field width; use Style.Width or
// FlexGrow to size it.
func (in *Input) Measure(avail float64) geom.Size {
	return geom.Size{W: geom.Min(avail, 360), H: lineHeight(FontBody)}
}

// Paint implements Widget.
func (in *Input) Paint(c *gfx.Canvas) {
	text, color := in.Display(), Ink
	if text == "" {
		text, color = in.Placeholder, Muted
	}
	ts := in.Style.text(FontBody)
	ts.Color = in.fg(color)
	c.Text(text, in.textRect(), ts)
	paintFocusRing(c, &in.Base)

	inner := in.Bounds.Inset(geom.All(BorderThin))
	for _, a := range in.adornments() {
		paintAdornment(c, a, inner, in.Style.BorderRadius-BorderThin)
	}
}

func paintAdornment(c *gfx.Canvas, a adornment, field geom.Rect, radius float64) {
	icon := geom.R(a.rect.X+(a.rect.W-36)/2, a.rect.Y+(a.rect.H-36)/2, 36, 36)
	if a.primary {
		c.Clip(field, radius) // keep the fill inside the field's rounded corners
		c.FillRoundRect(a.rect, 0, Ink)
		c.ResetClip()
		c.Icon(a.icon, icon, iconStroke, Paper)
		return
	}
	c.Line(geom.Point{X: a.rect.X, Y: a.rect.Y + 16}, geom.Point{X: a.rect.X, Y: a.rect.Bottom() - 16}, 2, Mid)
	c.Icon(a.icon, icon, iconStroke, Ink)
}

// ---- touch & keyboard ----

// HandlePointer implements PointerHandler.
func (in *Input) HandlePointer(e *PointerEvent) {
	if e.Type != PointerClick || in.ReadOnly {
		return
	}
	for _, a := range in.adornments() {
		if a.rect.Contains(geom.Point{X: e.X, Y: e.Y}) {
			a.action()
			return
		}
	}
	in.edit()
}

func (in *Input) edit() {
	if in.doc == nil {
		return
	}
	kind := KeyboardText
	if in.Type == TypeNumber {
		kind = KeyboardNumeric
	}
	in.doc.RequestText(in.self, KeyboardRequest{
		Title: titleOf(&in.Base, in.Placeholder), Initial: in.Value, Kind: kind, MaxLen: in.MaxLength,
		Done: func(text string, ok bool) {
			if ok {
				in.commit(text)
			}
		},
	})
}

// commit applies text typed on the keyboard.
func (in *Input) commit(text string) {
	if in.Type == TypeNumber {
		if in.Mask != "" {
			in.Value = TrimToMask(in.Mask, text)
		} else if v, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(text), ",", "."), 64); err == nil {
			in.SetNumber(v)
		} else {
			return // not a number: keep the previous value
		}
	} else {
		in.Value = truncateRunes(text, in.MaxLength)
	}
	Fire(in.OnChange, in)
	if in.Type == TypeSearch {
		Fire(in.OnSubmit, in)
	}
}

// titleOf picks the keyboard title: the placeholder, else the element ID.
func titleOf(b *Base, placeholder string) string {
	if placeholder != "" {
		return placeholder
	}
	return b.ID
}

func truncateRunes(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max])
}

// localized is the number text with the configured decimal separator.
func (in *Input) localized() string {
	if in.DecimalSeparator == "" {
		return in.Value
	}
	return strings.Replace(in.Value, ".", in.DecimalSeparator, 1)
}
