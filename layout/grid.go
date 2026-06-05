package layout

import (
	"strconv"
	"strings"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
)

// ---- track sizing types ---------------------------------------------------

// trackKind describes how a Grid track (column or row) is sized.
type trackKind int

const (
	trackFr    trackKind = iota // flexible fraction of remaining space
	trackFixed                  // fixed number of terminal cells
)

// trackSize describes the desired size of one column or row track.
type trackSize struct {
	kind  trackKind
	fr    float64 // used when kind == trackFr
	cells int     // used when kind == trackFixed
}

// ---- template parser ------------------------------------------------------

// parseTrackTemplate parses a space-separated size template into trackSizes.
//
// Token syntax:
//
//	"Nfr"  — flexible track with weight N (e.g. "1fr", "2fr", "0.5fr")
//	"N"    — fixed track of exactly N terminal cells (e.g. "20", "40")
//	"auto" — equivalent to "1fr" (equal share of remaining space)
//
// Examples:
//
//	"1fr 2fr 1fr" → three fr tracks, middle gets twice as much space
//	"20 1fr 20"   → fixed 20 cells, flexible middle, fixed 20 cells
func parseTrackTemplate(template string) []trackSize {
	tokens := strings.Fields(template)
	out := make([]trackSize, 0, len(tokens))
	for _, tok := range tokens {
		switch {
		case strings.HasSuffix(tok, "fr"):
			prefix := strings.TrimSuffix(tok, "fr")
			fr := 1.0
			if prefix != "" {
				if v, err := strconv.ParseFloat(prefix, 64); err == nil {
					fr = v
				}
			}
			out = append(out, trackSize{kind: trackFr, fr: fr})
		case tok == "auto":
			out = append(out, trackSize{kind: trackFr, fr: 1.0})
		default:
			if n, err := strconv.Atoi(tok); err == nil {
				out = append(out, trackSize{kind: trackFixed, cells: n})
			} else {
				out = append(out, trackSize{kind: trackFr, fr: 1.0})
			}
		}
	}
	return out
}

// ---- track resolver -------------------------------------------------------

// resolveTracks distributes `available` cells among `count` tracks according to
// the template. If len(template) < count, the last template entry is repeated.
// If len(template) > count, extra entries are ignored.
// Gap is the number of cells between tracks.
// Returns the computed cell count for each of the `count` tracks.
func resolveTracks(template []trackSize, count, available, gap int) []int {
	if count <= 0 {
		return nil
	}

	// Build a per-track slice, repeating the last entry as necessary.
	tracks := make([]trackSize, count)
	for i := 0; i < count; i++ {
		if i < len(template) {
			tracks[i] = template[i]
		} else {
			tracks[i] = template[len(template)-1]
		}
	}

	// Subtract gaps from available space.
	avail := available - gap*(count-1)
	if avail < 0 {
		avail = 0
	}

	// Compute fixed sum and total fr weight.
	fixedSum := 0
	totalFr := 0.0
	for _, t := range tracks {
		if t.kind == trackFixed {
			fixedSum += t.cells
		} else {
			totalFr += t.fr
		}
	}

	frSpace := avail - fixedSum
	if frSpace < 0 {
		frSpace = 0
	}

	sizes := make([]int, count)
	frSum := 0
	lastFrIdx := -1

	for i, t := range tracks {
		if t.kind == trackFixed {
			sizes[i] = t.cells
			if sizes[i] < 0 {
				sizes[i] = 0
			}
		} else {
			if totalFr > 0 {
				sizes[i] = int(t.fr / totalFr * float64(frSpace))
			} else {
				sizes[i] = 0
			}
			if sizes[i] < 0 {
				sizes[i] = 0
			}
			frSum += sizes[i]
			lastFrIdx = i
		}
	}

	// Distribute remainder caused by integer truncation to the last fr track.
	if lastFrIdx >= 0 {
		remainder := frSpace - frSum
		if remainder > 0 {
			sizes[lastFrIdx] += remainder
		}
	}

	return sizes
}

// ---- Grid -----------------------------------------------------------------

// GridChild wraps a component with its grid position and optional span.
type GridChild struct {
	Component oat.Component
	Row, Col  int
	RowSpan   int // default 1
	ColSpan   int // default 1
}

// Grid arranges children in a fixed rows×cols grid.
// Each cell has equal width and height (terminal grids rarely need unequal cells).
type Grid struct {
	oat.BaseComponent
	rows        int
	cols        int
	children    []GridChild
	rowGap      int
	colGap      int
	colTemplate []trackSize // nil = legacy equal-cell distribution
	rowTemplate []trackSize // nil = legacy equal-cell distribution
}

// NewGrid creates a Grid with the given number of rows and columns.
func NewGrid(rows, cols int) *Grid {
	return &Grid{rows: rows, cols: cols}
}

// WithStyle sets the style.
func (g *Grid) WithStyle(s latte.Style) *Grid { g.Style = s; return g }

// WithGap sets the gap between rows and columns.
func (g *Grid) WithGap(rowGap, colGap int) *Grid { g.rowGap = rowGap; g.colGap = colGap; return g }

// WithColumnTemplate sets column widths from a space-separated template string.
// Tokens: "Nfr" for flexible (e.g. "1fr"), "N" for fixed cells, "auto" for equal share.
// Example: grid.WithColumnTemplate("1fr 2fr 1fr")
// Overrides the equal-distribution default; number of template tokens need not equal cols.
func (g *Grid) WithColumnTemplate(tmpl string) *Grid {
	g.colTemplate = parseTrackTemplate(tmpl)
	return g
}

