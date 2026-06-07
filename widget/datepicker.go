package widget

import (
	"fmt"
	"time"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// DatePicker is a calendar widget for picking a date.
//
// Default keybindings (when focused):
//   - ←/→          Previous/next day
//   - ↑/↓          Previous/next week
//   - PgUp/PgDn    Previous/next month
//   - Ctrl+PgUp/Dn Previous/next year
//   - Home          First day of month
//   - End           Last day of month
//   - Enter         Confirm selection and fire onChange
//   - t / T         Jump to today
type DatePicker struct {
	oat.BaseComponent
	oat.FocusBehavior
	oat.BaseHitRegion

	current  time.Time // the month/year being displayed
	selected time.Time // the actively selected date (zero = none)
	hasValue bool

	onChange func(time.Time)

	callerStyle   latte.Style
	headerStyle   latte.Style
	weekdayStyle  latte.Style
	dayStyle      latte.Style
	todayStyle    latte.Style
	selectedStyle latte.Style
	otherStyle    latte.Style // days outside current month

	// Mouse hit-testing.
	lastScreenX int
	lastScreenY int
}

const (
	dpWidth  = 22 // "Su Mo Tu We Th Fr Sa" + borders
	dpHeight = 9  // header + weekdays + 6 weeks
)

var weekdays = [7]string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}

// NewDatePicker creates a DatePicker showing the current month.
func NewDatePicker() *DatePicker {
	now := time.Now()
	dp := &DatePicker{
		current:  time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local),
		selected: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local),
		hasValue: false,
	}
	dp.EnsureID()
	return dp
}

// WithID sets a user-defined identifier.
func (dp *DatePicker) WithID(id string) *DatePicker { dp.ID = id; return dp }

// WithValue sets the initially selected date.
func (dp *DatePicker) WithValue(t time.Time) *DatePicker {
	dp.selected = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
	dp.current = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local)
	dp.hasValue = true
	return dp
}

// WithOnChange registers a callback fired when the user confirms a date with Enter.
func (dp *DatePicker) WithOnChange(fn func(time.Time)) *DatePicker {
	dp.onChange = fn
	return dp
}

// Value returns the selected date and whether a date has been confirmed.
func (dp *DatePicker) Value() (time.Time, bool) { return dp.selected, dp.hasValue }

// SetValue selects a date programmatically without firing onChange.
func (dp *DatePicker) SetValue(t time.Time) {
	dp.selected = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
	dp.current = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local)
	dp.hasValue = true
}

// ApplyTheme wires theme colours.
func (dp *DatePicker) ApplyTheme(t latte.Theme) {
	dp.Style = t.Text.Merge(dp.callerStyle)
	dp.FocusStyle = latte.Style{BorderFG: t.FocusBorder}
	dp.headerStyle = t.Accent.WithBold()
	dp.weekdayStyle = t.Muted
	dp.dayStyle = t.Text
	dp.todayStyle = t.Warning
	dp.selectedStyle = t.Accent
	dp.otherStyle = t.Muted
}

// Measure returns the fixed calendar size.
func (dp *DatePicker) Measure(c oat.Constraint) oat.Size {
	return c.Clamp(oat.Size{Width: dpWidth, Height: dpHeight})
}

// Render draws the calendar.
func (dp *DatePicker) Render(buf *oat.Buffer, region oat.Region) {
	style := dp.EffectiveStyle(dp.IsFocused())
	sub := buf.Sub(region)
	dp.SetHitRegion(sub.Region())
	dp.lastScreenX = sub.Region().X
	dp.lastScreenY = sub.Region().Y
	sub.FillBG(style)

	today := time.Now()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)

	// Row 0: "◀ January 2025 ▶" header.
	header := fmt.Sprintf("◀ %s %d ▶", dp.current.Month().String()[:3], dp.current.Year())
	// Centre in dpWidth.
	pad := (dpWidth - len([]rune(header))) / 2
	if pad < 0 {
		pad = 0
	}
	sub.DrawText(pad, 0, header, dp.headerStyle)

	// Row 1: weekday names.
	for col, wd := range weekdays {
		x := col * 3
		sub.DrawText(x, 1, wd, dp.weekdayStyle)
	}

	// Rows 2-7: day grid.
	// Find which weekday the 1st falls on.
	firstWeekday := int(dp.current.Weekday()) // 0=Sun
	month := dp.current.Month()
	year := dp.current.Year()

	day := 1
	totalDays := daysInMonth(year, month)

	for row := 0; row < 6; row++ {
		for col := 0; col < 7; col++ {
			cell := row*7 + col
			d := cell - firstWeekday + 1
			x := col * 3
			y := 2 + row

			if d < 1 || d > totalDays {
				// Show prev/next month days dimmed.
				var ghost int
				if d < 1 {
					prev := prevMonth(year, month)
					ghost = daysInMonth(prev.Year(), prev.Month()) + d
				} else {
					ghost = d - totalDays
				}
				sub.DrawText(x, y, fmt.Sprintf("%2d", ghost), dp.otherStyle)
				continue
			}

			date := time.Date(year, month, d, 0, 0, 0, 0, time.Local)
			dayStyle := dp.dayStyle
			label := fmt.Sprintf("%2d", d)

			if date.Equal(today) {
				dayStyle = dp.todayStyle
			}
			if dp.hasValue && date.Equal(dp.selected) {
				dayStyle = dp.selectedStyle
				label = fmt.Sprintf("[%d]", d)
				if d < 10 {
					label = fmt.Sprintf("[%d]", d) // "[9]" is 3 chars — fine
				}
			}
			if day == d {
				_ = day
			}

			runes := []rune(label)
			if len(runes) > 2 {
				runes = runes[:2]
			}
			sub.DrawText(x, y, string(runes), dayStyle)
			day++
		}
	}
}

