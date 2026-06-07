package widget

import (
	"fmt"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// TimePicker is an interactive HH:MM[:SS] time-entry widget.
//
// Visual (24-hour):   [14] : [30] : [05]
// Visual (12-hour):   [02] : [30] [PM]
//
// Default keybindings (when focused):
//   - ↑/↓      Increment/decrement the active field
//   - ←/→      Move between fields (hour ↔ minute ↔ second)
//   - 0-9      Type digits directly (two-digit entry)
//   - a/A / p/P Toggle AM/PM (12-hour mode)
type TimePicker struct {
	oat.BaseComponent
	oat.FocusBehavior
	oat.BaseHitRegion

	hour    int // 0-23
	minute  int // 0-59
	second  int // 0-59
	showSec bool
	use12h  bool

	field    int  // 0=hour, 1=minute, 2=second
	digitBuf string // one or two accumulated digit characters

	onChange func(hour, minute, second int)

	callerStyle  latte.Style
	activeStyle  latte.Style
	inactiveStyle latte.Style
	sepStyle     latte.Style
}

// NewTimePicker creates a 24-hour HH:MM TimePicker.
func NewTimePicker() *TimePicker {
	t := &TimePicker{}
	t.EnsureID()
	return t
}

// WithID sets a user-defined identifier.
func (t *TimePicker) WithID(id string) *TimePicker { t.ID = id; return t }

// With12Hour switches to 12-hour AM/PM mode.
func (t *TimePicker) With12Hour() *TimePicker { t.use12h = true; return t }

// WithSeconds enables the seconds field.
func (t *TimePicker) WithSeconds() *TimePicker { t.showSec = true; return t }

// WithValue sets the initial time.
func (t *TimePicker) WithValue(hour, minute, second int) *TimePicker {
	t.hour = clampRating(hour, 0, 23)
	t.minute = clampRating(minute, 0, 59)
	t.second = clampRating(second, 0, 59)
	return t
}

// WithOnChange registers a callback fired when the time changes.
func (t *TimePicker) WithOnChange(fn func(hour, minute, second int)) *TimePicker {
	t.onChange = fn
	return t
}

// Value returns (hour, minute, second) in 24-hour format.
func (t *TimePicker) Value() (int, int, int) { return t.hour, t.minute, t.second }

// SetValue sets the time programmatically.
func (t *TimePicker) SetValue(hour, minute, second int) {
	t.hour = clampRating(hour, 0, 23)
	t.minute = clampRating(minute, 0, 59)
	t.second = clampRating(second, 0, 59)
	t.notify()
}

// ApplyTheme wires theme colours.
func (t *TimePicker) ApplyTheme(th latte.Theme) {
	t.Style = th.Text.Merge(t.callerStyle)
	t.FocusStyle = latte.Style{BorderFG: th.FocusBorder}
	t.activeStyle = th.Accent.WithBold()
	t.inactiveStyle = th.Text
	t.sepStyle = th.Muted
}

// Measure returns the preferred size.
func (t *TimePicker) Measure(c oat.Constraint) oat.Size {
	w := t.widgetWidth()
	return c.Clamp(oat.Size{Width: w, Height: 1})
}

func (t *TimePicker) widgetWidth() int {
	// "[HH] : [MM]" = 12
	// "[HH] : [MM] : [SS]" = 19
	// "+  [AM]" = 5
	w := 12
	if t.showSec {
		w += 7
	}
	if t.use12h {
		w += 5
	}
	return w
}

// Render draws the time fields.
func (t *TimePicker) Render(buf *oat.Buffer, region oat.Region) {
	style := t.EffectiveStyle(t.IsFocused())
	sub := buf.Sub(region)
	t.SetHitRegion(sub.Region())
	sub.FillBG(style)

	h12, ampm := t.displayHour()
	fields := []struct {
		label string
		field int
	}{
		{fmt.Sprintf("[%02d]", h12), 0},
		{fmt.Sprintf("[%02d]", t.minute), 1},
	}
	if t.showSec {
		fields = append(fields, struct {
			label string
			field int
		}{fmt.Sprintf("[%02d]", t.second), 2})
	}

	x := 0
	for i, f := range fields {
		if i > 0 {
			x += sub.DrawText(x, 0, " : ", t.sepStyle)
		}
		st := t.inactiveStyle
		if f.field == t.field && t.IsFocused() {
			st = t.activeStyle
		}
		x += sub.DrawText(x, 0, f.label, st)
	}

	if t.use12h {
		sub.DrawText(x, 0, " "+ampm, t.inactiveStyle)
	}
}

// HandleKey processes keyboard navigation and digit entry.
func (t *TimePicker) HandleKey(ev *oat.KeyEvent) bool {
	switch ev.Key() {
	case tcell.KeyUp:
		t.stepField(1)
		return true
	case tcell.KeyDown:
		t.stepField(-1)
		return true
	case tcell.KeyLeft:
		t.prevField()
		return true
	case tcell.KeyRight:
		t.nextField()
		return true
	case tcell.KeyRune:
		r := ev.Rune()
		if r >= '0' && r <= '9' {
			t.typeDigit(r)
			return true
		}
		if t.use12h && (r == 'a' || r == 'A') {
			if t.hour >= 12 {
				t.hour -= 12
			}
			t.notify()
			return true
		}
		if t.use12h && (r == 'p' || r == 'P') {
			if t.hour < 12 {
				t.hour += 12
			}
			t.notify()
			return true
		}
	}
	return false
}

// KeyBindings advertises shortcuts.
func (t *TimePicker) KeyBindings() []oat.KeyBinding {
	return []oat.KeyBinding{
		{Key: tcell.KeyUp, Label: "↑↓", Description: "Adjust"},
		{Key: tcell.KeyLeft, Label: "←→", Description: "Field"},
	}
}

// Children satisfies oat.Layout.
func (t *TimePicker) Children() []oat.Component { return nil }

// AddChild satisfies oat.Layout (no-op).
func (t *TimePicker) AddChild(_ oat.Component) {}

// ── helpers ───────────────────────────────────────────────────────────────────

func (t *TimePicker) displayHour() (int, string) {
	if !t.use12h {
		return t.hour, ""
	}
	h := t.hour % 12
	if h == 0 {
		h = 12
	}
	ampm := "AM"
	if t.hour >= 12 {
		ampm = "PM"
	}
	return h, ampm
}

func (t *TimePicker) numFields() int {
	if t.showSec {
		return 3
	}
	return 2
}

func (t *TimePicker) nextField() {
	t.digitBuf = ""
	t.field = (t.field + 1) % t.numFields()
}

func (t *TimePicker) prevField() {
	t.digitBuf = ""
	t.field = (t.field - 1 + t.numFields()) % t.numFields()
}

func (t *TimePicker) stepField(delta int) {
	t.digitBuf = ""
	switch t.field {
	case 0:
		t.hour = (t.hour + delta + 24) % 24
	case 1:
		t.minute = (t.minute + delta + 60) % 60
	case 2:
		t.second = (t.second + delta + 60) % 60
	}
	t.notify()
}

func (t *TimePicker) typeDigit(r rune) {
	t.digitBuf += string(r)
	if len(t.digitBuf) == 2 {
		n := int(t.digitBuf[0]-'0')*10 + int(t.digitBuf[1]-'0')
		t.digitBuf = ""
		switch t.field {
		case 0:
			if n <= 23 {
				t.hour = n
			}
		case 1:
			if n <= 59 {
				t.minute = n
			}
		case 2:
			if n <= 59 {
				t.second = n
			}
		}
		t.nextField()
		t.notify()
	}
}

func (t *TimePicker) notify() {
	if t.onChange != nil {
		t.onChange(t.hour, t.minute, t.second)
	}
}
