package widget

import (
	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// Toggle is a boolean on/off switch widget rendered as a sliding pill.
//
//	Label  ●━━━━━  ON      (on)
//	Label  ━━━━━○  off     (off)
//
// Default keybindings (when focused):
//   - Space / Enter  Toggle the value
type Toggle struct {
	oat.BaseComponent
	oat.FocusBehavior
	oat.BaseHitRegion

	value    bool
	label    string
	onChange func(bool)

	callerStyle latte.Style
	onStyle     latte.Style
	offStyle    latte.Style
	knobStyle   latte.Style
}

// NewToggle creates an off Toggle with an optional label.
func NewToggle(label string) *Toggle {
	t := &Toggle{label: label}
	t.EnsureID()
	return t
}

// WithID sets a user-defined identifier.
func (t *Toggle) WithID(id string) *Toggle { t.ID = id; return t }

// WithValue sets the initial value.
func (t *Toggle) WithValue(v bool) *Toggle { t.value = v; return t }

// WithOnChange registers a callback invoked when the value changes.
func (t *Toggle) WithOnChange(fn func(bool)) *Toggle { t.onChange = fn; return t }

// Value returns the current boolean state.
func (t *Toggle) Value() bool { return t.value }

// SetValue sets the value programmatically without firing onChange.
func (t *Toggle) SetValue(v bool) { t.value = v }

// Toggle flips the current value and fires onChange.
func (t *Toggle) Toggle() {
	t.value = !t.value
	if t.onChange != nil {
		t.onChange(t.value)
	}
}

// ApplyTheme wires theme colours.
func (t *Toggle) ApplyTheme(th latte.Theme) {
	t.Style = th.Text.Merge(t.callerStyle)
	t.FocusStyle = latte.Style{BorderFG: th.FocusBorder}
	t.onStyle = th.Success
	t.offStyle = th.Muted
	t.knobStyle = th.Accent
}

// Measure returns the preferred size.
func (t *Toggle) Measure(c oat.Constraint) oat.Size {
	w := toggleTrackWidth + 1 // track + space
	if t.label != "" {
		w += len([]rune(t.label)) + 1
	}
	return c.Clamp(oat.Size{Width: w, Height: 1})
}

const toggleTrackWidth = 8 // "━━━━●━━━" or "━━━●━━━━"

// Render draws the toggle.
func (t *Toggle) Render(buf *oat.Buffer, region oat.Region) {
	style := t.EffectiveStyle(t.IsFocused())
	sub := buf.Sub(region)
	t.SetHitRegion(sub.Region())
	sub.FillBG(style)

	x := 0
	if t.label != "" {
		x += sub.DrawText(x, 0, t.label+" ", style)
	}

	if t.value {
		// ON: ━━━━━━━● ON
		sub.DrawText(x, 0, "━━━━━━━", t.onStyle)
		sub.DrawText(x+7, 0, "●", t.knobStyle)
		sub.DrawText(x+8, 0, " ON ", t.onStyle.WithBold())
	} else {
		// OFF: ○━━━━━━━ off
		sub.DrawText(x, 0, "○", t.offStyle)
		sub.DrawText(x+1, 0, "━━━━━━━", t.offStyle)
		sub.DrawText(x+8, 0, " off", t.offStyle)
	}
}

// HandleKey processes Space and Enter to toggle.
func (t *Toggle) HandleKey(ev *oat.KeyEvent) bool {
	switch ev.Key() {
	case tcell.KeyEnter:
		t.Toggle()
		return true
	case tcell.KeyRune:
		if ev.Rune() == ' ' {
			t.Toggle()
			return true
		}
	}
	return false
}

// HandleMouse handles click anywhere on the widget.
func (t *Toggle) HandleMouse(ev *oat.MouseEvent) bool {
	if ev.Buttons()&tcell.Button1 == 0 {
		return false
	}
	mx, my := ev.Position()
	if !t.ContainsScreenPos(mx, my) {
		return false
	}
	t.Toggle()
	return true
}

// KeyBindings advertises shortcuts.
func (t *Toggle) KeyBindings() []oat.KeyBinding {
	return []oat.KeyBinding{
		{Key: tcell.KeyRune, Rune: ' ', Label: "Space", Description: "Toggle"},
	}
}

// Children satisfies oat.Layout.
func (t *Toggle) Children() []oat.Component { return nil }

// AddChild satisfies oat.Layout (no-op).
func (t *Toggle) AddChild(_ oat.Component) {}
