package widget

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// NumberInput is a focusable numeric input with [−] value [+] controls.
//
// Default keybindings (when focused):
//   - ← / −   Decrement by step
//   - → / +   Increment by step
//   - Home     Set to min
//   - End      Set to max
//   - 0-9 / .  Type a value directly; Enter/Tab confirms
//   - Esc      Cancel typing, restore previous value
type NumberInput struct {
	oat.BaseComponent
	oat.FocusBehavior
	oat.BaseHitRegion

	value float64
	min   float64
	max   float64
	step  float64

	// Typing mode: user is entering a raw numeric string.
	typing  bool
	typeBuf string
	prev    float64 // value before typing started

	decimals int // number of decimal places to show (auto-derived from step)

	// Screen coordinates set during Render for mouse hit-testing.
	lastScreenX int
	lastScreenW int

	onChange func(float64)

	callerStyle latte.Style
	accentStyle latte.Style
}

// NewNumberInput creates a NumberInput with the given range.
// Step defaults to 1; decimals are derived automatically from step.
func NewNumberInput(min, max float64) *NumberInput {
	n := &NumberInput{
		min:  min,
		max:  max,
		step: 1,
	}
	n.value = min
	n.decimals = countDecimals(1)
	n.EnsureID()
	return n
}

// WithID sets a user-defined identifier.
func (n *NumberInput) WithID(id string) *NumberInput { n.ID = id; return n }

// WithValue sets the initial value (clamped to [min, max]).
func (n *NumberInput) WithValue(v float64) *NumberInput {
	n.value = clampF64(v, n.min, n.max)
	return n
}

// WithStep sets the increment/decrement step and derives decimal display precision.
func (n *NumberInput) WithStep(step float64) *NumberInput {
	if step > 0 {
		n.step = step
		n.decimals = countDecimals(step)
	}
	return n
}

// WithOnChange registers a callback invoked whenever the value changes.
func (n *NumberInput) WithOnChange(fn func(float64)) *NumberInput {
	n.onChange = fn
	return n
}

// Value returns the current numeric value.
func (n *NumberInput) Value() float64 { return n.value }

// SetValue sets the value programmatically (clamped to [min, max]).
func (n *NumberInput) SetValue(v float64) {
	n.value = clampF64(v, n.min, n.max)
	if n.onChange != nil {
		n.onChange(n.value)
	}
}

// GetValue implements oat.ValueGetter.
func (n *NumberInput) GetValue() interface{} { return n.value }

// ApplyTheme wires theme colours.
func (n *NumberInput) ApplyTheme(t latte.Theme) {
	n.Style = t.Text.Merge(n.callerStyle)
	n.FocusStyle = latte.Style{BorderFG: t.FocusBorder}
	n.accentStyle = t.Accent
}

// Measure returns the preferred size: "[−] " + value + " [+]", 1 row.
func (n *NumberInput) Measure(c oat.Constraint) oat.Size {
	valStr := n.formatValue()
	w := 4 + len(valStr) + 4 // "[−] " + value + " [+]"
	if w < 12 {
		w = 12
	}
	return c.Clamp(oat.Size{Width: w, Height: 1})
}

// Render draws "[−] 42.0 [+]" centred in the region.
func (n *NumberInput) Render(buf *oat.Buffer, region oat.Region) {
	style := n.EffectiveStyle(n.IsFocused())
	sub := buf.Sub(region)
	n.SetHitRegion(sub.Region())
	n.lastScreenX = sub.Region().X
	n.lastScreenW = sub.Region().Width
	sub.FillBG(style)

	btnStyle := n.accentStyle
	if n.IsFocused() {
		btnStyle.Bold = true
	}

	x := 0
	// Decrement button.
	x += sub.DrawText(x, 0, "[−]", btnStyle)
	x += sub.DrawText(x, 0, " ", style)

	// Value (or live typing buffer).
	var valStr string
	if n.typing {
		valStr = n.typeBuf + "▌"
	} else {
		valStr = n.formatValue()
	}
	x += sub.DrawText(x, 0, valStr, style)

	// Increment button.
	sub.DrawText(x, 0, " [+]", btnStyle)
}

