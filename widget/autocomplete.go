package widget

import (
	"strings"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// Autocomplete is an EditText with a fuzzy-match dropdown of suggestions.
// It is Focusable; focus state is delegated to the embedded EditText so the
// text cursor and input border behave naturally.
//
// Default keybindings (when focused and dropdown open):
//   - ↑ / ↓   Navigate suggestions
//   - Enter    Confirm selected suggestion
//   - Esc      Close dropdown without selecting
//   - Other    Delegated to the inner EditText; re-filters after each keystroke
type Autocomplete struct {
	oat.BaseComponent
	oat.FocusBehavior

	input        *EditText
	suggestions  []string
	filtered     []string
	dropdownOpen bool
	dropdownSel  int
	maxItems     int
	matchFn      func(item, query string) bool
	onSelect     func(string)

	// callerStyle is preserved across theme applications.
	callerStyle latte.Style

	// dropdownStyle and selectedStyle are derived from the theme.
	dropdownStyle latte.Style
	selectedStyle latte.Style
}

// NewAutocomplete creates an Autocomplete backed by the given list of suggestions.
// The default match function is case-insensitive strings.Contains.
func NewAutocomplete(suggestions []string) *Autocomplete {
	a := &Autocomplete{
		suggestions: suggestions,
		maxItems:    5,
		matchFn:     defaultMatchFn,
		input:       NewEditText(),
	}
	a.EnsureID()
	return a
}

// WithID sets a user-defined identifier on this component.
func (a *Autocomplete) WithID(id string) *Autocomplete { a.ID = id; return a }

// WithOnSelect registers a callback invoked when a suggestion is confirmed.
func (a *Autocomplete) WithOnSelect(fn func(string)) *Autocomplete {
	a.onSelect = fn
	return a
}

// WithMatchFn replaces the default match function.
// The function receives (item, query) and returns true when item matches.
func (a *Autocomplete) WithMatchFn(fn func(item, query string) bool) *Autocomplete {
	a.matchFn = fn
	return a
}

// WithMaxItems sets the maximum number of dropdown rows shown at once (default 5).
func (a *Autocomplete) WithMaxItems(n int) *Autocomplete {
	if n < 1 {
		n = 1
	}
	a.maxItems = n
	return a
}

// WithHint sets a persistent muted hint label above the input field.
func (a *Autocomplete) WithHint(hint string) *Autocomplete {
	a.input.WithHint(hint)
	return a
}

// WithPlaceholder sets the placeholder text shown when the input is empty.
func (a *Autocomplete) WithPlaceholder(p string) *Autocomplete {
	a.input.WithPlaceholder(p)
	return a
}

// GetText returns the current text in the input field.
func (a *Autocomplete) GetText() string { return a.input.GetText() }

// SetText replaces the input text and re-filters suggestions.
func (a *Autocomplete) SetText(s string) {
	a.input.SetText(s)
	a.reFilter()
}

// GetValue implements oat.ValueGetter. Returns the current input text.
func (a *Autocomplete) GetValue() interface{} { return a.GetText() }

// ApplyTheme applies theme tokens to the Autocomplete and its inner EditText.
func (a *Autocomplete) ApplyTheme(t latte.Theme) {
	a.Style = t.Input.Merge(a.callerStyle)
	a.FocusStyle = t.InputFocus
	a.dropdownStyle = t.Muted
	a.selectedStyle = t.ListSelected
	a.input.ApplyTheme(t)
}

// SetFocused satisfies oat.Focusable. Delegates to the inner input as well.
func (a *Autocomplete) SetFocused(focused bool) {
	a.FocusBehavior.SetFocused(focused)
	a.input.SetFocused(focused)
}

// Measure returns the size of the input plus the dropdown height when open.
func (a *Autocomplete) Measure(c oat.Constraint) oat.Size {
	inputSize := a.input.Measure(c)
	if a.dropdownOpen && len(a.filtered) > 0 {
		dropH := len(a.filtered)
		if dropH > a.maxItems {
			dropH = a.maxItems
		}
		return oat.Size{Width: inputSize.Width, Height: inputSize.Height + dropH}
	}
	return inputSize
}

// Render draws the input field and, when open, the suggestion dropdown below it.
func (a *Autocomplete) Render(buf *oat.Buffer, region oat.Region) {
	sub := buf.Sub(region)

	// Measure the input's natural height (without dropdown).
	inputSize := a.input.Measure(region.ToConstraint())
	inputH := inputSize.Height
	if inputH > region.Height {
		inputH = region.Height
	}

	// Render the input field.
	inputRegion := oat.Region{X: 0, Y: 0, Width: region.Width, Height: inputH}
	a.input.Render(sub, inputRegion)

	// Render the dropdown if open and there are matching items.
	if a.dropdownOpen && len(a.filtered) > 0 && inputH < region.Height {
		dropH := len(a.filtered)
		if dropH > a.maxItems {
			dropH = a.maxItems
		}
		if inputH+dropH > region.Height {
			dropH = region.Height - inputH
		}
		if dropH <= 0 {
			return
		}

		dropRegion := oat.Region{X: 0, Y: inputH, Width: region.Width, Height: dropH}
		dropSub := sub.Sub(dropRegion)
		dropSub.FillBG(a.dropdownStyle)

		for i := 0; i < dropH && i < len(a.filtered); i++ {
			rowStyle := a.dropdownStyle
			if i == a.dropdownSel {
				rowStyle = a.selectedStyle
			}
			// Fill the row background.
			for x := 0; x < region.Width; x++ {
				dropSub.SetCell(x, i, ' ', rowStyle)
			}
			dropSub.DrawText(1, i, a.filtered[i], rowStyle)
		}
	}
}

// HandleKey processes keyboard input for the Autocomplete.
func (a *Autocomplete) HandleKey(ev *oat.KeyEvent) bool {
	if a.dropdownOpen {
		switch ev.Key() {
		case tcell.KeyUp:
			if a.dropdownSel > 0 {
				a.dropdownSel--
			}
			return true

		case tcell.KeyDown:
			if a.dropdownSel < len(a.filtered)-1 {
				a.dropdownSel++
			}
			return true

		case tcell.KeyEnter:
			if a.dropdownSel >= 0 && a.dropdownSel < len(a.filtered) {
				chosen := a.filtered[a.dropdownSel]
				a.input.SetText(chosen)
				a.dropdownOpen = false
				if a.onSelect != nil {
					a.onSelect(chosen)
				}
			}
			return true

		case tcell.KeyEscape:
			a.dropdownOpen = false
			return true

		default:
			// Delegate to the input and re-filter afterwards.
			consumed := a.input.HandleKey(ev)
			a.reFilter()
			return consumed
		}
	}

	// Dropdown is closed: delegate to input, then re-filter.
	consumed := a.input.HandleKey(ev)
	a.reFilter()
	return consumed
}

// KeyBindings advertises available shortcuts to the StatusBar.
func (a *Autocomplete) KeyBindings() []oat.KeyBinding {
	return []oat.KeyBinding{
		{Key: tcell.KeyUp, Label: "↑", Description: "Prev"},
		{Key: tcell.KeyDown, Label: "↓", Description: "Next"},
		{Key: tcell.KeyEnter, Label: "Enter", Description: "Select"},
		{Key: tcell.KeyEscape, Label: "Esc", Description: "Close"},
	}
}

// Children satisfies oat.Layout so focus collection and theme propagation
// reach the inner EditText.
func (a *Autocomplete) Children() []oat.Component {
	return []oat.Component{a.input}
}

// AddChild is a no-op for Autocomplete.
func (a *Autocomplete) AddChild(_ oat.Component) {}

// reFilter rebuilds the filtered suggestions slice from the current input text.
func (a *Autocomplete) reFilter() {
	query := a.input.GetText()
	a.filtered = a.filtered[:0]
	for _, s := range a.suggestions {
		if a.matchFn(s, query) {
			a.filtered = append(a.filtered, s)
		}
	}
	if len(a.filtered) == 0 {
		a.dropdownOpen = false
	} else {
		a.dropdownOpen = true
		a.dropdownSel = clamp(a.dropdownSel, 0, len(a.filtered)-1)
	}
}

// defaultMatchFn reports whether item contains query (case-insensitive).
func defaultMatchFn(item, query string) bool {
	return strings.Contains(strings.ToLower(item), strings.ToLower(query))
}
