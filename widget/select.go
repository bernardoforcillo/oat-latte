package widget

import (
	"strings"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// Select is a single-value picker with an inline dropdown.
//
// Default keybindings (when focused):
//   - Enter / Space  Toggle dropdown open/closed
//   - ↑ / ↓          Navigate options (when open)
//   - Home / End     Jump to first/last option (when open)
//   - Esc            Close without selecting
//
// Filtering: typing any letter while the dropdown is open filters options
// by prefix (case-insensitive); Backspace removes one filter character.
type Select struct {
	oat.BaseComponent
	oat.FocusBehavior
	oat.BaseHitRegion

	options  []string
	selected int // index into options (-1 = none)
	open     bool
	cursor   int    // highlighted row in dropdown
	filter   string // live prefix filter

	placeholder string
	maxVisible  int // max dropdown rows shown at once (default 6)
	scrollOff   int // first visible row in dropdown

	onChange func(index int, value string)

	callerStyle latte.Style
	dropStyle   latte.Style
	selStyle    latte.Style

	// Screen coords recorded during Render for mouse hit-testing.
	lastScreenX int
	lastScreenY int
	lastScreenW int
}

// NewSelect creates a Select with the given option list.
func NewSelect(options ...string) *Select {
	s := &Select{
		options:    options,
		selected:   -1,
		maxVisible: 6,
	}
	s.EnsureID()
	return s
}

// WithID sets a user-defined identifier.
func (s *Select) WithID(id string) *Select { s.ID = id; return s }

// WithPlaceholder sets the text shown when no option is selected.
func (s *Select) WithPlaceholder(p string) *Select { s.placeholder = p; return s }

// WithOptions replaces the full option list.
func (s *Select) WithOptions(opts ...string) *Select {
	s.options = opts
	if s.selected >= len(opts) {
		s.selected = -1
	}
	return s
}

// WithValue selects the option with the given index (0-based).
func (s *Select) WithValue(index int) *Select {
	if index >= 0 && index < len(s.options) {
		s.selected = index
	}
	return s
}

// WithValueString selects the first option whose text equals v.
func (s *Select) WithValueString(v string) *Select {
	for i, o := range s.options {
		if o == v {
			s.selected = i
			return s
		}
	}
	return s
}

// WithMaxVisible sets how many rows the dropdown shows before scrolling.
func (s *Select) WithMaxVisible(n int) *Select {
	if n > 0 {
		s.maxVisible = n
	}
	return s
}

// WithOnChange registers a callback called when the selection changes.
func (s *Select) WithOnChange(fn func(index int, value string)) *Select {
	s.onChange = fn
	return s
}

// Value returns the currently selected index (-1 = none).
func (s *Select) Value() int { return s.selected }

// ValueString returns the selected option text, or "" if none is selected.
func (s *Select) ValueString() string {
	if s.selected >= 0 && s.selected < len(s.options) {
		return s.options[s.selected]
	}
	return ""
}

// SetValue selects by index programmatically.
func (s *Select) SetValue(index int) {
	if index < 0 || index >= len(s.options) {
		return
	}
	s.selected = index
	if s.onChange != nil {
		s.onChange(index, s.options[index])
	}
}

// ApplyTheme wires theme colours.
func (s *Select) ApplyTheme(t latte.Theme) {
	s.Style = t.Text.Merge(s.callerStyle)
	s.FocusStyle = latte.Style{BorderFG: t.FocusBorder}
	s.dropStyle = t.Panel
	s.selStyle = t.Accent
}

// Measure returns the size: one row for the control, plus visible rows when open.
func (s *Select) Measure(c oat.Constraint) oat.Size {
	w := s.controlWidth()
	if w < 16 {
		w = 16
	}
	h := 1
	if s.open {
		h += s.visibleCount()
	}
	return c.Clamp(oat.Size{Width: w, Height: h})
}

// Render draws the select button and (when open) the dropdown.
func (s *Select) Render(buf *oat.Buffer, region oat.Region) {
	style := s.EffectiveStyle(s.IsFocused())
	sub := buf.Sub(region)
	s.SetHitRegion(sub.Region())
	s.lastScreenX = sub.Region().X
	s.lastScreenY = sub.Region().Y
	s.lastScreenW = sub.Region().Width

	// Draw the control row.
	sub.FillBG(style)
	label := s.controlLabel()
	runes := []rune(label)
	if len(runes) > region.Width {
		runes = runes[:region.Width]
	}
	for len(runes) < region.Width {
		runes = append(runes, ' ')
	}
	sub.DrawText(0, 0, string(runes), style)

	if !s.open || region.Height < 2 {
		return
	}

	// Draw dropdown rows.
	visible := s.visibleCount()
	filtered := s.filteredOptions()
	for row := 0; row < visible && row < region.Height-1; row++ {
		idx := s.scrollOff + row
		if idx >= len(filtered) {
			break
		}
		opt := filtered[idx].text
		runes := []rune(" " + opt)
		if len(runes) > region.Width {
			runes = runes[:region.Width]
		}
		for len(runes) < region.Width {
			runes = append(runes, ' ')
		}
		rowStyle := s.dropStyle
		if s.scrollOff+row == s.cursor {
			rowStyle = s.selStyle
		}
		sub.DrawText(0, 1+row, string(runes), rowStyle)
	}
}

// HandleKey processes keyboard input.
func (s *Select) HandleKey(ev *oat.KeyEvent) bool {
	if !s.open {
		switch ev.Key() {
		case tcell.KeyEnter:
			s.openDropdown()
			return true
		case tcell.KeyRune:
			if ev.Rune() == ' ' {
				s.openDropdown()
				return true
			}
		}
		return false
	}

	// Dropdown is open.
	switch ev.Key() {
	case tcell.KeyEscape:
		s.closeDropdown()
		return true
	case tcell.KeyEnter:
		s.confirm()
		return true
	case tcell.KeyUp:
		if s.cursor > 0 {
			s.cursor--
			s.ensureVisible()
		}
		return true
	case tcell.KeyDown:
		filtered := s.filteredOptions()
		if s.cursor < len(filtered)-1 {
			s.cursor++
			s.ensureVisible()
		}
		return true
	case tcell.KeyHome:
		s.cursor = 0
		s.scrollOff = 0
		return true
	case tcell.KeyEnd:
		filtered := s.filteredOptions()
		if len(filtered) > 0 {
			s.cursor = len(filtered) - 1
			s.ensureVisible()
		}
		return true
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(s.filter) > 0 {
			runes := []rune(s.filter)
			s.filter = string(runes[:len(runes)-1])
			s.cursor = 0
			s.scrollOff = 0
		}
		return true
	case tcell.KeyRune:
		s.filter += string(ev.Rune())
		s.cursor = 0
		s.scrollOff = 0
		return true
	}
	return false
}

// HandleMouse handles clicks on the control and dropdown rows.
func (s *Select) HandleMouse(ev *oat.MouseEvent) bool {
	if ev.Buttons()&tcell.Button1 == 0 {
		return false
	}
	mx, my := ev.Position()
	if !s.ContainsScreenPos(mx, my) {
		if s.open {
			s.closeDropdown()
		}
		return false
	}
	ry := my - s.lastScreenY
	if ry == 0 {
		// Click on the control bar.
		if s.open {
			s.closeDropdown()
		} else {
			s.openDropdown()
		}
		return true
	}
	if s.open && ry >= 1 {
		row := ry - 1
		idx := s.scrollOff + row
		filtered := s.filteredOptions()
		if idx < len(filtered) {
			s.cursor = idx
			s.confirm()
			return true
		}
	}
	return false
}

// KeyBindings advertises shortcuts.
func (s *Select) KeyBindings() []oat.KeyBinding {
	return []oat.KeyBinding{
		{Key: tcell.KeyEnter, Label: "Enter", Description: "Open"},
		{Key: tcell.KeyUp, Label: "↑↓", Description: "Navigate"},
	}
}

// Children satisfies oat.Layout.
func (s *Select) Children() []oat.Component { return nil }

// AddChild satisfies oat.Layout (no-op).
func (s *Select) AddChild(_ oat.Component) {}

// ── internal ──────────────────────────────────────────────────────────────────

type filteredOption struct {
	text         string
	originalIndex int
}

func (s *Select) filteredOptions() []filteredOption {
	if s.filter == "" {
		out := make([]filteredOption, len(s.options))
		for i, o := range s.options {
			out[i] = filteredOption{text: o, originalIndex: i}
		}
		return out
	}
	low := strings.ToLower(s.filter)
	var out []filteredOption
	for i, o := range s.options {
		if strings.Contains(strings.ToLower(o), low) {
			out = append(out, filteredOption{text: o, originalIndex: i})
		}
	}
	return out
}

func (s *Select) visibleCount() int {
	filtered := s.filteredOptions()
	n := len(filtered)
	if n > s.maxVisible {
		n = s.maxVisible
	}
	return n
}

func (s *Select) controlLabel() string {
	arrow := "▼"
	if s.open {
		arrow = "▲"
	}
	var label string
	if s.filter != "" {
		label = s.filter + "▌"
	} else if s.selected >= 0 && s.selected < len(s.options) {
		label = s.options[s.selected]
	} else {
		label = s.placeholder
	}
	return " " + label + " " + arrow + " "
}

func (s *Select) controlWidth() int {
	max := len([]rune(s.placeholder)) + 4
	for _, o := range s.options {
		if l := len([]rune(o)) + 4; l > max {
			max = l
		}
	}
	return max
}

func (s *Select) openDropdown() {
	s.open = true
	s.filter = ""
	s.cursor = 0
	if s.selected >= 0 {
		s.cursor = s.selected
	}
	s.scrollOff = 0
	s.ensureVisible()
}

func (s *Select) closeDropdown() {
	s.open = false
	s.filter = ""
}

func (s *Select) confirm() {
	filtered := s.filteredOptions()
	if s.cursor >= 0 && s.cursor < len(filtered) {
		orig := filtered[s.cursor].originalIndex
		s.selected = orig
		if s.onChange != nil {
			s.onChange(orig, s.options[orig])
		}
	}
	s.closeDropdown()
}

func (s *Select) ensureVisible() {
	if s.cursor < s.scrollOff {
		s.scrollOff = s.cursor
	}
	if s.cursor >= s.scrollOff+s.maxVisible {
		s.scrollOff = s.cursor - s.maxVisible + 1
	}
}
