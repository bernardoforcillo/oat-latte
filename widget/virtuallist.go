package widget

// VirtualList is a focusable, vertically-scrollable list that renders only
// the visible window of items. Because the itemBuilder is called lazily for
// visible rows only, lists with millions of entries are fully practical.
//
// Fixed item height: every item occupies exactly 1 row. This covers the vast
// majority of list-display use cases. For variable-height items, compose
// multiple widgets instead.
//
// Default keybindings (when focused):
//   - ↑ / ↓    Navigate — move cursor one row
//   - PgUp     Page up — move cursor 10 rows up
//   - PgDn     Page down — move cursor 10 rows down
//   - Home     Top — jump to first item
//   - End      Bottom — jump to last item
//   - Enter    Select — invoke the onSelect callback

import (
	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// VirtualList renders only the rows currently visible in its region.
// itemBuilder is invoked lazily — only for indices within the visible window.
type VirtualList struct {
	oat.BaseComponent
	oat.FocusBehavior
	oat.BaseHitRegion

	itemCount   int
	itemBuilder func(index int) string
	// itemStyler, when non-nil, overrides per-row styling.
	// The selected bool is true only for the cursor row while the widget is focused.
	itemStyler func(index int, selected bool) latte.Style

	cursor    int
	scrollOff int
	onSelect  func(index int, text string)

	selectedStyle latte.Style
	normalStyle   latte.Style
	callerStyle   latte.Style
}

// NewVirtualList creates a VirtualList with itemCount items.
// itemBuilder is called lazily only for visible rows during Render,
// making it practical for millions of items.
func NewVirtualList(itemCount int, itemBuilder func(index int) string) *VirtualList {
	vl := &VirtualList{
		itemCount:   itemCount,
		itemBuilder: itemBuilder,
	}
	vl.EnsureID()
	return vl
}

// WithID sets a user-defined identifier on this component.
func (vl *VirtualList) WithID(id string) *VirtualList { vl.ID = id; return vl }

// WithItemStyler registers a per-row style function.
// It is called for every visible row during Render; selected is true only for
// the cursor row while the list is focused. Return a zero latte.Style to fall
// back to the theme defaults.
func (vl *VirtualList) WithItemStyler(fn func(index int, selected bool) latte.Style) *VirtualList {
	vl.itemStyler = fn
	return vl
}

// WithOnSelect registers a callback invoked when the user presses Enter.
// The callback receives the cursor index and the text returned by itemBuilder.
func (vl *VirtualList) WithOnSelect(fn func(index int, text string)) *VirtualList {
	vl.onSelect = fn
	return vl
}

// UpdateCount updates the item count, e.g. after appending to the data source.
// The cursor is clamped to [0, n-1] if necessary.
func (vl *VirtualList) UpdateCount(n int) {
	vl.itemCount = n
	if vl.itemCount <= 0 {
		vl.cursor = 0
		vl.scrollOff = 0
		return
	}
	if vl.cursor >= vl.itemCount {
		vl.cursor = vl.itemCount - 1
	}
}

// CursorIndex returns the current cursor position.
func (vl *VirtualList) CursorIndex() int { return vl.cursor }

// ScrollTo scrolls the viewport so that index is visible without moving the cursor.
func (vl *VirtualList) ScrollTo(index int) {
	if index < 0 {
		index = 0
	}
	if index >= vl.itemCount {
		index = max(0, vl.itemCount-1)
	}
	vl.scrollOff = index
}

// ApplyTheme applies theme tokens to the VirtualList.
func (vl *VirtualList) ApplyTheme(t latte.Theme) {
	vl.Style = t.Text.Merge(vl.callerStyle)
	vl.FocusStyle = latte.Style{BorderFG: t.FocusBorder}
	vl.normalStyle = t.Text
	vl.selectedStyle = t.ListSelected
}

// Measure returns the desired size given the available constraint.
// The width is capped to MaxWidth (or 20 if unconstrained) and the height is
// capped to itemCount (or MaxHeight, whichever is smaller).
func (vl *VirtualList) Measure(c oat.Constraint) oat.Size {
	h := vl.itemCount
	if c.MaxHeight >= 0 && h > c.MaxHeight {
		h = c.MaxHeight
	}
	w := c.MaxWidth
	if w < 0 {
		w = 20
	}
	return oat.Size{Width: w, Height: h}
}

// Render draws the visible window of items into buf within region.
// Only the rows whose index falls within [scrollOff, scrollOff+region.Height)
// are drawn — itemBuilder is never called for off-screen items.
func (vl *VirtualList) Render(buf *oat.Buffer, region oat.Region) {
	style := vl.EffectiveStyle(vl.IsFocused())
	sub := buf.Sub(region)
	vl.SetHitRegion(sub.Region())
	sub.FillBG(style)

	if vl.itemCount == 0 || region.Height <= 0 || region.Width <= 0 {
		return
	}

	// Clamp scroll offset to keep cursor visible.
	if vl.cursor < vl.scrollOff {
		vl.scrollOff = vl.cursor
	}
	if vl.cursor >= vl.scrollOff+region.Height {
		vl.scrollOff = vl.cursor - region.Height + 1
	}
	maxScroll := max(0, vl.itemCount-region.Height)
	vl.scrollOff = clamp(vl.scrollOff, 0, maxScroll)

	for row := 0; row < region.Height; row++ {
		idx := vl.scrollOff + row
		if idx >= vl.itemCount {
			break
		}

		isCursor := idx == vl.cursor && vl.IsFocused()

		// Determine row style.
		var rowStyle latte.Style
		if vl.itemStyler != nil {
			rowStyle = vl.itemStyler(idx, isCursor)
		}
		if rowStyle == (latte.Style{}) {
			if isCursor {
				rowStyle = vl.selectedStyle
			} else {
				rowStyle = vl.normalStyle
			}
		}
		// Final fallback: if still zero (e.g. no theme applied), use the
		// effective component style so something is always visible.
		if rowStyle == (latte.Style{}) {
			rowStyle = style
		}

		// Fill the entire row with the row's background first.
		for x := 0; x < region.Width; x++ {
			sub.SetCell(x, row, ' ', rowStyle)
		}

		// Fetch and truncate text to region width.
		text := vl.itemBuilder(idx)
		runes := []rune(text)
		if len(runes) > region.Width {
			runes = runes[:region.Width]
			text = string(runes)
		}

		sub.DrawText(0, row, text, rowStyle)
	}
}

// HandleKey processes keyboard navigation events.
func (vl *VirtualList) HandleKey(ev *oat.KeyEvent) bool {
	switch ev.Key() {
	case tcell.KeyUp:
		if vl.cursor > 0 {
			vl.cursor--
		}
		return true
	case tcell.KeyDown:
		if vl.cursor < vl.itemCount-1 {
			vl.cursor++
		}
		return true
	case tcell.KeyPgUp:
		vl.cursor = clamp(vl.cursor-10, 0, max(0, vl.itemCount-1))
		return true
	case tcell.KeyPgDn:
		vl.cursor = clamp(vl.cursor+10, 0, max(0, vl.itemCount-1))
		return true
	case tcell.KeyHome:
		vl.cursor = 0
		vl.scrollOff = 0
		return true
	case tcell.KeyEnd:
		vl.cursor = max(0, vl.itemCount-1)
		return true
	case tcell.KeyEnter:
		if vl.onSelect != nil && vl.itemCount > 0 {
			vl.onSelect(vl.cursor, vl.itemBuilder(vl.cursor))
		}
		return true
	}
	return false
}

// KeyBindings advertises the keyboard shortcuts for the StatusBar footer.
func (vl *VirtualList) KeyBindings() []oat.KeyBinding {
	return []oat.KeyBinding{
		{Key: tcell.KeyUp, Label: "↑↓", Description: "Navigate"},
		{Key: tcell.KeyPgUp, Label: "PgUp", Description: "Page"},
		{Key: tcell.KeyHome, Label: "Home", Description: "Top"},
		{Key: tcell.KeyEnter, Label: "Enter", Description: "Select"},
	}
}

// Children satisfies the oat.Layout interface; VirtualList has no children.
func (vl *VirtualList) Children() []oat.Component { return nil }

// AddChild satisfies the oat.Layout interface; VirtualList does not accept children.
func (vl *VirtualList) AddChild(_ oat.Component) {}
