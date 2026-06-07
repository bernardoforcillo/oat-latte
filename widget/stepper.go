package widget

import (
	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// Step is a single step in a Stepper wizard.
type Step struct {
	Label    string        // short label shown in the progress bar
	Content  oat.Component // the body component for this step
	Validate func() error  // optional; called before advancing. nil = always valid.
}

// Stepper is a multi-step wizard widget for guided flows such as onboarding,
// deploy wizards, and setup sequences.
//
// Default keybindings (when focused):
//   - → / Tab        Next step (or Finish on last step)
//   - ← / Shift+Tab  Back
//   - Esc            Cancel
type Stepper struct {
	oat.BaseComponent
	oat.FocusBehavior
	oat.BaseHitRegion

	steps  []Step
	active int // 0-indexed current step

	onFinish func() // called when user presses Finish on last step
	onCancel func() // called when user presses Esc

	// validation error from last Next attempt
	lastErr string

	// styles
	activeStepStyle latte.Style // current step label
	completedStyle  latte.Style // completed step labels
	pendingStyle    latte.Style // future step labels
	connectorStyle  latte.Style // ─── between steps
	errorStyle      latte.Style // validation error text
	btnStyle        latte.Style // Back/Next buttons
	callerStyle     latte.Style

	theme *latte.Theme
}

// NewStepper creates a Stepper with the given steps.
func NewStepper(steps ...Step) *Stepper {
	s := &Stepper{
		steps:  append([]Step{}, steps...),
		active: 0,
	}
	s.EnsureID()
	return s
}

// WithID sets a user-defined identifier on this component.
func (s *Stepper) WithID(id string) *Stepper { s.ID = id; return s }

// WithOnFinish sets the callback invoked when the user presses Finish on the last step.
func (s *Stepper) WithOnFinish(fn func()) *Stepper { s.onFinish = fn; return s }

// WithOnCancel sets the callback invoked when the user presses Esc.
func (s *Stepper) WithOnCancel(fn func()) *Stepper { s.onCancel = fn; return s }

// ActiveIndex returns the current step index (0-based).
func (s *Stepper) ActiveIndex() int { return s.active }

// IsLastStep returns true when active == len(steps)-1.
func (s *Stepper) IsLastStep() bool {
	return len(s.steps) > 0 && s.active == len(s.steps)-1
}

// Next advances to the next step. Runs Validate() first if set.
// Returns false (and sets lastErr) if validation fails.
// On the last step, calls onFinish and returns true.
func (s *Stepper) Next() bool {
	if s.active >= len(s.steps) {
		return false
	}
	step := s.steps[s.active]
	if step.Validate != nil {
		if err := step.Validate(); err != nil {
			s.lastErr = err.Error()
			return false
		}
	}
	s.lastErr = ""
	if s.active == len(s.steps)-1 {
		// Last step: finish.
		if s.onFinish != nil {
			s.onFinish()
		}
		return true
	}
	s.active++
	return true
}

// Back moves to the previous step. No-op on step 0.
func (s *Stepper) Back() {
	if s.active > 0 {
		s.active--
		s.lastErr = ""
	}
}

// ApplyTheme applies theme tokens to the Stepper and propagates the theme to
// all step content components that implement oat.ThemeReceiver.
func (s *Stepper) ApplyTheme(t latte.Theme) {
	s.theme = &t
	s.Style = t.Text.Merge(s.callerStyle)
	s.FocusStyle = latte.Style{BorderFG: t.FocusBorder}
	s.activeStepStyle = t.Accent
	s.activeStepStyle.Bold = true
	s.completedStyle = t.Text
	s.pendingStyle = t.Muted
	s.connectorStyle = t.Muted
	s.errorStyle = latte.Style{FG: tcell.ColorRed}
	s.btnStyle = t.Accent
	// Propagate theme to all step contents.
	for _, step := range s.steps {
		if tr, ok := step.Content.(oat.ThemeReceiver); ok {
			tr.ApplyTheme(t)
		}
	}
}

// Measure returns the desired size for the Stepper given the constraint.
func (s *Stepper) Measure(c oat.Constraint) oat.Size {
	progressH := 1
	btnH := 1
	errH := 0
	if s.lastErr != "" {
		errH = 1
	}

	innerH := c.MaxHeight - progressH - btnH - errH
	if innerH < 0 {
		innerH = 0
	}

	// Measure active content.
	if s.active < len(s.steps) {
		inner := oat.Constraint{MaxWidth: c.MaxWidth, MaxHeight: innerH}
		_ = s.steps[s.active].Content.Measure(inner)
	}
	return c.Clamp(oat.Size{Width: c.MaxWidth, Height: c.MaxHeight})
}

// Render draws the Stepper into buf within the given region.
// Layout rows:
//   - Row 0: progress bar
//   - Rows 1..height-btnH-errH-1: content area
//   - (Optional) row before buttons: error message in red
//   - Last row: Back/Next/Finish buttons
func (s *Stepper) Render(buf *oat.Buffer, region oat.Region) {
	style := s.EffectiveStyle(s.IsFocused())
	sub := buf.Sub(region)
	s.SetHitRegion(sub.Region())
	sub.FillBG(style)

	if region.Height == 0 {
		return
	}

	// --- Row 0: progress bar ---
	x := 0
	for i, step := range s.steps {
		var label string
		var stepStyle latte.Style
		switch {
		case i < s.active:
			label = "✓ " + step.Label
			stepStyle = s.completedStyle
		case i == s.active:
			label = "● " + step.Label
			stepStyle = s.activeStepStyle
		default:
			label = "○ " + step.Label
			stepStyle = s.pendingStyle
		}

		labelRunes := []rune(label)
		for _, r := range labelRunes {
			if x >= region.Width {
				break
			}
			sub.SetCell(x, 0, r, stepStyle)
			x++
		}

		// Draw connector between steps.
		if i < len(s.steps)-1 {
			connector := " ─── "
			for _, r := range []rune(connector) {
				if x >= region.Width {
					break
				}
				sub.SetCell(x, 0, r, s.connectorStyle)
				x++
			}
		}

		if x >= region.Width {
			break
		}
	}

	// --- Content area ---
	btnH := 1
	errH := 0
	if s.lastErr != "" {
		errH = 1
	}

	contentY := 1
	contentH := region.Height - contentY - errH - btnH
	if contentH > 0 && s.active < len(s.steps) {
		contentRegion := oat.Region{X: 0, Y: contentY, Width: region.Width, Height: contentH}
		s.steps[s.active].Content.Render(sub, contentRegion)
	}

	// --- Error line (if any) ---
	if s.lastErr != "" {
		errY := region.Height - btnH - 1
		sub.DrawText(0, errY, "⚠ "+s.lastErr, s.errorStyle)
	}

	// --- Button row (last row) ---
	btnY := region.Height - 1
	backLabel := "[ ← Back ]"
	nextLabel := "[ Next → ]"
	if s.IsLastStep() {
		nextLabel = "[ ✓ Finish ]"
	}

	// Draw Back on left (dim if on first step).
	backStyle := s.btnStyle
	if s.active == 0 {
		backStyle = s.pendingStyle
	}
	sub.DrawText(0, btnY, backLabel, backStyle)

	// Draw Next/Finish on right.
	nextX := region.Width - len([]rune(nextLabel))
	if nextX < 0 {
		nextX = 0
	}
	sub.DrawText(nextX, btnY, nextLabel, s.btnStyle)
}

// HandleKey processes key events for the Stepper.
// The active content component gets first chance to consume the event.
// Unhandled events are used to navigate between steps.
func (s *Stepper) HandleKey(ev *oat.KeyEvent) bool {
	// First, try to delegate to the active content component.
	if s.active < len(s.steps) {
		if fk, ok := s.steps[s.active].Content.(interface {
			HandleKey(*oat.KeyEvent) bool
		}); ok {
			if fk.HandleKey(ev) {
				s.lastErr = "" // clear error on interaction
				return true
			}
		}
	}

	switch ev.Key() {
	case tcell.KeyRight, tcell.KeyTab:
		s.Next()
		return true
	case tcell.KeyLeft, tcell.KeyBacktab:
		s.Back()
		return true
	case tcell.KeyEscape:
		if s.onCancel != nil {
			s.onCancel()
		}
		return false // let canvas dismiss if needed
	}
	return false
}

// KeyBindings returns the keyboard shortcuts advertised by the Stepper.
func (s *Stepper) KeyBindings() []oat.KeyBinding {
	return []oat.KeyBinding{
		{Key: tcell.KeyRight, Label: "→/Tab", Description: "Next step"},
		{Key: tcell.KeyLeft, Label: "←", Description: "Back"},
		{Key: tcell.KeyEscape, Label: "Esc", Description: "Cancel"},
	}
}

// Children implements oat.Layout. Returns the active step's content component
// so that focus walking and theme propagation descend into it.
func (s *Stepper) Children() []oat.Component {
	if s.active < len(s.steps) {
		return []oat.Component{s.steps[s.active].Content}
	}
	return nil
}

// AddChild implements oat.Layout. Appends a step with an empty label.
func (s *Stepper) AddChild(_ oat.Component) {}
