package widget

import (
	"strings"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// Command is a single entry in a CommandPalette.
type Command struct {
	Name        string
	Description string
	Handler     func()
}

// CommandPalette is a Ctrl+P-style command palette overlay.
// It is meant to be shown as a modal overlay via canvas.ShowDialog().
//
// The palette renders a rounded-border container near the top-third of
// the screen, containing a text input for filtering and a scrollable list
// of matching commands.
//
// Default keybindings (when focused):
//   - ↑ / ↓      Navigate — move selection up/down
//   - Enter      Execute — run the selected command's handler
//   - Esc        (not consumed) — lets the canvas dismiss the dialog
//   - Any other  Delegate to the inner input and refilter
type CommandPalette struct {
	oat.BaseComponent
	oat.FocusBehavior

	commands   []Command
	filtered   []Command
	input      *EditText
	selected   int
	maxVisible int // default 8

	style       latte.Style // derived from t.Dialog
	titleStyle  latte.Style // derived from t.DialogTitle
	textStyle   latte.Style // derived from t.Text
	selectedStyle latte.Style // derived from t.ListSelected
}

// NewCommandPalette creates a CommandPalette pre-populated with the given commands.
func NewCommandPalette(commands []Command) *CommandPalette {
	cp := &CommandPalette{
		commands:   append([]Command{}, commands...),
		maxVisible: 8,
	}
	cp.EnsureID()

	cp.input = NewEditText().
		WithPlaceholder("Type to filter commands…")
	cp.input.WithOnChange(func(_ string) {
		cp.refilter()
	})
	cp.refilter()
	return cp
}

// WithID sets a user-defined identifier on this component.
func (cp *CommandPalette) WithID(id string) *CommandPalette { cp.ID = id; return cp }

// AddCommand appends a command to the palette and re-runs the current filter.
func (cp *CommandPalette) AddCommand(cmd Command) {
	cp.commands = append(cp.commands, cmd)
	cp.refilter()
}

// ApplyTheme applies theme tokens to the CommandPalette and its inner input.
func (cp *CommandPalette) ApplyTheme(t latte.Theme) {
	cp.style = t.Dialog
	cp.titleStyle = t.DialogTitle
	cp.textStyle = t.Text
	cp.selectedStyle = t.ListSelected
	cp.input.ApplyTheme(t)
}

// --- oat.Focusable -----------------------------------------------------------

// IsFocused delegates to the inner input.
func (cp *CommandPalette) IsFocused() bool { return cp.input.IsFocused() }

// SetFocused propagates focus to both the palette's own FocusBehavior and the
// inner input, so that the input's border highlights correctly.
func (cp *CommandPalette) SetFocused(focused bool) {
	cp.FocusBehavior.SetFocused(focused)
	cp.input.SetFocused(focused)
}

// HandleKey processes navigation keys and delegates typing to the inner input.
func (cp *CommandPalette) HandleKey(ev *oat.KeyEvent) bool {
	switch ev.Key() {
	case tcell.KeyUp:
		if cp.selected > 0 {
			cp.selected--
		}
		return true
	case tcell.KeyDown:
		if cp.selected < len(cp.filtered)-1 {
			cp.selected++
		}
		return true
	case tcell.KeyEnter:
		if cp.selected >= 0 && cp.selected < len(cp.filtered) {
			if cp.filtered[cp.selected].Handler != nil {
				cp.filtered[cp.selected].Handler()
			}
		}
		return true
	case tcell.KeyEscape:
		// Do NOT consume — let the canvas handle dialog dismiss.
		return false
	default:
		// Delegate to input; refilter on every keystroke.
		consumed := cp.input.HandleKey(ev)
		if consumed {
			cp.refilter()
		}
		return consumed
	}
}

// KeyBindings advertises shortcuts for the StatusBar.
func (cp *CommandPalette) KeyBindings() []oat.KeyBinding {
	return []oat.KeyBinding{
		{Key: tcell.KeyUp, Label: "↑", Description: "Up"},
		{Key: tcell.KeyDown, Label: "↓", Description: "Down"},
		{Key: tcell.KeyEnter, Label: "Enter", Description: "Execute"},
	}
}

// --- oat.Layout --------------------------------------------------------------

// Children returns the inner input so that the focus walk and theme propagation
// descend into it.
func (cp *CommandPalette) Children() []oat.Component {
	return []oat.Component{cp.input}
}

// AddChild is a no-op; the palette manages its own children.
func (cp *CommandPalette) AddChild(_ oat.Component) {}

// --- oat.Component -----------------------------------------------------------

func (cp *CommandPalette) Measure(c oat.Constraint) oat.Size {
	maxW := c.MaxWidth
	if maxW < 0 {
		maxW = 60
	}
	paletteW := maxW * 2 / 3
	if paletteW > 60 {
		paletteW = 60
	}
	if paletteW > maxW {
		paletteW = maxW
	}
	if paletteW < 20 {
		paletteW = 20
	}

	listH := len(cp.filtered)
	if listH > cp.maxVisible {
		listH = cp.maxVisible
	}
	inputH := 3 // bordered single-line input
	total := inputH + listH + 2 // top + bottom border of the outer container
	return oat.Size{Width: paletteW, Height: total}
}

func (cp *CommandPalette) Render(buf *oat.Buffer, region oat.Region) {
	// Resolve palette dimensions.
	paletteW := region.Width * 2 / 3
	if paletteW > 60 {
		paletteW = 60
	}
	if paletteW > region.Width {
		paletteW = region.Width
	}
	if paletteW < 20 {
		paletteW = 20
	}

	listH := len(cp.filtered)
	if listH > cp.maxVisible {
		listH = cp.maxVisible
	}
	inputH := 3
	paletteH := inputH + listH + 2

	if paletteH > region.Height {
		paletteH = region.Height
	}

	// Place near the top-third of the screen.
	centerX := (region.Width - paletteW) / 2
	centerY := region.Height / 4

	// Clamp to stay within region.
	if centerX < 0 {
		centerX = 0
	}
	if centerY < 0 {
		centerY = 0
	}
	if centerX+paletteW > region.Width {
		paletteW = region.Width - centerX
	}
	if centerY+paletteH > region.Height {
		paletteH = region.Height - centerY
	}

	sub := buf.Sub(oat.Region{X: centerX, Y: centerY, Width: paletteW, Height: paletteH})
	sub.FillBG(cp.style)

	borderStyle := cp.style.Border
	if borderStyle == latte.BorderNone || borderStyle == latte.BorderExplicitNone {
		borderStyle = latte.BorderRounded
	}
	sub.DrawBorderTitle(borderStyle, "Command Palette", cp.titleStyle, cp.style, oat.AnchorCenter)

	// Inner area: inside the 1-cell border on all sides.
	innerW := paletteW - 2
	innerH := paletteH - 2
	if innerW < 0 {
		innerW = 0
	}
	if innerH < 0 {
		innerH = 0
	}
	inner := sub.Sub(oat.Region{X: 1, Y: 1, Width: innerW, Height: innerH})

	// Render the input in the top 3 rows of the inner area.
	if inputH <= innerH {
		cp.input.Render(inner, oat.Region{X: 0, Y: 0, Width: innerW, Height: inputH})
	} else if innerH > 0 {
		cp.input.Render(inner, oat.Region{X: 0, Y: 0, Width: innerW, Height: innerH})
	}

	// Render the filtered command list below the input.
	listStartY := inputH
	availListH := innerH - listStartY
	if availListH <= 0 {
		return
	}

	for i, cmd := range cp.filtered {
		if i >= cp.maxVisible {
			break
		}
		rowY := listStartY + i
		if rowY >= innerH {
			break
		}

		rowStyle := cp.textStyle
		if i == cp.selected {
			rowStyle = cp.selectedStyle
		}

		// Compose "  Name   Description" truncated to innerW.
		line := "  " + cmd.Name
		if cmd.Description != "" {
			line += "  " + cmd.Description
		}
		runes := []rune(line)
		if len(runes) > innerW {
			runes = runes[:innerW]
		}
		// Pad the rest of the row to fill background.
		inner.DrawText(0, rowY, string(runes), rowStyle)
		for x := len(runes); x < innerW; x++ {
			inner.SetCell(x, rowY, ' ', rowStyle)
		}
	}
}

// --- internal ----------------------------------------------------------------

// refilter re-computes the filtered list from the current input text.
// If the query is empty all commands are shown.
func (cp *CommandPalette) refilter() {
	query := strings.ToLower(cp.input.GetText())
	if query == "" {
		cp.filtered = make([]Command, len(cp.commands))
		copy(cp.filtered, cp.commands)
	} else {
		cp.filtered = cp.filtered[:0]
		for _, cmd := range cp.commands {
			if strings.Contains(strings.ToLower(cmd.Name), query) ||
				strings.Contains(strings.ToLower(cmd.Description), query) {
				cp.filtered = append(cp.filtered, cmd)
			}
		}
	}
	// Clamp selection.
	if cp.selected >= len(cp.filtered) {
		cp.selected = len(cp.filtered) - 1
	}
	if cp.selected < 0 {
		cp.selected = 0
	}
}
