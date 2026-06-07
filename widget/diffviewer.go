package widget

import (
	"fmt"
	"strings"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// diffKind classifies a single line in a diff result.
type diffKind int

const (
	diffEqual   diffKind = iota // present in both original and modified
	diffRemoved                 // present in original only
	diffAdded                   // present in modified only
)

// diffLine represents one line in a unified diff.
type diffLine struct {
	kind   diffKind
	left   string // original line (for removed/unchanged)
	right  string // modified line (for added/unchanged)
	leftN  int    // 1-indexed line number in original (-1 if not present)
	rightN int    // 1-indexed line number in modified (-1 if not present)
}

// maxDiffLines caps the per-side line count to avoid O(n²) memory issues.
const maxDiffLines = 500

// computeDiff produces a slice of diffLines comparing original to modified
// using a simple dynamic-programming LCS algorithm.
func computeDiff(original, modified []string) []diffLine {
	// Cap inputs to avoid excessive memory usage.
	if len(original) > maxDiffLines {
		original = original[:maxDiffLines]
	}
	if len(modified) > maxDiffLines {
		modified = modified[:maxDiffLines]
	}

	m := len(original)
	n := len(modified)

	// Build LCS table. lcs[i][j] = length of LCS of original[:i] and modified[:j].
	lcs := make([][]int, m+1)
	for i := range lcs {
		lcs[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if original[i-1] == modified[j-1] {
				lcs[i][j] = lcs[i-1][j-1] + 1
			} else if lcs[i-1][j] >= lcs[i][j-1] {
				lcs[i][j] = lcs[i-1][j]
			} else {
				lcs[i][j] = lcs[i][j-1]
			}
		}
	}

	// Backtrack from lcs[m][n] to produce the edit sequence.
	var result []diffLine
	leftN := 1
	rightN := 1

	var backtrack func(i, j int)
	backtrack = func(i, j int) {
		if i == 0 && j == 0 {
			return
		}
		if i > 0 && j > 0 && original[i-1] == modified[j-1] {
			backtrack(i-1, j-1)
			result = append(result, diffLine{
				kind:   diffEqual,
				left:   original[i-1],
				right:  modified[j-1],
				leftN:  leftN,
				rightN: rightN,
			})
			leftN++
			rightN++
		} else if j > 0 && (i == 0 || lcs[i][j-1] >= lcs[i-1][j]) {
			backtrack(i, j-1)
			result = append(result, diffLine{
				kind:   diffAdded,
				left:   "",
				right:  modified[j-1],
				leftN:  -1,
				rightN: rightN,
			})
			rightN++
		} else {
			backtrack(i-1, j)
			result = append(result, diffLine{
				kind:   diffRemoved,
				left:   original[i-1],
				right:  "",
				leftN:  leftN,
				rightN: -1,
			})
			leftN++
		}
	}

	backtrack(m, n)
	return result
}

// DiffMode controls the display layout.
type DiffMode int

const (
	DiffModeUnified    DiffMode = iota // single-column, + and - prefixed
	DiffModeSideBySide                 // two-column, left=original, right=modified
)

// DiffViewer is a scrollable widget that shows a side-by-side or unified diff
// between two text strings.
//
// Default keybindings (when focused):
//   - ↑ / ↓        Scroll up / down one line
//   - PgUp / PgDn  Page up / down
//   - Home / End   Jump to first / last line
type DiffViewer struct {
	oat.BaseComponent
	oat.FocusBehavior
	oat.BaseHitRegion

	lines     []diffLine // computed diff
	scrollOff int        // first visible line
	visibleH  int        // updated each Render call
	mode      DiffMode   // unified or side-by-side

	// styles
	addedStyle   latte.Style
	removedStyle latte.Style
	normalStyle  latte.Style
	gutterStyle  latte.Style
	callerStyle  latte.Style
}

// NewDiffViewer creates a viewer comparing original to modified (line-split strings).
func NewDiffViewer(original, modified string) *DiffViewer {
	dv := &DiffViewer{
		mode: DiffModeUnified,
		addedStyle: latte.Style{
			BG: tcell.ColorDarkGreen,
			FG: tcell.ColorWhite,
		},
		removedStyle: latte.Style{
			BG: tcell.ColorDarkRed,
			FG: tcell.ColorWhite,
		},
	}
	dv.EnsureID()
	dv.SetContent(original, modified)
	return dv
}

// WithID sets a user-defined identifier on this component.
func (dv *DiffViewer) WithID(id string) *DiffViewer {
	dv.ID = id
	return dv
}

// WithMode sets the display mode (unified or side-by-side).
func (dv *DiffViewer) WithMode(mode DiffMode) *DiffViewer {
	dv.mode = mode
	return dv
}

// SetContent replaces the diff content by computing a new diff from
// the supplied original and modified strings.
func (dv *DiffViewer) SetContent(original, modified string) {
	origLines := strings.Split(original, "\n")
	modLines := strings.Split(modified, "\n")
	dv.lines = computeDiff(origLines, modLines)
	dv.scrollOff = 0
}

// ApplyTheme applies semantic theme tokens to DiffViewer styles.
func (dv *DiffViewer) ApplyTheme(t latte.Theme) {
	dv.Style = t.Text.Merge(dv.callerStyle)
	dv.FocusStyle = latte.Style{BorderFG: t.FocusBorder}
	dv.normalStyle = t.Text
	dv.gutterStyle = t.Muted
	// Fixed colors that work on both dark and light terminals.
	dv.addedStyle = latte.Style{BG: tcell.ColorDarkGreen, FG: tcell.ColorWhite}
	dv.removedStyle = latte.Style{BG: tcell.ColorDarkRed, FG: tcell.ColorWhite}
}

// Measure returns the preferred size clamped to the available constraint.
func (dv *DiffViewer) Measure(c oat.Constraint) oat.Size {
	return c.Clamp(oat.Size{Width: 40, Height: len(dv.lines)})
}

// Render draws the diff into buf within region.
func (dv *DiffViewer) Render(buf *oat.Buffer, region oat.Region) {
	sub := buf.Sub(region)
	dv.SetHitRegion(sub.Region())

	style := dv.EffectiveStyle(dv.IsFocused())
	sub.FillBG(style)

	visibleH := region.Height
	dv.visibleH = visibleH

	totalLines := len(dv.lines)
	if totalLines == 0 {
		return
	}

	// Guard scrollOff.
	if dv.scrollOff >= totalLines {
		dv.scrollOff = clamp(totalLines-1, 0, totalLines-1)
	}
	if dv.scrollOff < 0 {
		dv.scrollOff = 0
	}

	switch dv.mode {
	case DiffModeSideBySide:
		dv.renderSideBySide(sub, region, visibleH)
	default:
		dv.renderUnified(sub, region, visibleH)
	}
}

// renderUnified draws the diff in unified mode (single column, +/- prefixed).
func (dv *DiffViewer) renderUnified(sub *oat.Buffer, region oat.Region, visibleH int) {
	// Gutter width: 4-digit line number + 1 space = 5 chars.
	const gutterW = 5

	for row := 0; row < visibleH; row++ {
		lineIdx := dv.scrollOff + row
		if lineIdx >= len(dv.lines) {
			break
		}
		dl := dv.lines[lineIdx]

		var lineStyle latte.Style
		var prefix string
		var lineNum int
		var text string

		switch dl.kind {
		case diffEqual:
			lineStyle = dv.normalStyle
			prefix = "  "
			lineNum = dl.leftN
			text = dl.left
		case diffRemoved:
			lineStyle = dv.removedStyle
			prefix = "- "
			lineNum = dl.leftN
			text = dl.left
		case diffAdded:
			lineStyle = dv.addedStyle
			prefix = "+ "
			lineNum = dl.rightN
			text = dl.right
		}

		// Fill entire row with line style first for colored background.
		for x := 0; x < region.Width; x++ {
			sub.SetCell(x, row, ' ', lineStyle)
		}

		// Draw line number in gutter style.
		var gutterText string
		if lineNum > 0 {
			gutterText = fmt.Sprintf("%4d ", lineNum)
		} else {
			gutterText = "     "
		}
		x := sub.DrawText(0, row, gutterText, dv.gutterStyle)

		// Draw prefix ("+", "-", "  ").
		x = sub.DrawText(x, row, prefix, lineStyle)

		// Draw the text content, truncated to available width.
		available := region.Width - x
		if available > 0 {
			runes := []rune(text)
			if len(runes) > available {
				runes = runes[:available]
			}
			sub.DrawText(x, row, string(runes), lineStyle)
		}

		_ = gutterW
	}
}

// renderSideBySide draws the diff in side-by-side mode (two columns separated by │).
func (dv *DiffViewer) renderSideBySide(sub *oat.Buffer, region oat.Region, visibleH int) {
	totalW := region.Width
	// Divider in the middle.
	dividerX := totalW / 2
	leftW := dividerX       // left panel width (0 .. dividerX-1)
	rightW := totalW - dividerX - 1 // right panel width (dividerX+1 .. totalW-1)

	if leftW <= 0 || rightW <= 0 {
		// Too narrow — fall back to unified.
		dv.renderUnified(sub, region, visibleH)
		return
	}

	// Gutter: 4 digits + 1 space = 5 chars per side.
	const gutterW = 5

	dividerStyle := dv.gutterStyle

	for row := 0; row < visibleH; row++ {
		lineIdx := dv.scrollOff + row
		if lineIdx >= len(dv.lines) {
			// Draw divider on empty rows.
			sub.SetCell(dividerX, row, '│', dividerStyle)
			break
		}
		dl := dv.lines[lineIdx]

		// Draw divider.
		sub.SetCell(dividerX, row, '│', dividerStyle)

		// Determine left and right content.
		var leftStyle, rightStyle latte.Style
		var leftText, rightText string
		var leftNum, rightNum int

		switch dl.kind {
		case diffEqual:
			leftStyle = dv.normalStyle
			rightStyle = dv.normalStyle
			leftText = dl.left
			rightText = dl.right
			leftNum = dl.leftN
			rightNum = dl.rightN
		case diffRemoved:
			leftStyle = dv.removedStyle
			rightStyle = dv.normalStyle
			leftText = dl.left
			rightText = ""
			leftNum = dl.leftN
			rightNum = -1
		case diffAdded:
			leftStyle = dv.normalStyle
			rightStyle = dv.addedStyle
			leftText = ""
			rightText = dl.right
			leftNum = -1
			rightNum = dl.rightN
		}

		// Fill left panel background.
		for x := 0; x < leftW; x++ {
			sub.SetCell(x, row, ' ', leftStyle)
		}
		// Fill right panel background.
		for x := dividerX + 1; x < totalW; x++ {
			sub.SetCell(x, row, ' ', rightStyle)
		}

		// Draw left gutter and text.
		{
			var gutterText string
			if leftNum > 0 {
				gutterText = fmt.Sprintf("%4d ", leftNum)
			} else {
				gutterText = "     "
			}
			x := sub.DrawText(0, row, gutterText, dv.gutterStyle)
			available := leftW - x
			if available > 0 && leftText != "" {
				runes := []rune(leftText)
				if len(runes) > available {
					runes = runes[:available]
				}
				sub.DrawText(x, row, string(runes), leftStyle)
			}
		}

		// Draw right gutter and text.
		{
			var gutterText string
			if rightNum > 0 {
				gutterText = fmt.Sprintf("%4d ", rightNum)
			} else {
				gutterText = "     "
			}
			startX := dividerX + 1
			x := sub.DrawText(startX, row, gutterText, dv.gutterStyle)
			available := totalW - x
			if available > 0 && rightText != "" {
				runes := []rune(rightText)
				if len(runes) > available {
					runes = runes[:available]
				}
				sub.DrawText(x, row, string(runes), rightStyle)
			}
		}

		_ = gutterW
	}
}

// HandleKey processes keyboard input for scrolling.
func (dv *DiffViewer) HandleKey(ev *oat.KeyEvent) bool {
	visibleH := dv.visibleH
	if visibleH <= 0 {
		visibleH = 1
	}
	total := len(dv.lines)

	switch ev.Key() {
	case tcell.KeyUp:
		dv.scrollOff = clamp(dv.scrollOff-1, 0, clamp(total-1, 0, total))
		return true
	case tcell.KeyDown:
		dv.scrollOff = clamp(dv.scrollOff+1, 0, clamp(total-1, 0, total))
		return true
	case tcell.KeyPgUp:
		dv.scrollOff = clamp(dv.scrollOff-visibleH, 0, clamp(total-1, 0, total))
		return true
	case tcell.KeyPgDn:
		maxOff := clamp(total-visibleH, 0, total)
		dv.scrollOff = clamp(dv.scrollOff+visibleH, 0, maxOff)
		return true
	case tcell.KeyHome:
		dv.scrollOff = 0
		return true
	case tcell.KeyEnd:
		dv.scrollOff = clamp(total-visibleH, 0, clamp(total-1, 0, total))
		return true
	}
	return false
}

// KeyBindings returns the advertised keyboard shortcuts for the StatusBar.
func (dv *DiffViewer) KeyBindings() []oat.KeyBinding {
	return []oat.KeyBinding{
		{Key: tcell.KeyUp, Label: "↑↓", Description: "Scroll"},
		{Key: tcell.KeyPgUp, Label: "PgUp", Description: "Page"},
		{Key: tcell.KeyHome, Label: "Home", Description: "Top"},
	}
}

// Children satisfies oat.Layout; DiffViewer has no sub-components.
func (dv *DiffViewer) Children() []oat.Component { return nil }

// AddChild is a no-op for DiffViewer.
func (dv *DiffViewer) AddChild(_ oat.Component) {}
