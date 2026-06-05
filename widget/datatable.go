package widget

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strings"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// DataTable is a focusable, filterable, sortable, paginated data table with
// multi-row selection and CSV export.
//
// Default keybindings (when focused):
//   - ↑↓      Navigate rows
//   - ←→      Page through pages (when pageSize > 0)
//   - Home    Jump to first row
//   - End     Jump to last row
//   - Enter   Invoke onSelect callback
//   - Space   Toggle selection on current row
//   - Ctrl-A  Select all / deselect all
//   - s       Advance sort column
type DataTable struct {
	oat.BaseComponent
	oat.FocusBehavior
	oat.BaseHitRegion

	columns  []Column
	allRows  [][]string
	filtered [][]string // recomputed by recompute(); nil until first use

	sortCol int  // -1 = unsorted
	sortAsc bool

	globalFilter string

	pageSize int // 0 = no pagination
	page     int // 0-indexed

	cursor   int          // cursor within current page rows
	selected map[int]bool // selected indices into filtered (not page-relative)

	onSelect func(int, []string) // absolute filtered index, row data

	selectedStyle latte.Style
	cursorStyle   latte.Style
	headerStyle   latte.Style
	mutedStyle    latte.Style
	callerStyle   latte.Style
	showHeader    bool
}

// NewDataTable creates a DataTable with the given column definitions.
func NewDataTable(columns []Column) *DataTable {
	dt := &DataTable{
		columns:    columns,
		sortCol:    -1,
		sortAsc:    true,
		pageSize:   0,
		selected:   make(map[int]bool),
		showHeader: true,
	}
	dt.EnsureID()
	return dt
}

// WithID sets a user-defined identifier on this component.
func (dt *DataTable) WithID(id string) *DataTable { dt.ID = id; return dt }

// WithRows replaces the table's row data and recomputes filtered state.
func (dt *DataTable) WithRows(rows [][]string) *DataTable {
	dt.allRows = rows
	dt.recompute()
	return dt
}

// WithPageSize sets the number of rows per page (0 = show all).
func (dt *DataTable) WithPageSize(n int) *DataTable { dt.pageSize = n; return dt }

// WithShowHeader controls whether the header row and separator are rendered.
func (dt *DataTable) WithShowHeader(show bool) *DataTable { dt.showHeader = show; return dt }

// WithOnSelect registers a callback invoked when the user presses Enter.
// The callback receives the absolute filtered-row index and the row data.
func (dt *DataTable) WithOnSelect(fn func(int, []string)) *DataTable {
	dt.onSelect = fn
	return dt
}

// SetGlobalFilter sets the filter string, recomputes filtered rows, and resets the page.
func (dt *DataTable) SetGlobalFilter(s string) {
	dt.globalFilter = s
	dt.recompute()
	dt.page = 0
}

// SortBy sorts by the given column index.
// If the same column is already active, it cycles asc → desc → unsorted.
// If a different column is chosen, sorting starts ascending.
func (dt *DataTable) SortBy(col int) {
	if col == dt.sortCol {
		if dt.sortAsc {
			dt.sortAsc = false
		} else {
			dt.sortCol = -1
			dt.sortAsc = true
		}
	} else {
		dt.sortCol = col
		dt.sortAsc = true
	}
	dt.recompute()
}

// SelectedIndices returns a sorted list of selected filtered-row indices.
func (dt *DataTable) SelectedIndices() []int {
	indices := make([]int, 0, len(dt.selected))
	for idx := range dt.selected {
		indices = append(indices, idx)
	}
	sort.Ints(indices)
	return indices
}

