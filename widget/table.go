package widget

import (
	"strings"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// Column describes a single column in a Table.
type Column struct {
	Header string
	Width  int // 0 = auto (distribute remaining space equally among auto columns)
}

// Table is a focusable data table with headers, rows, and keyboard navigation.
//
// Default keybindings (when focused):
//   - ↑     Up — select the previous row
//   - ↓     Down — select the next row
//   - Home  Jump — select the first row
//   - End   Jump — select the last row
type Table struct {
	oat.BaseComponent
	oat.FocusBehavior
	oat.BaseHitRegion

	columns    []Column
	rows       [][]string
	selected   int  // selected row index (-1 = none)
	scrollOff  int  // index of first visible row
	sortCol    int  // -1 = unsorted
	sortAsc    bool
	showHeader bool

	selectedStyle latte.Style
	callerStyle   latte.Style
}

// NewTable creates a Table with the given column definitions.
func NewTable(columns []Column) *Table {
	t := &Table{
		columns:    columns,
		selected:   -1,
		sortCol:    -1,
		sortAsc:    true,
		showHeader: true,
	}
	t.EnsureID()
	return t
}

// WithID sets a user-defined identifier on this component.
func (t *Table) WithID(id string) *Table { t.ID = id; return t }

// WithRows replaces the table's row data.
func (t *Table) WithRows(rows [][]string) *Table {
	t.rows = rows
	t.selected = clamp(t.selected, -1, len(rows)-1)
	return t
}

// WithShowHeader controls whether the header row and separator are rendered.
// Defaults to true.
func (t *Table) WithShowHeader(show bool) *Table { t.showHeader = show; return t }

// AddRow appends a row to the table.
func (t *Table) AddRow(row []string) *Table {
	t.rows = append(t.rows, row)
	return t
}

// SetRows replaces the table's row data.
func (t *Table) SetRows(rows [][]string) {
	t.rows = rows
	if t.selected >= len(rows) {
		t.selected = len(rows) - 1
	}
}

// SelectedIndex returns the index of the currently selected row, or -1.
func (t *Table) SelectedIndex() int { return t.selected }

// SelectedRow returns the currently selected row's cells, or nil if nothing
// is selected.
func (t *Table) SelectedRow() []string {
	if t.selected < 0 || t.selected >= len(t.rows) {
		return nil
	}
	return t.rows[t.selected]
}

// GetValue implements oat.ValueGetter. Returns SelectedRow().
func (t *Table) GetValue() interface{} { return t.SelectedRow() }

// ApplyTheme applies theme tokens to the Table.
func (t *Table) ApplyTheme(theme latte.Theme) {
	t.Style = theme.Text.Merge(t.callerStyle)
	t.FocusStyle = latte.Style{BorderFG: theme.FocusBorder}
	t.selectedStyle = theme.ListSelected
}

// colWidths computes the effective display width for each column given the
// total available width.
func (t *Table) colWidths(totalW int) []int {
	widths := make([]int, len(t.columns))
	fixedSum := 0
	autoCount := 0
	for i, col := range t.columns {
		if col.Width > 0 {
			widths[i] = col.Width
			fixedSum += col.Width
		} else {
			autoCount++
		}
	}
	// Account for '│' separators between columns.
	separators := len(t.columns) - 1
	if separators < 0 {
		separators = 0
	}
	remaining := totalW - fixedSum - separators
	if remaining < 0 {
		remaining = 0
	}
	if autoCount > 0 {
		share := remaining / autoCount
		if share < 1 {
			share = 1
		}
		for i, col := range t.columns {
			if col.Width == 0 {
				widths[i] = share
			}
		}
	}
	return widths
}

func (t *Table) Measure(c oat.Constraint) oat.Size {
	headerH := 0
	if t.showHeader {
		headerH = 2 // header row + separator line
	}
	bodyH := len(t.rows)
	totalH := headerH + bodyH
	if c.MaxHeight >= 0 && totalH > c.MaxHeight {
		totalH = c.MaxHeight
	}

	totalW := 0
	for _, col := range t.columns {
		totalW += col.Width
	}
	if len(t.columns) > 1 {
		totalW += len(t.columns) - 1 // separators
	}
	if c.MaxWidth >= 0 && totalW > c.MaxWidth {
		totalW = c.MaxWidth
	}
	if totalW == 0 && c.MaxWidth >= 0 {
		totalW = c.MaxWidth
	}
	return oat.Size{Width: totalW, Height: totalH}
}

func (t *Table) Render(buf *oat.Buffer, region oat.Region) {
	style := t.EffectiveStyle(t.IsFocused())
	sub := buf.Sub(region)
	t.SetHitRegion(sub.Region())
	sub.FillBG(style)

	if len(t.columns) == 0 {
		return
	}

	colW := t.colWidths(region.Width)

	y := 0

	if t.showHeader {
		// Draw header row.
		x := 0
		for i, col := range t.columns {
			header := col.Header
			headerRunes := []rune(header)
			w := colW[i]
			// Clip header to column width.
			if len(headerRunes) > w {
				headerRunes = headerRunes[:w]
			}
			text := string(headerRunes)
			// Pad to column width.
			for len([]rune(text)) < w {
				text += " "
			}
			headerStyle := latte.Style{FG: style.FG, BG: style.BG, Bold: true}
			sub.DrawText(x, y, text, headerStyle)
			x += w
			if i < len(t.columns)-1 {
				sub.SetCell(x, y, '│', style)
				x++
			}
		}
		y++

		// Draw separator line.
		x = 0
		for i, w := range colW {
			line := strings.Repeat("─", w)
			sub.DrawText(x, y, line, style)
			x += w
			if i < len(t.columns)-1 {
				sub.SetCell(x, y, '┼', style)
				x++
			}
		}
		y++
	}

	// Visible body rows.
	headerH := y
	visibleH := region.Height - headerH
	if visibleH < 0 {
		visibleH = 0
	}

	// Adjust scroll to keep selected visible.
	if t.selected >= 0 {
		if t.selected < t.scrollOff {
			t.scrollOff = t.selected
		}
		if t.selected >= t.scrollOff+visibleH {
			t.scrollOff = t.selected - visibleH + 1
		}
	}
	if t.scrollOff < 0 {
		t.scrollOff = 0
	}

	for row := 0; row < visibleH; row++ {
		rowIdx := t.scrollOff + row
		if rowIdx >= len(t.rows) {
			break
		}
		rowData := t.rows[rowIdx]

		rowStyle := style
		if rowIdx == t.selected {
			rowStyle = t.selectedStyle
			if t.selectedStyle == (latte.Style{}) {
				rowStyle = style
				rowStyle.Reverse = true
			}
		}

		x := 0
		for i, w := range colW {
			cellText := ""
			if i < len(rowData) {
				cellText = rowData[i]
			}
			cellRunes := []rune(cellText)
			if len(cellRunes) > w {
				cellRunes = cellRunes[:w]
			}
			text := string(cellRunes)
			for len([]rune(text)) < w {
				text += " "
			}
			sub.DrawText(x, y+row, text, rowStyle)
			x += w
			if i < len(t.columns)-1 {
				sub.SetCell(x, y+row, '│', style)
				x++
			}
		}
	}
}

func (t *Table) HandleKey(ev *oat.KeyEvent) bool {
	switch ev.Key() {
	case tcell.KeyUp:
		if t.selected > 0 {
			t.selected--
		} else if t.selected == -1 && len(t.rows) > 0 {
			t.selected = 0
		}
		return true
	case tcell.KeyDown:
		if t.selected < len(t.rows)-1 {
			t.selected++
		} else if t.selected == -1 && len(t.rows) > 0 {
			t.selected = 0
		}
		return true
	case tcell.KeyHome:
		if len(t.rows) > 0 {
			t.selected = 0
			t.scrollOff = 0
		}
		return true
	case tcell.KeyEnd:
		if len(t.rows) > 0 {
			t.selected = len(t.rows) - 1
		}
		return true
	}
	return false
}

// Children implements oat.Layout. Table has no child components.
func (t *Table) Children() []oat.Component { return nil }

// AddChild implements oat.Layout. No-op for Table.
func (t *Table) AddChild(_ oat.Component) {}

func (t *Table) KeyBindings() []oat.KeyBinding {
	return []oat.KeyBinding{
		{Key: tcell.KeyUp, Label: "↑", Description: "Navigate"},
		{Key: tcell.KeyDown, Label: "↓", Description: "Navigate"},
		{Key: tcell.KeyHome, Label: "Home", Description: "Jump"},
		{Key: tcell.KeyEnd, Label: "End", Description: "Jump"},
	}
}