// WithRowTemplate sets row heights from a space-separated template string.
func (g *Grid) WithRowTemplate(tmpl string) *Grid {
	g.rowTemplate = parseTrackTemplate(tmpl)
	return g
}

// ApplyTheme is a no-op for Grid — it carries no semantic role in the theme.
func (g *Grid) ApplyTheme(_ latte.Theme) {}

// Add places a component at (row, col) with span (1,1).
func (g *Grid) Add(row, col int, c oat.Component) *Grid {
	g.children = append(g.children, GridChild{Component: c, Row: row, Col: col, RowSpan: 1, ColSpan: 1})
	return g
}

// AddSpan places a component at (row, col) with the given row and column span.
func (g *Grid) AddSpan(row, col, rowSpan, colSpan int, c oat.Component) *Grid {
	g.children = append(g.children, GridChild{Component: c, Row: row, Col: col, RowSpan: rowSpan, ColSpan: colSpan})
	return g
}

// AddChild satisfies oat.Layout (appends at next available cell).
func (g *Grid) AddChild(c oat.Component) {
	pos := len(g.children)
	row := pos / g.cols
	col := pos % g.cols
	g.Add(row, col, c)
}

// Children satisfies oat.Layout.
func (g *Grid) Children() []oat.Component {
	out := make([]oat.Component, len(g.children))
	for i, gc := range g.children {
		out[i] = gc.Component
	}
	return out
}

// Measure returns the size needed to fit all rows and columns.
func (g *Grid) Measure(c oat.Constraint) oat.Size {
	pad := toOatInsets(g.Style.Padding)
	inner := c.Shrink(pad)

	availW := inner.MaxWidth
	if availW < 0 {
		availW = 0
	}
	availH := inner.MaxHeight
	if availH < 0 {
		availH = 0
	}

	totalW := 0
	totalH := 0

	if g.colTemplate != nil && g.cols > 0 {
		colWidths := resolveTracks(g.colTemplate, g.cols, availW, g.colGap)
		for _, w := range colWidths {
			totalW += w
		}
		totalW += g.colGap * (g.cols - 1)
	} else {
		cellW := 0
		if g.cols > 0 {
			cellW = (inner.MaxWidth - g.colGap*(g.cols-1)) / g.cols
		}
		totalW = cellW*g.cols + g.colGap*(g.cols-1)
	}

	if g.rowTemplate != nil && g.rows > 0 {
		rowHeights := resolveTracks(g.rowTemplate, g.rows, availH, g.rowGap)
		for _, h := range rowHeights {
			totalH += h
		}
		totalH += g.rowGap * (g.rows - 1)
	} else {
		cellH := 0
		if g.rows > 0 {
			cellH = (inner.MaxHeight - g.rowGap*(g.rows-1)) / g.rows
		}
		totalH = cellH*g.rows + g.rowGap*(g.rows-1)
	}

	totalW += g.Style.Padding.Left + g.Style.Padding.Right
	totalH += g.Style.Padding.Top + g.Style.Padding.Bottom
	return oat.Size{Width: clamp(totalW, 0, c.MaxWidth), Height: clamp(totalH, 0, c.MaxHeight)}
}

// Render draws each child in its assigned cell.
func (g *Grid) Render(buf *oat.Buffer, region oat.Region) {
	inner := region.Inner(toOatInsets(g.Style.Padding))
	sub := buf.Sub(inner)

	if g.cols == 0 || g.rows == 0 {
		return
	}

	// Resolve per-track sizes.
	var colWidths, rowHeights []int

	if g.colTemplate != nil {
		colWidths = resolveTracks(g.colTemplate, g.cols, inner.Width, g.colGap)
	} else {
		cellW := (inner.Width - g.colGap*(g.cols-1)) / g.cols
		colWidths = make([]int, g.cols)
		for i := range colWidths {
			colWidths[i] = cellW
		}
	}

	if g.rowTemplate != nil {
		rowHeights = resolveTracks(g.rowTemplate, g.rows, inner.Height, g.rowGap)
	} else {
		cellH := (inner.Height - g.rowGap*(g.rows-1)) / g.rows
		rowHeights = make([]int, g.rows)
		for i := range rowHeights {
			rowHeights[i] = cellH
		}
	}

	// Compute prefix-sum offsets: colOffsets[i] = x start of column i.
	colOffsets := make([]int, g.cols+1)
	for i := 0; i < g.cols; i++ {
		colOffsets[i+1] = colOffsets[i] + colWidths[i] + g.colGap
	}

	rowOffsets := make([]int, g.rows+1)
	for i := 0; i < g.rows; i++ {
		rowOffsets[i+1] = rowOffsets[i] + rowHeights[i] + g.rowGap
	}

	for _, gc := range g.children {
		if gc.Row >= g.rows || gc.Col >= g.cols {
			continue
		}
		rs := gc.RowSpan
		if rs < 1 {
			rs = 1
		}
		cs := gc.ColSpan
		if cs < 1 {
			cs = 1
		}

		// Clamp spans to grid bounds.
		if gc.Col+cs > g.cols {
			cs = g.cols - gc.Col
		}
		if gc.Row+rs > g.rows {
			rs = g.rows - gc.Row
		}

		x := colOffsets[gc.Col]
		y := rowOffsets[gc.Row]
		// Width = sum of spanned column widths + gaps between them.
		w := colOffsets[gc.Col+cs] - colOffsets[gc.Col] - g.colGap
		h := rowOffsets[gc.Row+rs] - rowOffsets[gc.Row] - g.rowGap

		if w < 0 {
			w = 0
		}
		if h < 0 {
			h = 0
		}

		childRegion := oat.Region{X: x, Y: y, Width: w, Height: h}
		gc.Component.Render(sub, childRegion)
	}
}
