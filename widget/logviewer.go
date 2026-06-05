package widget

import (
	"bufio"
	"io"
	"strings"
	"sync"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// ANSISpan is a text fragment with an associated style derived from ANSI SGR
// escape sequences.
type ANSISpan struct {
	Text  string
	Style latte.Style
}

// parseANSI parses a string containing ANSI SGR escape sequences and returns
// a slice of ANSISpan values, each holding a plain-text fragment and the style
// that should be used to render it.
func parseANSI(line string) []ANSISpan {
	var spans []ANSISpan
	var cur latte.Style

	runes := []rune(line)
	n := len(runes)
	i := 0
	start := 0

	for i < n {
		// Look for ESC character.
		if runes[i] != '\033' {
			i++
			continue
		}

		// Flush any plain text before this escape sequence.
		if i > start {
			spans = append(spans, ANSISpan{Text: string(runes[start:i]), Style: cur})
		}

		// Expect '[' as the CSI introducer.
		if i+1 >= n || runes[i+1] != '[' {
			// Not a CSI sequence — skip the ESC and continue.
			i++
			start = i
			continue
		}
		i += 2 // skip ESC [

		// Read numeric parameters separated by ';', terminated by 'm'.
		var params []int
		param := -1 // -1 means "no digit seen yet"
		for i < n {
			r := runes[i]
			i++
			if r >= '0' && r <= '9' {
				if param < 0 {
					param = 0
				}
				param = param*10 + int(r-'0')
			} else if r == ';' {
				if param < 0 {
					param = 0
				}
				params = append(params, param)
				param = -1
			} else if r == 'm' {
				if param < 0 {
					param = 0
				}
				params = append(params, param)
				break
			} else {
				// Unrecognised terminator — skip and stop parsing.
				break
			}
		}

		// Apply SGR parameters to cur.
		applyParams(&cur, params)
		start = i
	}

	// Flush any remaining plain text.
	if start < n {
		spans = append(spans, ANSISpan{Text: string(runes[start:]), Style: cur})
	}

	return spans
}

// applyParams updates a Style according to a slice of SGR numeric parameters.
func applyParams(s *latte.Style, params []int) {
	for j := 0; j < len(params); j++ {
		p := params[j]
		switch {
		case p == 0:
			*s = latte.Style{}
		case p == 1:
			s.Bold = true
		case p == 3:
			s.Italic = true
		case p == 4:
			s.Underline = true
		case p == 5:
			s.Blink = true
		case p == 7:
			s.Reverse = true
		case p >= 30 && p <= 37:
			s.FG = tcell.PaletteColor(p - 30)
		case p == 38:
			// 38;5;N — 256-colour foreground.
			if j+2 < len(params) && params[j+1] == 5 {
				s.FG = tcell.PaletteColor(params[j+2])
				j += 2
			}
		case p == 39:
			s.FG = latte.ColorDefault
		case p >= 40 && p <= 47:
			s.BG = tcell.PaletteColor(p - 40)
		case p == 48:
			// 48;5;N — 256-colour background.
			if j+2 < len(params) && params[j+1] == 5 {
				s.BG = tcell.PaletteColor(params[j+2])
				j += 2
			}
		case p == 49:
			s.BG = latte.ColorDefault
		case p >= 90 && p <= 97:
			s.FG = tcell.PaletteColor(p - 90 + 8)
		case p >= 100 && p <= 107:
			s.BG = tcell.PaletteColor(p - 100 + 8)
		// anything else: ignore
		}
	}
}

// LogViewer is a streaming log viewer widget for the oat-latte TUI framework.
// It renders a scrollable, filterable view of log lines that may contain ANSI
// colour escape sequences.
//
// Default keybindings (when focused):
//   - ↑ / ↓      Scroll up / down one line
//   - PgUp / PgDn  Page up / down
//   - End          Jump to bottom and enable follow mode
//   - Ctrl+F       Open filter bar
//   - /            Open filter bar
//   - f            Toggle follow mode
type LogViewer struct {
	oat.BaseComponent
	oat.FocusBehavior
	oat.BaseHitRegion

	mu        sync.Mutex
	lines     []string
	maxLines  int
	scrollOff int
	follow    bool

	filterActive bool
	filterBuf    []rune

	normalStyle latte.Style
	dimStyle    latte.Style
	accentStyle latte.Style
}

// NewLogViewer creates a new LogViewer with follow mode enabled and a default
// maximum of 10 000 retained lines.
func NewLogViewer() *LogViewer {
	lv := &LogViewer{
		maxLines: 10000,
		follow:   true,
	}
	lv.EnsureID()
	return lv
}

// WithMaxLines sets the maximum number of lines retained in memory.
// Older lines are discarded when the limit is exceeded.
func (lv *LogViewer) WithMaxLines(n int) *LogViewer {
	lv.maxLines = n
	return lv
}

// WithFollow sets the initial follow mode.
// When follow is true the viewer scrolls to the bottom automatically as new
// lines are added.
func (lv *LogViewer) WithFollow(f bool) *LogViewer {
	lv.follow = f
	return lv
}

// WithID sets a user-defined identifier on this component.
func (lv *LogViewer) WithID(id string) *LogViewer {
	lv.ID = id
	return lv
}

// StreamFrom starts a goroutine that reads lines from r and appends them to
// the viewer.  The redraw callback is invoked after each line so that the
// enclosing application can request a screen refresh.  The goroutine exits on
// EOF or any read error.
func (lv *LogViewer) StreamFrom(r io.Reader, redraw func()) *LogViewer {
	go func() {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			lv.AddLine(scanner.Text())
			if redraw != nil {
				redraw()
			}
		}
	}()
	return lv
}

