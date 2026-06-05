package widget

import (
	"fmt"
	"math"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// clampF returns v clamped to [lo, hi] for float64 values.
func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Slider is a horizontal range slider widget.
//
// Default keybindings (when focused):
//   - ←/→     Decrease/increase value by one step
//   - Home     Jump to minimum value
//   - End      Jump to maximum value
//   - PgUp     Increase value by 10 steps
//   - PgDn     Decrease value by 10 steps
type Slider struct {
	oat.BaseComponent
	oat.FocusBehavior
	oat.BaseHitRegion

	min, max float64
	value    float64
	step     float64

	label     string // optional label shown left of bar
	showValue bool   // show numeric value right of bar

	onChange func(float64)

	trackStyle  latte.Style // the empty track (muted)
	filledStyle latte.Style // the filled portion (accent)
	thumbStyle  latte.Style // the thumb character style
	callerStyle latte.Style

	// barScreenX and barScreenW store the bar's absolute screen coordinates
	// set during Render, so HandleMouse can compute the click ratio.
	barScreenX int
	barScreenW int
}

// NewSlider creates a horizontal slider with the given min and max values.
// Default: step=1, showValue=true.
func NewSlider(min, max float64) *Slider {
	s := &Slider{
		min:       min,
		max:       max,
		value:     min,
		step:      1,
		showValue: true,
	}
	s.EnsureID()
	return s
}

// WithID sets a user-defined identifier on this component.
func (s *Slider) WithID(id string) *Slider { s.ID = id; return s }

// WithValue sets the initial slider value, clamped to [min, max].
func (s *Slider) WithValue(v float64) *Slider { s.SetValue(v); return s }

// WithStep sets the step size for keyboard navigation.
func (s *Slider) WithStep(step float64) *Slider { s.step = step; return s }

// WithLabel sets an optional label shown to the left of the bar.
func (s *Slider) WithLabel(label string) *Slider { s.label = label; return s }

// WithShowValue controls whether the numeric value is shown to the right of the bar.
func (s *Slider) WithShowValue(show bool) *Slider { s.showValue = show; return s }

// WithOnChange registers a callback invoked when the slider value changes.
func (s *Slider) WithOnChange(fn func(float64)) *Slider { s.onChange = fn; return s }

// Value returns the current slider value.
func (s *Slider) Value() float64 { return s.value }

// GetValue implements oat.ValueGetter. Returns the current value as float64.
func (s *Slider) GetValue() interface{} { return s.value }

// SetValue sets the slider value, clamped to [min, max] and snapped to the nearest step.
func (s *Slider) SetValue(v float64) {
	s.value = s.stepSnap(v)
	if s.onChange != nil {
		s.onChange(s.value)
	}
}

// stepSnap rounds v to the nearest multiple of step within [min, max].
func (s *Slider) stepSnap(v float64) float64 {
	if s.step <= 0 {
		return clampF(v, s.min, s.max)
	}
	snapped := math.Round((v-s.min)/s.step)*s.step + s.min
	return clampF(snapped, s.min, s.max)
}

// ApplyTheme applies theme tokens to the Slider.
func (s *Slider) ApplyTheme(t latte.Theme) {
	s.Style = t.Text.Merge(s.callerStyle)
	s.FocusStyle = latte.Style{BorderFG: t.FocusBorder}
	s.trackStyle = t.Muted
	s.filledStyle = t.Accent
	s.thumbStyle = t.Accent
	s.thumbStyle.Bold = true
}

// Measure returns the desired size of the slider.
// Width: label + 1 space + bar (min 10) + 1 space + value (max 8 chars).
func (s *Slider) Measure(c oat.Constraint) oat.Size {
	labelW := 0
	if s.label != "" {
		labelW = len([]rune(s.label)) + 1
	}
	valueW := 0
	if s.showValue {
		valueW = 8
	}
	w := labelW + 10 + valueW
	return c.Clamp(oat.Size{Width: w, Height: 1})
}

// Render draws the slider into buf within the given region.
func (s *Slider) Render(buf *oat.Buffer, region oat.Region) {
	style := s.EffectiveStyle(s.IsFocused())
	sub := buf.Sub(region)
	s.SetHitRegion(sub.Region())
	sub.FillBG(style)

	x := 0

	// Draw optional label.
	if s.label != "" {
		x = sub.DrawText(x, 0, s.label+" ", style)
	}

	barX := x
	valueW := 0
	if s.showValue {
		valueW = 8
	}

	barW := region.Width - barX - valueW
	if barW < 2 {
		barW = 2
	}

	// Store absolute bar position for mouse handling.
	s.barScreenX = sub.Region().X + barX
	s.barScreenW = barW

	// Compute thumb position.
	ratio := 0.0
	if s.max > s.min {
		ratio = (s.value - s.min) / (s.max - s.min)
	}
	ratio = clampF(ratio, 0, 1)
	thumbX := barX + int(ratio*float64(barW-1))

	// Draw bar: filled, thumb, empty.
	for i := 0; i < barW; i++ {
		cx := barX + i
		switch {
		case i < thumbX-barX:
			sub.SetCell(cx, 0, '─', s.filledStyle)
		case cx == thumbX:
			sub.SetCell(cx, 0, '●', s.thumbStyle)
		default:
			sub.SetCell(cx, 0, '─', s.trackStyle)
		}
	}

	x = barX + barW

	// Draw value label.
	if s.showValue {
		sub.DrawText(x, 0, fmt.Sprintf(" %6.1f", s.value), style)
	}
}

// HandleKey processes keyboard events when the slider is focused.
func (s *Slider) HandleKey(ev *oat.KeyEvent) bool {
	switch ev.Key() {
	case tcell.KeyLeft:
		s.SetValue(s.value - s.step)
		return true
	case tcell.KeyRight:
		s.SetValue(s.value + s.step)
		return true
	case tcell.KeyHome:
		s.SetValue(s.min)
		return true
	case tcell.KeyEnd:
		s.SetValue(s.max)
		return true
	case tcell.KeyPgUp:
		s.SetValue(s.value + s.step*10)
		return true
	case tcell.KeyPgDn:
		s.SetValue(s.value - s.step*10)
		return true
	}
	return false
}

// HandleMouse processes mouse click events to set the slider value.
func (s *Slider) HandleMouse(ev *oat.MouseEvent) bool {
	if ev.Buttons()&tcell.Button1 == 0 {
		return false
	}
	if s.barScreenW <= 0 {
		return false
	}
	mx, _ := ev.Position()
	ratio := float64(mx-s.barScreenX) / float64(s.barScreenW-1)
	newVal := s.min + ratio*(s.max-s.min)
	s.SetValue(newVal)
	return true
}

// KeyBindings returns the advertised keyboard shortcuts for this slider.
func (s *Slider) KeyBindings() []oat.KeyBinding {
	return []oat.KeyBinding{
		{Key: tcell.KeyLeft, Label: "←→", Description: "Adjust"},
		{Key: tcell.KeyHome, Label: "Home", Description: "Min"},
		{Key: tcell.KeyEnd, Label: "End", Description: "Max"},
	}
}

// Children implements oat.Layout. Slider has no children.
func (s *Slider) Children() []oat.Component { return nil }

// AddChild implements oat.Layout. Slider ignores added children.
func (s *Slider) AddChild(_ oat.Component) {}
