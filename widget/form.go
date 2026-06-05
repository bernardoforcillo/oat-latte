package widget

import (
	"fmt"
	"regexp"
	"unicode/utf8"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// Validator is a function that returns an error message or "" if valid.
type Validator func(value string) string

// Required returns a Validator that rejects empty strings.
func Required() Validator {
	return func(value string) string {
		if value == "" {
			return "This field is required"
		}
		return ""
	}
}

// MinLength returns a Validator that rejects strings shorter than n runes.
func MinLength(n int) Validator {
	return func(value string) string {
		if utf8.RuneCountInString(value) < n {
			return fmt.Sprintf("Must be at least %d characters", n)
		}
		return ""
	}
}

// MaxLength returns a Validator that rejects strings longer than n runes.
func MaxLength(n int) Validator {
	return func(value string) string {
		if utf8.RuneCountInString(value) > n {
			return fmt.Sprintf("Must be at most %d characters", n)
		}
		return ""
	}
}

// Pattern returns a Validator that rejects strings not matching the given regexp.
// The pattern is compiled once when the validator is constructed; if the pattern
// is invalid the validator always returns an error.
func Pattern(re string) Validator {
	compiled, err := regexp.Compile(re)
	if err != nil {
		return func(_ string) string {
			return fmt.Sprintf("Invalid pattern: %s", err)
		}
	}
	return func(value string) string {
		if !compiled.MatchString(value) {
			return fmt.Sprintf("Must match pattern: %s", re)
		}
		return ""
	}
}

// ── Field ─────────────────────────────────────────────────────────────────────

// Field is a labeled input field with optional validators.
type Field struct {
	Label      string
	input      *EditText
	validators []Validator
	err        string // current validation error ("" = valid)
}

// NewField creates a Field with the given label.
func NewField(label string) *Field {
	f := &Field{
		Label: label,
		input: NewEditText(),
	}
	return f
}

// WithHint sets a persistent hint label on the inner input.
func (f *Field) WithHint(hint string) *Field {
	f.input.WithHint(hint)
	return f
}

// WithPlaceholder sets placeholder text on the inner input.
func (f *Field) WithPlaceholder(p string) *Field {
	f.input.WithPlaceholder(p)
	return f
}

// WithValidator appends one or more validators to this field.
func (f *Field) WithValidator(v ...Validator) *Field {
	f.validators = append(f.validators, v...)
	return f
}

// GetText returns the current text content of this field's input.
func (f *Field) GetText() string { return f.input.GetText() }

// SetText sets the text content of this field's input programmatically.
func (f *Field) SetText(s string) { f.input.SetText(s) }

// Validate runs all validators against the current text.
// Returns true if all validators pass; sets f.err on the first failure.
func (f *Field) Validate() bool {
	text := f.input.GetText()
	for _, v := range f.validators {
		if msg := v(text); msg != "" {
			f.err = msg
			return false
		}
	}
	f.err = ""
	return true
}

// Error returns the most recent validation error, or "" if valid.
func (f *Field) Error() string { return f.err }

// ── Form ──────────────────────────────────────────────────────────────────────

// Form is a vertical stack of labeled Fields with a submit button.
// The form itself does not directly handle focus; focus traversal is handled by
// the canvas walking through Children().
//
// Default keybindings (when the submit button is focused):
//   - Enter / Space  Submit — validate all fields and call onSubmit
type Form struct {
	oat.BaseComponent
	oat.FocusBehavior

	fields      []*Field
	submitLabel string // default "Submit"
	onSubmit    func(values map[string]string)
	hasErrors   bool

	style      latte.Style
	errorStyle latte.Style
	labelStyle latte.Style

	submitBtn *Button

	// callerStyle preserves the style set by the caller before any theme
	// application.
	callerStyle latte.Style
}

// NewForm creates an empty Form.
func NewForm() *Form {
	f := &Form{
		submitLabel: "Submit",
	}
	f.EnsureID()
	f.submitBtn = NewButton("Submit", func() { f.trySubmit() })
	return f
}

// WithID sets a user-defined identifier on this component.
func (f *Form) WithID(id string) *Form { f.ID = id; return f }

// WithSubmitLabel overrides the submit button label. Must be called before
// the first render or ApplyTheme call to take effect on the button.
func (f *Form) WithSubmitLabel(label string) *Form {
	f.submitLabel = label
	f.submitBtn = NewButton(label, func() { f.trySubmit() })
	return f
}

// WithOnSubmit registers the callback invoked when all fields are valid and
// the user presses the submit button. The map keys are Field.Label values.
func (f *Form) WithOnSubmit(fn func(map[string]string)) *Form {
	f.onSubmit = fn
	return f
}

// AddField appends a field to the form and returns the form for chaining.
func (f *Form) AddField(field *Field) *Form {
	f.fields = append(f.fields, field)
	return f
}

// Fields returns the slice of fields in this form.
func (f *Form) Fields() []*Field { return f.fields }

// ApplyTheme applies theme tokens to the form and all its children.
func (f *Form) ApplyTheme(t latte.Theme) {
	f.style = t.Text.Merge(f.callerStyle)
	f.errorStyle = t.Error
	f.labelStyle = t.Muted
	for _, field := range f.fields {
		field.input.ApplyTheme(t)
	}
	f.submitBtn.ApplyTheme(t)
}

// --- oat.Layout --------------------------------------------------------------

// Children returns all field inputs followed by the submit button.
// The canvas walks this list for focus traversal and theme propagation.
func (f *Form) Children() []oat.Component {
	children := make([]oat.Component, 0, len(f.fields)+1)
	for _, field := range f.fields {
		children = append(children, field.input)
	}
	children = append(children, f.submitBtn)
	return children
}

// AddChild is a no-op; use AddField to add fields programmatically.
func (f *Form) AddChild(_ oat.Component) {}

// --- oat.Focusable -----------------------------------------------------------

// HandleKey processes Enter on the submit button row. All other events are
// handled by the focused sub-widget (input or button) through the normal
// canvas dispatch.
func (f *Form) HandleKey(ev *oat.KeyEvent) bool {
	// Enter at the form level (no focused child) attempts submission.
	if ev.Key() == tcell.KeyEnter {
		f.trySubmit()
		return true
	}
	// Tab / Shift+Tab: do not consume — let canvas handle focus cycling.
	return false
}

// --- oat.Component -----------------------------------------------------------

func (f *Form) Measure(c oat.Constraint) oat.Size {
	w := c.MaxWidth
	if w < 0 {
		w = 40
	}
	h := 0
	for _, field := range f.fields {
		// Each field = 1 label line + inputH (3 for bordered input).
		h += 1 + 3
		if field.err != "" {
			h++ // error line
		}
	}
	// Submit button: 3 rows (bordered).
	h += 3
	return oat.Size{Width: w, Height: h}
}

func (f *Form) Render(buf *oat.Buffer, region oat.Region) {
	sub := buf.Sub(region)
	sub.FillBG(f.style)

	y := 0
	for _, field := range f.fields {
		if y >= region.Height {
			break
		}
		// Draw label above input as muted text.
		labelStyle := f.labelStyle
		if labelStyle == (latte.Style{}) {
			labelStyle = f.style
		}
		sub.DrawText(0, y, field.Label, labelStyle)
		y++

		// Render the field's input (3 rows tall = bordered single-line).
		inputH := 3
		if y+inputH > region.Height {
			inputH = region.Height - y
		}
		if inputH > 0 {
			field.input.Render(sub, oat.Region{X: 0, Y: y, Width: region.Width, Height: inputH})
			y += inputH
		}

		// Draw error line if present.
		if field.err != "" && y < region.Height {
			errStyle := f.errorStyle
			if errStyle == (latte.Style{}) {
				errStyle = latte.Style{FG: latte.ColorBrightRed, Bold: true}
			}
			sub.DrawText(0, y, "✗ "+field.err, errStyle)
			y++
		}
	}

	// Draw submit button.
	btnH := 3
	if y+btnH > region.Height {
		btnH = region.Height - y
	}
	if btnH > 0 && y < region.Height {
		f.submitBtn.Render(sub, oat.Region{X: 0, Y: y, Width: region.Width, Height: btnH})
	}
}

// --- internal ----------------------------------------------------------------

// trySubmit validates all fields; if all pass it calls onSubmit.
func (f *Form) trySubmit() {
	allValid := true
	for _, field := range f.fields {
		if !field.Validate() {
			allValid = false
		}
	}
	f.hasErrors = !allValid
	if allValid && f.onSubmit != nil {
		values := make(map[string]string, len(f.fields))
		for _, field := range f.fields {
			values[field.Label] = field.GetText()
		}
		f.onSubmit(values)
	}
}