// AddLine appends a single log line in a thread-safe manner.
// When the buffer exceeds maxLines the oldest entries are removed.
func (lv *LogViewer) AddLine(line string) {
	lv.mu.Lock()
	defer lv.mu.Unlock()

	lv.lines = append(lv.lines, line)
	if lv.maxLines > 0 && len(lv.lines) > lv.maxLines {
		// Trim from the front.
		excess := len(lv.lines) - lv.maxLines
		lv.lines = lv.lines[excess:]
	}
}

// Clear removes all retained lines and resets the scroll offset.
func (lv *LogViewer) Clear() {
	lv.mu.Lock()
	defer lv.mu.Unlock()

	lv.lines = lv.lines[:0]
	lv.scrollOff = 0
}

// ApplyTheme applies semantic theme tokens to the LogViewer styles.
func (lv *LogViewer) ApplyTheme(t latte.Theme) {
	lv.Style = t.Text
	lv.FocusStyle = latte.Style{BorderFG: t.FocusBorder}
	lv.normalStyle = t.Text
	lv.dimStyle = t.Muted
	lv.accentStyle = t.Accent
}

// Measure returns the desired size clamped to the available constraint.
// LogViewer fills whatever space it is given; the caller constrains it.
func (lv *LogViewer) Measure(c oat.Constraint) oat.Size {
	return c.Clamp(oat.Size{Width: 40, Height: 10})
}

// Render draws the visible log lines into buf within region.
func (lv *LogViewer) Render(buf *oat.Buffer, region oat.Region) {
	sub := buf.Sub(region)
	lv.SetHitRegion(sub.Region())

	style := lv.EffectiveStyle(lv.IsFocused())
	sub.FillBG(style)

	// Snapshot lines under the lock, then release immediately.
	lv.mu.Lock()
	snapshot := make([]string, len(lv.lines))
	copy(snapshot, lv.lines)
	lv.mu.Unlock()

	// Build the filter string for matching.
	filter := strings.ToLower(string(lv.filterBuf))

	// Build the visible slice (filtered).
	var visible []string
	if filter == "" {
		visible = snapshot
	} else {
		for _, l := range snapshot {
			if strings.Contains(strings.ToLower(l), filter) {
				visible = append(visible, l)
			}
		}
	}

	// Reserve one row for the filter bar when active.
	filterBarH := 0
	if lv.filterActive {
		filterBarH = 1
	}
	contentH := region.Height - filterBarH
	if contentH < 0 {
		contentH = 0
	}

	maxScroll := len(visible) - contentH
	if maxScroll < 0 {
		maxScroll = 0
	}

	// Follow mode: pin to the bottom.
	if lv.follow {
		lv.scrollOff = maxScroll
	}

	// Clamp scrollOff.
	if lv.scrollOff < 0 {
		lv.scrollOff = 0
	}
	if lv.scrollOff > maxScroll {
		lv.scrollOff = maxScroll
	}

	// Draw log lines.
	zeroStyle := latte.Style{}
	for row := 0; row < contentH; row++ {
		idx := lv.scrollOff + row
		if idx >= len(visible) {
			break
		}
		spans := parseANSI(visible[idx])
		x := 0
		for _, span := range spans {
			if x >= region.Width {
				break
			}
			s := span.Style
			if s == zeroStyle {
				s = lv.normalStyle
			}
			x = sub.DrawText(x, row, span.Text, s)
		}
	}

	// Draw filter bar.
	if lv.filterActive && filterBarH > 0 {
		barY := region.Height - 1
		prompt := "/ " + string(lv.filterBuf) + "_"
		promptRunes := []rune(prompt)
		x := sub.DrawText(0, barY, string(promptRunes), lv.accentStyle)
		// Fill remainder with dimStyle.
		for ; x < region.Width; x++ {
			sub.SetCell(x, barY, ' ', lv.dimStyle)
		}
	}
}

