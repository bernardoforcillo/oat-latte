package layout

import (
	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
)

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
	rows     int
	cols     int
	children []GridChild
	rowGap   int
	colGap   int
}

// NewGrid creates a Grid with the given number of rows and columns.
func NewGrid(rows, cols int) *Grid {
	return &Grid{rows: rows, cols: cols}
}

// WithStyle sets the style.
func (g *Grid) WithStyle(s latte.Style) *Grid { g.Style = s; return g }

// WithGap sets the gap between rows and columns.
func (g *Grid) WithGap(rowGap, colGap int) *Grid { g.rowGap = rowGap; g.colGap = colGap; return g }

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

	cellW := 0
	cellH := 0
	if g.cols > 0 {
		cellW = (inner.MaxWidth - g.colGap*(g.cols-1)) / g.cols
	}
	if g.rows > 0 {
		cellH = (inner.MaxHeight - g.rowGap*(g.rows-1)) / g.rows
	}

	totalW := cellW*g.cols + g.colGap*(g.cols-1) + g.Style.Padding.Left + g.Style.Padding.Right
	totalH := cellH*g.rows + g.rowGap*(g.rows-1) + g.Style.Padding.Top + g.Style.Padding.Bottom
	return oat.Size{Width: clamp(totalW, 0, c.MaxWidth), Height: clamp(totalH, 0, c.MaxHeight)}
}

// Render draws each child in its assigned cell.
func (g *Grid) Render(buf *oat.Buffer, region oat.Region) {
	inner := region.Inner(toOatInsets(g.Style.Padding))
	sub := buf.Sub(inner)

	if g.cols == 0 || g.rows == 0 {
		return
	}

	cellW := (inner.Width - g.colGap*(g.cols-1)) / g.cols
	cellH := (inner.Height - g.rowGap*(g.rows-1)) / g.rows

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

		x := gc.Col * (cellW + g.colGap)
		y := gc.Row * (cellH + g.rowGap)
		w := cellW*cs + g.colGap*(cs-1)
		h := cellH*rs + g.rowGap*(rs-1)

		childRegion := oat.Region{X: x, Y: y, Width: w, Height: h}
		gc.Component.Render(sub, childRegion)
	}
}
