package widget

import (
	"fmt"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// Rating is an interactive star-rating widget.
//
//	★★★☆☆  (3/5)
//
// Default keybindings (when focused):
//   - ←/→    Decrease/increase rating
//   - Home    Set to 0
//   - End     Set to max
//   - 0-9     Set directly (clamped to max)
type Rating struct {
	oat.BaseComponent
	oat.FocusBehavior
	oat.BaseHitRegion

	value    int // 0..max
	max      int
	showNum  bool // show "(n/max)" suffix
	onChange func(int)

	callerStyle  latte.Style
	filledStyle  latte.Style
	emptyStyle   latte.Style

	lastScreenX int // absolute X of first star cell, for mouse handling
}

const (
	starFilled = '★'
	starEmpty  = '☆'
)

// NewRating creates a 5-star Rating widget.
func NewRating() *Rating {
	r := &Rating{max: 5, showNum: true}
	r.EnsureID()
	return r
}

// WithID sets a user-defined identifier.
func (r *Rating) WithID(id string) *Rating { r.ID = id; return r }

// WithMax sets the maximum number of stars (default 5).
func (r *Rating) WithMax(max int) *Rating {
	if max > 0 {
		r.max = max
		if r.value > max {
			r.value = max
		}
	}
	return r
}

// WithValue sets the initial rating value.
func (r *Rating) WithValue(v int) *Rating {
	r.value = clampRating(v, 0, r.max)
	return r
}

// WithShowNumber controls whether "(n/max)" is shown after the stars.
func (r *Rating) WithShowNumber(show bool) *Rating { r.showNum = show; return r }

// WithOnChange registers a callback invoked when the rating changes.
func (r *Rating) WithOnChange(fn func(int)) *Rating { r.onChange = fn; return r }

// Value returns the current rating.
func (r *Rating) Value() int { return r.value }

// SetValue sets the rating programmatically.
func (r *Rating) SetValue(v int) {
	r.value = clampRating(v, 0, r.max)
	if r.onChange != nil {
		r.onChange(r.value)
	}
}

// ApplyTheme wires theme colours.
func (r *Rating) ApplyTheme(t latte.Theme) {
	r.Style = t.Text.Merge(r.callerStyle)
	r.FocusStyle = latte.Style{BorderFG: t.FocusBorder}
	r.filledStyle = t.Warning // gold/amber stars
	r.emptyStyle = t.Muted
}

// Measure returns the preferred size.
func (r *Rating) Measure(c oat.Constraint) oat.Size {
	w := r.max // one char per star
	if r.showNum {
		w += len(fmt.Sprintf(" (%d/%d)", r.max, r.max)) + 1
	}
	return c.Clamp(oat.Size{Width: w, Height: 1})
}

// Render draws the stars.
func (r *Rating) Render(buf *oat.Buffer, region oat.Region) {
	style := r.EffectiveStyle(r.IsFocused())
	sub := buf.Sub(region)
	r.SetHitRegion(sub.Region())
	r.lastScreenX = sub.Region().X
	sub.FillBG(style)

	x := 0
	for i := 0; i < r.max; i++ {
		ch := starEmpty
		st := r.emptyStyle
		if i < r.value {
			ch = starFilled
			st = r.filledStyle
		}
		sub.SetCell(x, 0, ch, st)
		x++
	}
	if r.showNum {
		num := fmt.Sprintf(" (%d/%d)", r.value, r.max)
		sub.DrawText(x, 0, num, r.emptyStyle)
	}
}

// HandleKey processes keyboard input.
func (r *Rating) HandleKey(ev *oat.KeyEvent) bool {
	switch ev.Key() {
	case tcell.KeyLeft:
		r.SetValue(r.value - 1)
		return true
	case tcell.KeyRight:
		r.SetValue(r.value + 1)
		return true
	case tcell.KeyHome:
		r.SetValue(0)
		return true
	case tcell.KeyEnd:
		r.SetValue(r.max)
		return true
	case tcell.KeyRune:
		rn := ev.Rune()
		if rn >= '0' && rn <= '9' {
			r.SetValue(int(rn - '0'))
			return true
		}
	}
	return false
}

// HandleMouse handles clicks on individual stars.
func (r *Rating) HandleMouse(ev *oat.MouseEvent) bool {
	if ev.Buttons()&tcell.Button1 == 0 {
		return false
	}
	mx, my := ev.Position()
	if !r.ContainsScreenPos(mx, my) {
		return false
	}
	rx := mx - r.lastScreenX
	_ = my
	if rx >= 0 && rx < r.max {
		// Click on star rx+1.
		if r.value == rx+1 {
			r.SetValue(0) // click same star = clear
		} else {
			r.SetValue(rx + 1)
		}
		return true
	}
	return false
}

// KeyBindings advertises shortcuts.
func (r *Rating) KeyBindings() []oat.KeyBinding {
	return []oat.KeyBinding{
		{Key: tcell.KeyLeft, Label: "←→", Description: "Rate"},
		{Key: tcell.KeyHome, Label: "0-9", Description: "Set directly"},
	}
}

// Children satisfies oat.Layout.
func (r *Rating) Children() []oat.Component { return nil }

// AddChild satisfies oat.Layout (no-op).
func (r *Rating) AddChild(_ oat.Component) {}

func clampRating(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
