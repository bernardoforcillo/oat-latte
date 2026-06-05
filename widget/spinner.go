package widget

import (
	"sync"
	"time"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
)

// SpinnerStyle selects the animation frame set used by a Spinner.
type SpinnerStyle int

const (
	SpinnerBraille SpinnerStyle = 0 // ⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏
	SpinnerDots    SpinnerStyle = 1 // ⣾⣽⣻⢿⡿⣟⣯⣷
	SpinnerArc     SpinnerStyle = 2 // ◜◠◝◞◡◟
	SpinnerLine    SpinnerStyle = 3 // -\/|
)

var spinnerFrames = map[SpinnerStyle][]rune{
	SpinnerBraille: {'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'},
	SpinnerDots:    {'⣾', '⣽', '⣻', '⢿', '⡿', '⣟', '⣯', '⣷'},
	SpinnerArc:     {'◜', '◠', '◝', '◞', '◡', '◟'},
	SpinnerLine:    {'-', '\\', '/', '|'},
}

// Spinner is an animated loading indicator. It is purely visual (not Focusable).
// Call Start to begin the animation and Stop to halt it.
type Spinner struct {
	oat.BaseComponent
	style    SpinnerStyle
	label    string
	frame    int
	running  bool
	mu       sync.Mutex
	stopCh   chan struct{}
	redrawFn func()
}

// NewSpinner creates a Spinner with the given animation style.
// redrawFn is called after each frame advance so the caller can trigger a
// screen refresh (e.g. the Canvas notify channel).
func NewSpinner(style SpinnerStyle, redrawFn func()) *Spinner {
	s := &Spinner{
		style:    style,
		redrawFn: redrawFn,
	}
	s.EnsureID()
	return s
}

// WithLabel sets an optional text label shown to the right of the spinner.
func (s *Spinner) WithLabel(label string) *Spinner { s.label = label; return s }

// WithID sets a user-defined identifier on this component.
func (s *Spinner) WithID(id string) *Spinner { s.ID = id; return s }

// ApplyTheme applies theme tokens to the Spinner.
func (s *Spinner) ApplyTheme(t latte.Theme) {
	s.Style = t.Accent
}

// Start begins the animation loop. It is idempotent — calling Start while
// already running is a no-op.
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.stopCh = make(chan struct{})
	stopCh := s.stopCh
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.mu.Lock()
				frames := spinnerFrames[s.style]
				if len(frames) > 0 {
					s.frame = (s.frame + 1) % len(frames)
				}
				s.mu.Unlock()
				if s.redrawFn != nil {
					s.redrawFn()
				}
			case <-stopCh:
				return
			}
		}
	}()
}

// Stop halts the animation loop. It is idempotent — calling Stop while not
// running is a no-op.
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	close(s.stopCh)
	s.running = false
}

// IsRunning reports whether the animation loop is active.
func (s *Spinner) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// CurrentFrame returns the rune for the current animation frame (thread-safe).
func (s *Spinner) CurrentFrame() rune {
	s.mu.Lock()
	defer s.mu.Unlock()
	frames := spinnerFrames[s.style]
	if len(frames) == 0 {
		return ' '
	}
	return frames[s.frame%len(frames)]
}

func (s *Spinner) Measure(c oat.Constraint) oat.Size {
	w := 1
	if s.label != "" {
		w += 1 + len([]rune(s.label)) // space separator + label
	}
	if c.MaxWidth >= 0 && w > c.MaxWidth {
		w = c.MaxWidth
	}
	return oat.Size{Width: w, Height: 1}
}

func (s *Spinner) Render(buf *oat.Buffer, region oat.Region) {
	sub := buf.Sub(region)
	sub.FillBG(s.Style)
	sub.SetCell(0, 0, s.CurrentFrame(), s.Style)
	if s.label != "" && region.Width > 2 {
		sub.DrawText(2, 0, s.label, s.Style)
	}
}