// HandleKey processes navigation.
func (dp *DatePicker) HandleKey(ev *oat.KeyEvent) bool {
	switch ev.Key() {
	case tcell.KeyLeft:
		dp.moveDay(-1)
		return true
	case tcell.KeyRight:
		dp.moveDay(1)
		return true
	case tcell.KeyUp:
		dp.moveDay(-7)
		return true
	case tcell.KeyDown:
		dp.moveDay(7)
		return true
	case tcell.KeyPgUp:
		if ev.Modifiers()&tcell.ModCtrl != 0 {
			dp.moveYear(-1)
		} else {
			dp.moveMonth(-1)
		}
		return true
	case tcell.KeyPgDn:
		if ev.Modifiers()&tcell.ModCtrl != 0 {
			dp.moveYear(1)
		} else {
			dp.moveMonth(1)
		}
		return true
	case tcell.KeyHome:
		dp.selected = time.Date(dp.current.Year(), dp.current.Month(), 1, 0, 0, 0, 0, time.Local)
		return true
	case tcell.KeyEnd:
		last := daysInMonth(dp.current.Year(), dp.current.Month())
		dp.selected = time.Date(dp.current.Year(), dp.current.Month(), last, 0, 0, 0, 0, time.Local)
		return true
	case tcell.KeyEnter:
		dp.hasValue = true
		if dp.onChange != nil {
			dp.onChange(dp.selected)
		}
		return true
	case tcell.KeyRune:
		if ev.Rune() == 't' || ev.Rune() == 'T' {
			now := time.Now()
			dp.selected = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
			dp.current = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
			return true
		}
	}
	return false
}

// HandleMouse handles clicks on the nav arrows and day cells.
func (dp *DatePicker) HandleMouse(ev *oat.MouseEvent) bool {
	if ev.Buttons()&tcell.Button1 == 0 {
		return false
	}
	mx, my := ev.Position()
	if !dp.ContainsScreenPos(mx, my) {
		return false
	}
	rx := mx - dp.lastScreenX
	ry := my - dp.lastScreenY

	// Header row: click ◀ or ▶.
	if ry == 0 {
		if rx == 0 {
			dp.moveMonth(-1)
			return true
		}
		if rx == dpWidth-1 {
			dp.moveMonth(1)
			return true
		}
		return false
	}

	// Day grid (rows 2-7).
	if ry >= 2 {
		col := rx / 3
		row := ry - 2
		if col > 6 || row > 5 {
			return false
		}
		firstWeekday := int(dp.current.Weekday())
		d := row*7 + col - firstWeekday + 1
		totalDays := daysInMonth(dp.current.Year(), dp.current.Month())
		if d >= 1 && d <= totalDays {
			dp.selected = time.Date(dp.current.Year(), dp.current.Month(), d, 0, 0, 0, 0, time.Local)
			dp.hasValue = true
			if dp.onChange != nil {
				dp.onChange(dp.selected)
			}
			return true
		}
	}
	return false
}

// KeyBindings advertises shortcuts.
func (dp *DatePicker) KeyBindings() []oat.KeyBinding {
	return []oat.KeyBinding{
		{Key: tcell.KeyLeft, Label: "←→↑↓", Description: "Navigate"},
		{Key: tcell.KeyPgUp, Label: "PgUp/Dn", Description: "Month"},
		{Key: tcell.KeyEnter, Label: "Enter", Description: "Select"},
		{Key: tcell.KeyRune, Rune: 't', Label: "t", Description: "Today"},
	}
}

// Children satisfies oat.Layout.
func (dp *DatePicker) Children() []oat.Component { return nil }

// AddChild satisfies oat.Layout (no-op).
func (dp *DatePicker) AddChild(_ oat.Component) {}

// ── helpers ───────────────────────────────────────────────────────────────────

func (dp *DatePicker) moveDay(delta int) {
	if dp.selected.IsZero() {
		dp.selected = dp.current
	}
	dp.selected = dp.selected.AddDate(0, 0, delta)
	dp.current = time.Date(dp.selected.Year(), dp.selected.Month(), 1, 0, 0, 0, 0, time.Local)
}

func (dp *DatePicker) moveMonth(delta int) {
	dp.current = dp.current.AddDate(0, delta, 0)
	// Keep selected in view if it was already set.
	if !dp.selected.IsZero() {
		dp.selected = dp.selected.AddDate(0, delta, 0)
	}
}

func (dp *DatePicker) moveYear(delta int) {
	dp.current = dp.current.AddDate(delta, 0, 0)
	if !dp.selected.IsZero() {
		dp.selected = dp.selected.AddDate(delta, 0, 0)
	}
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local).Day()
}

func prevMonth(year int, month time.Month) time.Time {
	return time.Date(year, month, 1, 0, 0, 0, 0, time.Local).AddDate(0, -1, 0)
}