// Export writes all filtered rows (with header if showHeader) to w as CSV.
func (dt *DataTable) Export(w io.Writer) error {
	if dt.filtered == nil {
		dt.recompute()
	}
	cw := csv.NewWriter(w)
	if dt.showHeader {
		headers := make([]string, len(dt.columns))
		for i, col := range dt.columns {
			headers[i] = col.Header
		}
		if err := cw.Write(headers); err != nil {
			return err
		}
	}
	for _, row := range dt.filtered {
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// recompute rebuilds the filtered (and sorted) slice from allRows.
// It also clamps cursor and page to the new length.
func (dt *DataTable) recompute() {
	// 1. Filter
	if dt.globalFilter == "" {
		dst := make([][]string, len(dt.allRows))
		copy(dst, dt.allRows)
		dt.filtered = dst
	} else {
		lower := strings.ToLower(dt.globalFilter)
		dst := dt.filtered[:0]
		if dst == nil {
			dst = make([][]string, 0, len(dt.allRows))
		} else {
			dst = dst[:0]
		}
		for _, row := range dt.allRows {
			for _, cell := range row {
				if strings.Contains(strings.ToLower(cell), lower) {
					dst = append(dst, row)
					break
				}
			}
		}
		dt.filtered = dst
	}

	// 2. Sort
	if dt.sortCol >= 0 && dt.sortCol < len(dt.columns) {
		col := dt.sortCol
		asc := dt.sortAsc
		sort.SliceStable(dt.filtered, func(i, j int) bool {
			a, b := "", ""
			if col < len(dt.filtered[i]) {
				a = dt.filtered[i][col]
			}
			if col < len(dt.filtered[j]) {
				b = dt.filtered[j][col]
			}
			if asc {
				return a < b
			}
			return a > b
		})
	}

	// 3. Clamp cursor and page
	total := len(dt.filtered)
	if total == 0 {
		dt.cursor = 0
		dt.page = 0
		return
	}
	tp := dt.totalPages()
	if dt.page >= tp {
		dt.page = tp - 1
	}
	if dt.page < 0 {
		dt.page = 0
	}
	rows, _ := dt.pageRows()
	maxCursor := len(rows) - 1
	if maxCursor < 0 {
		maxCursor = 0
	}
	dt.cursor = clamp(dt.cursor, 0, maxCursor)
}

// pageRows returns the rows visible on the current page and their absolute
// start index within filtered.
func (dt *DataTable) pageRows() (rows [][]string, startIdx int) {
	if dt.pageSize <= 0 {
		return dt.filtered, 0
	}
	startIdx = dt.page * dt.pageSize
	maxStart := len(dt.filtered) - 1
	if maxStart < 0 {
		maxStart = 0
	}
	startIdx = clamp(startIdx, 0, maxStart)
	end := startIdx + dt.pageSize
	if end > len(dt.filtered) {
		end = len(dt.filtered)
	}
	return dt.filtered[startIdx:end], startIdx
}

// totalPages returns the total number of pages.
func (dt *DataTable) totalPages() int {
	if dt.pageSize <= 0 || len(dt.filtered) == 0 {
		return 1
	}
	return (len(dt.filtered) + dt.pageSize - 1) / dt.pageSize
}

// colWidths computes the effective display width for each column given the
// total available width. Mirrors the same algorithm used by Table.
func (dt *DataTable) colWidths(totalW int) []int {
	widths := make([]int, len(dt.columns))
	fixedSum := 0
	autoCount := 0
	for i, col := range dt.columns {
		if col.Width > 0 {
			widths[i] = col.Width
			fixedSum += col.Width
		} else {
			autoCount++
		}
	}
	separators := len(dt.columns) - 1
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
		for i, col := range dt.columns {
			if col.Width == 0 {
				widths[i] = share
			}
		}
	}
	return widths
}

// ApplyTheme applies theme tokens to the DataTable.
func (dt *DataTable) ApplyTheme(t latte.Theme) {
	dt.Style = t.Text.Merge(dt.callerStyle)
	dt.FocusStyle = latte.Style{BorderFG: t.FocusBorder}
	dt.selectedStyle = t.ListSelected
	dt.cursorStyle = t.Accent
	dt.headerStyle = latte.Style{FG: t.Text.FG, BG: t.Text.BG, Bold: true}
	dt.mutedStyle = t.Muted
}

// Measure returns the desired size of the DataTable.
func (dt *DataTable) Measure(c oat.Constraint) oat.Size {
	headerH := 0
	if dt.showHeader {
		headerH = 2 // header row + separator line
	}
	bodyH := len(dt.allRows)
	totalH := headerH + bodyH
	if dt.pageSize > 0 {
		totalH++ // pagination footer
	}
	if c.MaxHeight >= 0 && totalH > c.MaxHeight {
		totalH = c.MaxHeight
	}

	totalW := 0
	for _, col := range dt.columns {
		totalW += col.Width
	}
	if len(dt.columns) > 1 {
		totalW += len(dt.columns) - 1 // separators
	}
	if c.MaxWidth >= 0 && totalW > c.MaxWidth {
		totalW = c.MaxWidth
	}
	if totalW == 0 && c.MaxWidth >= 0 {
		totalW = c.MaxWidth
	}
	return oat.Size{Width: totalW, Height: totalH}
}

// Render draws the DataTable into the buffer within the given region.
func (dt *DataTable) Render(buf *oat.Buffer, region oat.Region) {
	sub := buf.Sub(region)
	dt.SetHitRegion(sub.Region())
	style := dt.EffectiveStyle(dt.IsFocused())
	sub.FillBG(style)

	if len(dt.columns) == 0 {
		return
	}

	// Ensure filtered is initialized.
	if dt.filtered == nil {
		dt.recompute()
	}

	colW := dt.colWidths(region.Width)

	y := 0

	if dt.showHeader {
		// Draw header row.
		x := 0
		for i, col := range dt.columns {
			w := colW[i]
			header := col.Header

			// For the sort column, append indicator and reduce available text width.
			sortIndicator := ""
			if i == dt.sortCol {
				if dt.sortAsc {
					sortIndicator = " ▲"
				} else {
					sortIndicator = " ▼"
				}
			}

			availW := w
			if sortIndicator != "" {
				availW -= 2
				if availW < 0 {
					availW = 0
				}
			}

			headerRunes := []rune(header)
			if len(headerRunes) > availW {
				headerRunes = headerRunes[:availW]
			}
			text := string(headerRunes)
			// Pad to availW then append indicator.
			for len([]rune(text)) < availW {
				text += " "
			}
			text += sortIndicator
			// Ensure total length does not exceed column width.
			textRunes := []rune(text)
			if len(textRunes) > w {
				textRunes = textRunes[:w]
				text = string(textRunes)
			}
			// Pad to column width if shorter.
			for len([]rune(text)) < w {
				text += " "
			}

			hStyle := latte.Style{FG: style.FG, BG: style.BG, Bold: true}
			sub.DrawText(x, y, text, hStyle)
			x += w
			if i < len(dt.columns)-1 {
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
			if i < len(dt.columns)-1 {
				sub.SetCell(x, y, '┼', style)
				x++
			}
		}
		y++
	}

	rows, startIdx := dt.pageRows()

	pageFooterH := 0
	if dt.pageSize > 0 {
		pageFooterH = 1
	}

	visibleH := region.Height - y - pageFooterH
	if visibleH < 0 {
		visibleH = 0
	}

	limit := visibleH
	if limit > len(rows) {
		limit = len(rows)
	}

	for row := 0; row < limit; row++ {
		absIdx := startIdx + row

		// Determine row style.
		var rowStyle latte.Style
		isCursor := absIdx == startIdx+dt.cursor && dt.IsFocused()
		if isCursor {
			rowStyle = dt.cursorStyle
			if rowStyle == (latte.Style{}) {
				rowStyle = style
				rowStyle.Reverse = true
			}
		} else if dt.selected[absIdx] {
			rowStyle = dt.selectedStyle
			if rowStyle == (latte.Style{}) {
				rowStyle = style
				rowStyle.Reverse = true
			}
		} else {
			rowStyle = style
		}

		// Fill entire row background first.
		for fillX := 0; fillX < region.Width; fillX++ {
			sub.SetCell(fillX, y+row, ' ', rowStyle)
		}

		x := 0
		rowData := rows[row]
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
			if i < len(dt.columns)-1 {
				sub.SetCell(x, y+row, '│', style)
				x++
			}
		}
	}

	// Draw pagination footer.
	if dt.pageSize > 0 && region.Height > 0 {
		footerY := region.Height - 1
		footerText := fmt.Sprintf("Page %d/%d ← →  %d rows",
			dt.page+1, dt.totalPages(), len(dt.filtered))
		if len(dt.selected) > 0 {
			footerText += fmt.Sprintf("  (%d selected)", len(dt.selected))
		}
		mutedStyle := dt.mutedStyle
		if mutedStyle == (latte.Style{}) {
			mutedStyle = style
		}
		sub.DrawText(0, footerY, footerText, mutedStyle)
	}
}

// HandleKey processes keyboard input when the DataTable is focused.
func (dt *DataTable) HandleKey(ev *oat.KeyEvent) bool {
	if dt.filtered == nil {
		dt.recompute()
	}
	rows, startIdx := dt.pageRows()

	switch ev.Key() {
	case tcell.KeyUp:
		if dt.cursor > 0 {
			dt.cursor--
		}
		return true

	case tcell.KeyDown:
		if dt.cursor < len(rows)-1 {
			dt.cursor++
		}
		return true

	case tcell.KeyHome:
		dt.cursor = 0
		return true

	case tcell.KeyEnd:
		end := len(rows) - 1
		if end < 0 {
			end = 0
		}
		dt.cursor = end
		return true

	case tcell.KeyLeft:
		if dt.pageSize > 0 && dt.page > 0 {
			dt.page--
			dt.cursor = 0
		}
		return true

	case tcell.KeyRight:
		if dt.pageSize > 0 && dt.page < dt.totalPages()-1 {
			dt.page++
			dt.cursor = 0
		}
		return true

	case tcell.KeyEnter:
		if dt.onSelect != nil && len(rows) > 0 {
			dt.onSelect(startIdx+dt.cursor, rows[dt.cursor])
		}
		return true

	case tcell.KeyCtrlA:
		// If all filtered rows are selected, clear; else select all.
		if len(dt.selected) == len(dt.filtered) {
			dt.selected = make(map[int]bool)
		} else {
			for i := range dt.filtered {
				dt.selected[i] = true
			}
		}
		return true

	case tcell.KeyRune:
		switch ev.Rune() {
		case ' ':
			absIdx := startIdx + dt.cursor
			if dt.selected[absIdx] {
				delete(dt.selected, absIdx)
			} else {
				dt.selected[absIdx] = true
			}
			return true

		case 's':
			// Advance to next column; wrap from last to -1 (unsorted).
			if len(dt.columns) == 0 {
				return true
			}
			if dt.sortCol >= len(dt.columns)-1 {
				// Was at last column; go unsorted.
				dt.sortCol = -1
				dt.sortAsc = true
			} else {
				dt.sortCol++
				dt.sortAsc = true
			}
			dt.recompute()
			return true
		}
	}

	return false
}

// KeyBindings returns the keybindings advertised to the StatusBar.
func (dt *DataTable) KeyBindings() []oat.KeyBinding {
	return []oat.KeyBinding{
		{Key: tcell.KeyUp, Label: "↑↓", Description: "Navigate"},
		{Key: tcell.KeyLeft, Label: "←→", Description: "Page"},
		{Key: tcell.KeyEnter, Label: "Enter", Description: "Select"},
		{Key: tcell.KeyRune, Label: "Space", Description: "Toggle"},
	}
}

// Children implements oat.Layout. DataTable has no child components.
func (dt *DataTable) Children() []oat.Component { return nil }

// AddChild implements oat.Layout. No-op for DataTable.
func (dt *DataTable) AddChild(_ oat.Component) {}
