package layout

import (
	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// ScrollView is a layout container that clips its single child to a viewport
// and lets the user scroll vertically to reveal content that exceeds the
// available height.
//
// # Flex children inside ScrollView
//
// VFill and FlexChild behave differently depending on whether the content
// overflows the viewport:
//   - Content fits (no scrolling): the full viewport height is passed to the
//     child, so flex children expand normally.
//   - Content overflows (scrolling active): the child is measured unconstrained
//     and flex children receive zero remaining space (they collapse to zero
//     height). Avoid VFill / FlexChild inside a ScrollView that is expected to
//     scroll; use fixed-height children instead.
//
// # Nesting with Border
//
// Prefer Border(ScrollView(VBox(…))) over ScrollView(Border(VBox(…))).
// In the former the border chrome is fixed and only the VBox content scrolls.
// In the latter the entire Border (including the title row) scrolls — the top
// border disappears as the user scrolls down.
//
// # Focus model
//
// ScrollView is always present in the Tab-cycle so that it can be reached
// before the first render (when content/viewport heights are not yet known).
// HandleKey returns false for scroll keys when content fits the viewport so
// arrow keys fall through to inter-widget focus cycling as usual.
// When content overflows, HandleKey consumes ↑/↓, PgUp/PgDn, and Home/End.
//
// # Scroll bar
//
// An optional single-column scroll bar can be shown with WithScrollBar.
// By default it appears on the right edge; pass oat.AnchorLeft to move it to
// the left edge. Bar colours come from the active theme (Muted → track,
// Accent → thumb) and can be overridden per-component with WithTrackColor and
// WithThumbColor.
type ScrollView struct {
	oat.BaseComponent
	oat.FocusBehavior

	child     oat.Component
	scrollOff int // current vertical offset in rows
	contentH  int // cached: full unconstrained child height from last Measure
	viewportH int // cached: visible height from last Measure / Render

	showScrollBar   bool
	scrollBarAnchor oat.Anchor // AnchorRight (default) or AnchorLeft

	// callerTrackColor / callerThumbColor hold the colours explicitly set by
	// the caller via WithTrackColor / WithThumbColor.  A zero value
	// (latte.ColorDefault) means "not set — inherit from theme".  ApplyTheme
	// checks these before falling back to the theme token so that explicit
	// overrides survive SetTheme calls unchanged.
	callerTrackColor latte.Color
	callerThumbColor latte.Color

	// Effective scroll bar styles resolved by ApplyTheme (or by the builder
	// when no theme has been applied yet).
	trackStyle latte.Style
	thumbStyle latte.Style
}

// NewScrollView creates a ScrollView wrapping child.
// Use the builder methods to configure the scroll bar.
func NewScrollView(child oat.Component) *ScrollView {
	sv := &ScrollView{child: child}
	sv.EnsureID()
	return sv
}

// WithScrollBar enables or disables the scroll bar indicator.
// The optional anchor argument controls which edge the bar appears on:
//   - oat.AnchorRight (default) — right edge
//   - oat.AnchorLeft             — left edge
//
// Passing no anchor uses AnchorRight.
func (sv *ScrollView) WithScrollBar(show bool, anchor ...oat.Anchor) *ScrollView {
	sv.showScrollBar = show
	sv.scrollBarAnchor = oat.AnchorRight
	if len(anchor) > 0 {
		sv.scrollBarAnchor = anchor[0]
	}
	return sv
}

// WithTrackColor overrides the colour used for the scroll bar track (the
// gutter cells that are not covered by the thumb).  Pass any latte.Color —
// latte.RGB, latte.Hex, or a named palette constant.
//
// The override survives SetTheme calls: once set it is never replaced by
// ApplyTheme.  Pass latte.ColorDefault to revert to theme-driven behaviour.
func (sv *ScrollView) WithTrackColor(c latte.Color) *ScrollView {
	sv.callerTrackColor = c
	sv.trackStyle = latte.Style{FG: c}
	return sv
}

// WithThumbColor overrides the colour used for the scroll bar thumb (the
// indicator that moves as the user scrolls).  Pass any latte.Color —
// latte.RGB, latte.Hex, or a named palette constant.
//
// The override survives SetTheme calls: once set it is never replaced by
// ApplyTheme.  Pass latte.ColorDefault to revert to theme-driven behaviour.
func (sv *ScrollView) WithThumbColor(c latte.Color) *ScrollView {
	sv.callerThumbColor = c
	sv.thumbStyle = latte.Style{FG: c}
	return sv
}

// WithID sets a user-defined identifier on this component.
func (sv *ScrollView) WithID(id string) *ScrollView { sv.ID = id; return sv }

// --- oat.Layout ------------------------------------------------------------

// AddChild sets the single scrolled child (replaces any existing child).
func (sv *ScrollView) AddChild(c oat.Component) { sv.child = c }

// Children satisfies oat.Layout so the framework's DFS walkers (theme
// propagation, focus collection, ID lookup) recurse into the child tree.
func (sv *ScrollView) Children() []oat.Component {
	if sv.child == nil {
		return nil
	}
	return []oat.Component{sv.child}
}

// --- oat.ThemeReceiver -----------------------------------------------------

// ApplyTheme propagates the active theme to the child and resolves the scroll
// bar colours.  The resolution order for each colour is:
//  1. Caller override (WithTrackColor / WithThumbColor) — survives SetTheme.
//  2. Theme token: Muted.FG for the track, Accent.FG for the thumb.
func (sv *ScrollView) ApplyTheme(t latte.Theme) {
	trackColor := t.Muted.FG
	if sv.callerTrackColor != latte.ColorDefault {
		trackColor = sv.callerTrackColor
	}
	thumbColor := t.Accent.FG
	if sv.callerThumbColor != latte.ColorDefault {
		thumbColor = sv.callerThumbColor
	}
	sv.trackStyle = latte.Style{FG: trackColor}
	sv.thumbStyle = latte.Style{FG: thumbColor}
	if tr, ok := sv.child.(oat.ThemeReceiver); ok {
		tr.ApplyTheme(t)
	}
}

// --- oat.Focusable ---------------------------------------------------------

// HandleKey handles the scroll keys when ScrollView has focus and content
// overflows the viewport. When content fits (contentH <= viewportH), all keys
// return false so arrows fall through to inter-widget focus cycling.
// ↑/↓ scroll by one row, PgUp/PgDn by a full viewport, Home/End jump to the
// extremes. Returns true (consumed) only when scrolling is active and the key
// is a recognised scroll key.
func (sv *ScrollView) HandleKey(ev *oat.KeyEvent) bool {
	if sv.contentH <= sv.viewportH {
		return false
	}
	switch ev.Key() {
	case tcell.KeyUp:
		sv.ScrollTo(sv.scrollOff - 1)
		return true
	case tcell.KeyDown:
		sv.ScrollTo(sv.scrollOff + 1)
		return true
	case tcell.KeyPgUp:
		sv.ScrollTo(sv.scrollOff - sv.viewportH)
		return true
	case tcell.KeyPgDn:
		sv.ScrollTo(sv.scrollOff + sv.viewportH)
		return true
	case tcell.KeyHome:
		sv.ScrollTo(0)
		return true
	case tcell.KeyEnd:
		sv.ScrollTo(sv.contentH - sv.viewportH)
		return true
	}
	return false
}

// KeyBindings advertises scroll shortcuts to the StatusBar footer.
func (sv *ScrollView) KeyBindings() []oat.KeyBinding {
	return []oat.KeyBinding{
		{Key: tcell.KeyUp, Label: "↑", Description: "Scroll up"},
		{Key: tcell.KeyDown, Label: "↓", Description: "Scroll down"},
		{Key: tcell.KeyPgUp, Label: "PgUp", Description: "Page up"},
		{Key: tcell.KeyPgDn, Label: "PgDn", Description: "Page down"},
	}
}

// --- oat.Scrollable --------------------------------------------------------

// ScrollOffset returns the current scroll offset in rows.
func (sv *ScrollView) ScrollOffset() int { return sv.scrollOff }

// ContentHeight returns the full unconstrained height of the child content.
func (sv *ScrollView) ContentHeight() int { return sv.contentH }

// ScrollTo sets the scroll offset, clamped to the valid range
// [0, max(0, contentH - viewportH)].
func (sv *ScrollView) ScrollTo(off int) {
	max := sv.contentH - sv.viewportH
	if max < 0 {
		max = 0
	}
	if off < 0 {
		off = 0
	}
	if off > max {
		off = max
	}
	sv.scrollOff = off
}

// --- oat.Component ---------------------------------------------------------

// Measure returns the desired size of the ScrollView.
//
// Two-pass strategy:
//  1. Measure the child with MaxHeight: -1 (unconstrained) to discover the
//     full content height. Flex children (VFill, FlexChild) contribute zero
//     to this measurement — their height is determined at render time.
//  2. If the content fits within the available viewport (or the viewport is
//     itself unconstrained) the full viewport height is reported back and
//     flex children will expand normally at render time.
//  3. If the content overflows the viewport, contentH is set to the raw
//     unconstrained height and the viewport height is returned as the
//     ScrollView's own desired height.
func (sv *ScrollView) Measure(c oat.Constraint) oat.Size {
	if sv.child == nil {
		return oat.Size{}
	}

	// Reserve one column for the scroll bar when enabled.
	childMaxW := c.MaxWidth
	if sv.showScrollBar && childMaxW > 1 {
		childMaxW--
	}

	// Step 1: measure child without a height constraint.
	raw := sv.child.Measure(oat.Constraint{MaxWidth: childMaxW, MaxHeight: -1})

	// Step 2: decide whether scrolling is needed.
	sv.viewportH = c.MaxHeight // -1 when the ScrollView itself is unconstrained
	if c.MaxHeight < 0 || raw.Height <= c.MaxHeight {
		// Content fits (or parent is unconstrained): report the raw height so
		// flex children inside the child can expand at render time.
		sv.contentH = raw.Height
		return oat.Size{Width: raw.Width, Height: raw.Height}
	}

	// Step 3: content overflows — clip to viewport.
	sv.contentH = raw.Height
	return oat.Size{Width: raw.Width, Height: c.MaxHeight}
}

// Render draws the scrolled child into the viewport.
//
// When content fits the child receives the full region so flex children expand
// normally. When content overflows the child is rendered into a region that
// starts at Y = -scrollOff; tcell silently discards cell writes that fall
// outside the physical viewport, providing the scroll clipping for free.
func (sv *ScrollView) Render(buf *oat.Buffer, region oat.Region) {
	if sv.child == nil {
		return
	}

	// Update cached viewport height from the actually allocated region (it may
	// differ from what Measure saw if the parent over-allocated).
	sv.viewportH = region.Height

	// Clamp scroll offset in case viewport or content changed since last frame.
	sv.ScrollTo(sv.scrollOff)

	// Determine effective content width (reserve one column for scroll bar).
	contentW := region.Width
	if sv.showScrollBar && sv.contentH > sv.viewportH {
		contentW--
		if contentW < 0 {
			contentW = 0
		}
	}

	// Create a sub-buffer clamped to the viewport.
	viewportBuf := buf.Sub(region)

	if sv.contentH <= sv.viewportH {
		// No scrolling needed: pass the full region so flex children expand.
		sv.child.Render(viewportBuf, oat.Region{
			X: 0, Y: 0,
			Width: contentW, Height: region.Height,
		})
	} else {
		// Scrolling path: shift the child's coordinate origin upward by
		// scrollOff rows. Writes above the viewport (absY < clip.Y) and below
		// (absY >= clip.Y + clip.Height) are silently dropped by tcell.
		sv.child.Render(viewportBuf, oat.Region{
			X: 0, Y: -sv.scrollOff,
			Width: contentW, Height: sv.contentH,
		})
	}

	// Draw the scroll bar if enabled and content overflows.
	if sv.showScrollBar && sv.contentH > sv.viewportH {
		sv.renderScrollBar(viewportBuf, region)
	}
}

// renderScrollBar draws a single-column scroll bar in the gutter column.
//
// Layout (right anchor, region.Width-1 column; left anchor, column 0):
//
//	│  ← track rune above thumb
//	█  ← thumb rune(s) proportional to viewportH / contentH
//	│  ← track rune below thumb
func (sv *ScrollView) renderScrollBar(buf *oat.Buffer, region oat.Region) {
	barCol := region.Width - 1
	if sv.scrollBarAnchor == oat.AnchorLeft {
		barCol = 0
	}

	barH := region.Height
	if barH <= 0 {
		return
	}

	// Thumb height: proportional to viewport / content, minimum 1 row.
	thumbH := (sv.viewportH * barH) / sv.contentH
	if thumbH < 1 {
		thumbH = 1
	}

	// Thumb position: top row of the thumb within the bar.
	maxScroll := sv.contentH - sv.viewportH
	thumbTop := 0
	if maxScroll > 0 {
		thumbTop = (sv.scrollOff * (barH - thumbH)) / maxScroll
	}
	if thumbTop+thumbH > barH {
		thumbTop = barH - thumbH
	}

	trackStyle := sv.trackStyle
	thumbStyle := sv.thumbStyle

	for y := 0; y < barH; y++ {
		if y >= thumbTop && y < thumbTop+thumbH {
			buf.SetCell(barCol, y, '█', thumbStyle)
		} else {
			buf.SetCell(barCol, y, '│', trackStyle)
		}
	}
}
