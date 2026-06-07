package layout

import (
	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
)

// WrapBox lays out children horizontally and wraps them to the next line when
// they exceed the available width — like CSS flexbox with flex-wrap: wrap.
//
// Usage:
//
//	wrap := layout.NewWrapBox().WithGap(1).WithLineGap(0)
//	wrap.AddChild(widget.NewBadge("Go"))
//	wrap.AddChild(widget.NewBadge("TUI"))
//	wrap.AddChild(widget.NewButton("…", nil))
type WrapBox struct {
	oat.BaseComponent
	children []oat.Component
	gap      int // horizontal gap between items on the same line
	lineGap  int // vertical gap between lines
}

// NewWrapBox creates an empty WrapBox.
func NewWrapBox() *WrapBox {
	wb := &WrapBox{}
	wb.EnsureID()
	return wb
}

// WithID sets a user-defined identifier.
func (wb *WrapBox) WithID(id string) *WrapBox { wb.ID = id; return wb }

// WithGap sets the horizontal cell gap between items on the same line.
func (wb *WrapBox) WithGap(n int) *WrapBox { wb.gap = n; return wb }

// WithLineGap sets the vertical cell gap between lines.
func (wb *WrapBox) WithLineGap(n int) *WrapBox { wb.lineGap = n; return wb }

// ApplyTheme propagates the theme to all children.
func (wb *WrapBox) ApplyTheme(t latte.Theme) {
	for _, c := range wb.children {
		if tr, ok := c.(oat.ThemeReceiver); ok {
			tr.ApplyTheme(t)
		}
	}
}

// Measure returns the preferred size by simulating the wrap layout.
func (wb *WrapBox) Measure(c oat.Constraint) oat.Size {
	if len(wb.children) == 0 {
		return oat.Size{}
	}
	maxW := c.MaxWidth
	if maxW <= 0 {
		maxW = 9999
	}
	lines := wb.packLines(maxW)
	totalH := 0
	maxLineW := 0
	for i, line := range lines {
		lw, lh := wb.lineSize(line)
		if lw > maxLineW {
			maxLineW = lw
		}
		totalH += lh
		if i < len(lines)-1 {
			totalH += wb.lineGap
		}
	}
	return c.Clamp(oat.Size{Width: maxLineW, Height: totalH})
}

// Render draws all children at their computed wrap positions.
func (wb *WrapBox) Render(buf *oat.Buffer, region oat.Region) {
	if len(wb.children) == 0 {
		return
	}
	sub := buf.Sub(region)
	maxW := region.Width
	lines := wb.packLines(maxW)
	y := 0
	for _, line := range lines {
		_, lh := wb.lineSize(line)
		x := 0
		for i, item := range line {
			sz := item.sz
			childRegion := oat.Region{X: x, Y: y, Width: sz.Width, Height: lh}
			item.c.Render(sub, childRegion)
			x += sz.Width
			if i < len(line)-1 {
				x += wb.gap
			}
		}
		y += lh + wb.lineGap
	}
}

// Children satisfies oat.Layout for theme propagation and focus collection.
func (wb *WrapBox) Children() []oat.Component { return wb.children }

// AddChild appends a child to the WrapBox.
func (wb *WrapBox) AddChild(c oat.Component) { wb.children = append(wb.children, c) }

// ── internal ──────────────────────────────────────────────────────────────────

type wrapItem struct {
	c   oat.Component
	sz  oat.Size
}

// packLines groups children into lines that fit within maxW.
func (wb *WrapBox) packLines(maxW int) [][]wrapItem {
	var lines [][]wrapItem
	var cur []wrapItem
	curW := 0

	for _, child := range wb.children {
		sz := child.Measure(oat.Constraint{MaxWidth: maxW, MaxHeight: -1})
		gap := 0
		if len(cur) > 0 {
			gap = wb.gap
		}
		if len(cur) > 0 && curW+gap+sz.Width > maxW {
			// Start a new line.
			lines = append(lines, cur)
			cur = nil
			curW = 0
			gap = 0
		}
		cur = append(cur, wrapItem{c: child, sz: sz})
		curW += gap + sz.Width
	}
	if len(cur) > 0 {
		lines = append(lines, cur)
	}
	return lines
}

// lineSize returns the total width (including gaps) and maximum height of a line.
func (wb *WrapBox) lineSize(items []wrapItem) (w, h int) {
	for i, item := range items {
		if i > 0 {
			w += wb.gap
		}
		w += item.sz.Width
		if item.sz.Height > h {
			h = item.sz.Height
		}
	}
	return w, h
}