// HandleKey processes keyboard input.
func (n *NumberInput) HandleKey(ev *oat.KeyEvent) bool {
	if n.typing {
		return n.handleTyping(ev)
	}
	switch ev.Key() {
	case tcell.KeyLeft:
		n.SetValue(n.value - n.step)
		return true
	case tcell.KeyRight:
		n.SetValue(n.value + n.step)
		return true
	case tcell.KeyHome:
		n.SetValue(n.min)
		return true
	case tcell.KeyEnd:
		n.SetValue(n.max)
		return true
	case tcell.KeyRune:
		r := ev.Rune()
		switch r {
		case '-', '−':
			n.SetValue(n.value - n.step)
			return true
		case '+':
			n.SetValue(n.value + n.step)
			return true
		}
		// Start typing mode for digits and '.'.
		if r >= '0' && r <= '9' || r == '.' || r == '-' {
			n.typing = true
			n.prev = n.value
			n.typeBuf = string(r)
			return true
		}
	}
	return false
}

func (n *NumberInput) handleTyping(ev *oat.KeyEvent) bool {
	switch ev.Key() {
	case tcell.KeyEnter:
		// Confirm typed value.
		if v, err := strconv.ParseFloat(strings.TrimSpace(n.typeBuf), 64); err == nil {
			n.SetValue(v)
		}
		n.typing = false
		n.typeBuf = ""
		return true
	case tcell.KeyEscape:
		// Cancel — restore previous value.
		n.value = n.prev
		n.typing = false
		n.typeBuf = ""
		return true
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(n.typeBuf) > 0 {
			runes := []rune(n.typeBuf)
			n.typeBuf = string(runes[:len(runes)-1])
		}
		return true
	case tcell.KeyRune:
		r := ev.Rune()
		if r >= '0' && r <= '9' || r == '.' || (r == '-' && len(n.typeBuf) == 0) {
			n.typeBuf += string(r)
		}
		return true
	}
	return false
}

// HandleMouse handles click on [−] and [+] buttons.
func (n *NumberInput) HandleMouse(ev *oat.MouseEvent) bool {
	if ev.Buttons()&tcell.Button1 == 0 {
		return false
	}
	mx, my := ev.Position()
	_ = my
	// Use ContainsScreenPos to verify the click is within our region.
	if !n.ContainsScreenPos(mx, my) {
		return false
	}
	// Store screen X so we can compute button positions.
	// "[−]" is at relative x 0-2; "[+]" is at relative x (width-3) to (width-1).
	// We track the region via lastScreenX set during Render.
	rx := mx - n.lastScreenX
	if rx >= 0 && rx <= 2 {
		n.SetValue(n.value - n.step)
		return true
	}
	if rx >= n.lastScreenW-3 && n.lastScreenW > 0 {
		n.SetValue(n.value + n.step)
		return true
	}
	return false
}

// KeyBindings advertises shortcuts.
func (n *NumberInput) KeyBindings() []oat.KeyBinding {
	return []oat.KeyBinding{
		{Key: tcell.KeyLeft,  Label: "←→", Description: "Adjust"},
		{Key: tcell.KeyHome,  Label: "Home", Description: "Min"},
		{Key: tcell.KeyEnd,   Label: "End", Description: "Max"},
		{Key: tcell.KeyEnter, Label: "Enter", Description: "Confirm"},
	}
}

// Children satisfies oat.Layout.
func (n *NumberInput) Children() []oat.Component { return nil }

// AddChild satisfies oat.Layout (no-op).
func (n *NumberInput) AddChild(_ oat.Component) {}

// ── helpers ───────────────────────────────────────────────────────────────────

func (n *NumberInput) formatValue() string {
	if n.decimals == 0 {
		return fmt.Sprintf("%.0f", n.value)
	}
	return fmt.Sprintf("%.*f", n.decimals, n.value)
}

func clampF64(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// countDecimals returns the number of significant decimal places in v.
func countDecimals(v float64) int {
	if v >= 1 || v <= 0 {
		return 0
	}
	d := 0
	for v < 1 {
		v *= 10
		d++
	}
	// Handle floating point imprecision (e.g. 0.1*10 = 0.9999…).
	rounded := math.Round(v)
	if math.Abs(v-rounded) < 1e-9 {
		return d
	}
	return d + 1
}