// HandleKey processes keyboard input for scrolling and filtering.
func (lv *LogViewer) HandleKey(ev *oat.KeyEvent) bool {
	// Snapshot the line count for page calculations.
	lv.mu.Lock()
	lineCount := len(lv.lines)
	lv.mu.Unlock()

	// Approximate visible height from maxLines and filter; use lineCount as a
	// reasonable proxy.  Exact clamping happens in Render.
	visibleH := lineCount

	if lv.filterActive {
		switch ev.Key() {
		case tcell.KeyEscape:
			lv.filterBuf = lv.filterBuf[:0]
			lv.filterActive = false
			return true
		case tcell.KeyEnter:
			lv.filterActive = false
			return true
		case tcell.KeyBackspace, tcell.KeyBackspace2, tcell.KeyDelete:
			if len(lv.filterBuf) > 0 {
				lv.filterBuf = lv.filterBuf[:len(lv.filterBuf)-1]
			}
			return true
		case tcell.KeyRune:
			lv.filterBuf = append(lv.filterBuf, ev.Rune())
			return true
		}
		return false
	}

	// Normal mode.
	switch ev.Key() {
	case tcell.KeyUp:
		lv.scrollOff--
		if lv.scrollOff < 0 {
			lv.scrollOff = 0
		}
		lv.follow = false
		return true
	case tcell.KeyDown:
		lv.scrollOff++
		// Detect whether we have reached the bottom; if so re-enable follow.
		maxScroll := visibleH - 1
		if maxScroll < 0 {
			maxScroll = 0
		}
		if lv.scrollOff >= maxScroll {
			lv.follow = true
		}
		return true
	case tcell.KeyPgUp:
		lv.scrollOff -= visibleH
		if lv.scrollOff < 0 {
			lv.scrollOff = 0
		}
		lv.follow = false
		return true
	case tcell.KeyPgDn:
		lv.scrollOff += visibleH
		return true
	case tcell.KeyEnd:
		lv.follow = true
		return true
	case tcell.KeyCtrlF:
		lv.filterActive = true
		return true
	case tcell.KeyRune:
		switch ev.Rune() {
		case '/':
			lv.filterActive = true
			return true
		case 'f':
			lv.follow = !lv.follow
			return true
		}
	case tcell.KeyEscape:
		return false
	}

	return false
}

// KeyBindings returns the advertised keyboard shortcuts for the StatusBar.
func (lv *LogViewer) KeyBindings() []oat.KeyBinding {
	return []oat.KeyBinding{
		{Key: tcell.KeyUp, Label: "↑↓", Description: "Scroll"},
		{Key: tcell.KeyPgUp, Label: "PgUp", Description: "Page"},
		{Key: tcell.KeyEnd, Label: "End", Description: "Follow"},
		{Key: tcell.KeyCtrlF, Label: "Ctrl+F", Description: "Filter"},
	}
}

// Children satisfies oat.Layout; LogViewer has no sub-components.
func (lv *LogViewer) Children() []oat.Component { return nil }

// AddChild is a no-op for LogViewer.
func (lv *LogViewer) AddChild(_ oat.Component) {}
