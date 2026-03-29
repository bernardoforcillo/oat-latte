package oat

import (
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// Buffer is an abstraction over tcell.Screen that provides
// bounds-checked, clipped cell writes. It also handles the
// conversion from latte.Style to tcell.Style.
//
// Buffer tracks the most recently filled background colour (bg). Any component
// that draws with BG == ColorDefault inherits this colour instead of falling
// through to the terminal's own default (typically black). This gives the
// framework "virtual transparency": a child widget whose style has no explicit
// background will always appear on top of whatever background the parent
// painted, regardless of the terminal default.
//
// All rendering in oat-latte goes through a Buffer — no component
// ever writes directly to tcell.Screen.
//
// # Coordinate model
//
// Each Buffer has two independent concepts:
//   - origin (originX, originY): the screen-coordinate offset added to every
//     caller-relative (x, y) before writing. This is what shifts the coordinate
//     space for a child component.
//   - clip (clip Region): the screen-space rectangle outside of which all
//     writes are silently dropped. The clip is ALWAYS within the parent's clip,
//     regardless of the origin. This is what enforces viewport boundaries.
//
// Separating origin from clip allows ScrollView to shift the child's origin
// upward by -scrollOff rows (so row 0 in the child maps to screen row
// origin.Y - scrollOff, which is above the viewport) while still clipping all
// writes to the visible viewport rectangle. Without this separation, a negative
// Y offset in Sub would push the clip region above the viewport and allow
// content to bleed into neighbouring panels.
type Buffer struct {
	screen  tcell.Screen
	clip    Region      // screen-space write guard; all writes outside are dropped
	originX int         // screen X of coordinate (0,0) in this buffer
	originY int         // screen Y of coordinate (0,0) in this buffer
	bg      latte.Color // inherited background colour; ColorDefault until a fill is done
}

// newBuffer creates a Buffer wrapping the given tcell.Screen.
// The initial clip region covers the full screen.
func newBuffer(screen tcell.Screen) *Buffer {
	w, h := screen.Size()
	return &Buffer{
		screen:  screen,
		clip:    Region{X: 0, Y: 0, Width: w, Height: h},
		originX: 0,
		originY: 0,
	}
}

// Sub returns a new Buffer whose coordinate origin is at region's top-left
// corner (relative to this buffer's origin) and whose clip region is the
// intersection of region with this buffer's clip.
//
// Coordinates passed to the sub-buffer are relative to region's origin.
// The parent's current background colour is inherited so children that render
// with BG == ColorDefault appear on top of the parent's background.
//
// Negative region.X / region.Y values are fully supported: the coordinate
// origin is translated (so Y=0 in the child maps to above the parent's visible
// area) while the clip is still intersected with the parent, ensuring that
// writes outside the visible viewport are always discarded. This is the
// mechanism ScrollView uses to shift content up by -scrollOff rows while
// guaranteeing that nothing bleeds above the border into neighbouring panels.
func (b *Buffer) Sub(region Region) *Buffer {
	// New origin in absolute screen coordinates.
	newOriginX := b.originX + region.X
	newOriginY := b.originY + region.Y

	// The clip is the intersection of region (in absolute coords) with the
	// parent's existing clip.  We compute the region's absolute bounding box
	// first, then intersect.
	regAbsX := b.originX + region.X
	regAbsY := b.originY + region.Y
	regAbsRight := regAbsX + region.Width
	regAbsBottom := regAbsY + region.Height

	// Parent clip bounds.
	clipLeft := b.clip.X
	clipTop := b.clip.Y
	clipRight := b.clip.X + b.clip.Width
	clipBottom := b.clip.Y + b.clip.Height

	// Intersection.
	intLeft := regAbsX
	if clipLeft > intLeft {
		intLeft = clipLeft
	}
	intTop := regAbsY
	if clipTop > intTop {
		intTop = clipTop
	}
	intRight := regAbsRight
	if clipRight < intRight {
		intRight = clipRight
	}
	intBottom := regAbsBottom
	if clipBottom < intBottom {
		intBottom = clipBottom
	}

	w := intRight - intLeft
	h := intBottom - intTop
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}

	return &Buffer{
		screen:  b.screen,
		clip:    Region{X: intLeft, Y: intTop, Width: w, Height: h},
		originX: newOriginX,
		originY: newOriginY,
		bg:      b.bg,
	}
}

// resolveStyle substitutes the buffer's inherited background colour into style
// when style.BG is ColorDefault. This prevents children that carry no explicit
// BG from falling through to the terminal's default (typically black) and
// overwriting the background the parent already painted.
func (b *Buffer) resolveStyle(style latte.Style) latte.Style {
	if style.BG == latte.ColorDefault && b.bg != latte.ColorDefault {
		style.BG = b.bg
	}
	return style
}

// resolveBorderStyle returns a style where BorderBG is filled in from the
// buffer's inherited background when it is ColorDefault. Used so border runes
// are drawn on the same background as the rest of the panel, not the terminal
// default.
func (b *Buffer) resolveBorderStyle(style latte.Style) latte.Style {
	if style.BorderBG == latte.ColorDefault && b.bg != latte.ColorDefault {
		style.BorderBG = b.bg
	}
	return style
}

// SetCell writes a single rune at (x, y) relative to the buffer's origin.
// Out-of-bounds writes (outside the clip region) are silently dropped.
func (b *Buffer) SetCell(x, y int, ch rune, style latte.Style) {
	ax := b.originX + x
	ay := b.originY + y
	if ax < b.clip.X || ax >= b.clip.X+b.clip.Width {
		return
	}
	if ay < b.clip.Y || ay >= b.clip.Y+b.clip.Height {
		return
	}
	b.screen.SetContent(ax, ay, ch, nil, b.resolveStyle(style).ToTcell())
}

// SetCellTcell writes a cell using a raw tcell.Style (used internally for borders).
func (b *Buffer) SetCellTcell(x, y int, ch rune, style tcell.Style) {
	ax := b.originX + x
	ay := b.originY + y
	if ax < b.clip.X || ax >= b.clip.X+b.clip.Width {
		return
	}
	if ay < b.clip.Y || ay >= b.clip.Y+b.clip.Height {
		return
	}
	b.screen.SetContent(ax, ay, ch, nil, style)
}

// Fill fills the entire buffer region with the given rune and style.
// If style.BG is a concrete colour it is recorded as the buffer's current
// background so that subsequent children that draw with BG == ColorDefault
// inherit it rather than falling through to the terminal default.
func (b *Buffer) Fill(ch rune, style latte.Style) {
	if style.BG != latte.ColorDefault {
		b.bg = style.BG
	}
	ts := b.resolveStyle(style).ToTcell()
	for y := 0; y < b.clip.Height; y++ {
		for x := 0; x < b.clip.Width; x++ {
			b.screen.SetContent(b.clip.X+x, b.clip.Y+y, ch, nil, ts)
		}
	}
}

// FillBG fills the buffer region with spaces using the given background color.
func (b *Buffer) FillBG(style latte.Style) {
	b.Fill(' ', style)
}

// Width returns the width of this buffer's clip region.
func (b *Buffer) Width() int { return b.clip.Width }

// Height returns the height of this buffer's clip region.
func (b *Buffer) Height() int { return b.clip.Height }

// Region returns the current clipping region in absolute screen coordinates.
func (b *Buffer) Region() Region { return b.clip }

// DrawText writes a string starting at (x, y), clipped to the buffer bounds.
// Returns the x position after the last character written.
func (b *Buffer) DrawText(x, y int, text string, style latte.Style) int {
	ts := b.resolveStyle(style).ToTcell()
	cx := x
	for _, ch := range text {
		ax := b.originX + cx
		ay := b.originY + y
		if ax < b.clip.X || ax >= b.clip.X+b.clip.Width {
			cx++
			continue
		}
		if ay < b.clip.Y || ay >= b.clip.Y+b.clip.Height {
			cx++
			continue
		}
		b.screen.SetContent(ax, ay, ch, nil, ts)
		cx++
	}
	return cx
}

// DrawTextAligned writes text within a fixed-width cell [x, x+width),
// aligned according to latte.Alignment.
func (b *Buffer) DrawTextAligned(x, y, width int, text string, align latte.Alignment, style latte.Style) {
	runes := []rune(text)
	textLen := len(runes)
	if textLen > width {
		runes = runes[:width]
		textLen = width
	}

	startX := x
	switch align {
	case latte.AlignCenter:
		startX = x + (width-textLen)/2
	case latte.AlignEnd:
		startX = x + width - textLen
	}

	ts := b.resolveStyle(style).ToTcell()
	for i, ch := range runes {
		cx := startX + i
		if cx < x || cx >= x+width {
			continue
		}
		ax := b.originX + cx
		ay := b.originY + y
		if ax < b.clip.X || ax >= b.clip.X+b.clip.Width {
			continue
		}
		if ay < b.clip.Y || ay >= b.clip.Y+b.clip.Height {
			continue
		}
		b.screen.SetContent(ax, ay, ch, nil, ts)
	}
}

// DrawBorder draws a border around the full buffer region using the given style.
func (b *Buffer) DrawBorder(borderStyle latte.BorderStyle, style latte.Style) {
	b.DrawBorderTitle(borderStyle, "", latte.Style{}, style, AnchorLeft)
}

// DrawBorderTitle draws a border and optionally stamps " title " into the top rule.
// anchor (oat.Anchor, H-axis) controls the horizontal position of the title:
// AnchorLeft (after the opening corner), AnchorCenter, or AnchorRight (before
// the closing corner). titleStyle is used for the title text; if its FG is
// ColorDefault the border FG is used.
//
// Note: Anchor is the horizontal-axis type. The title always appears in the
// top border row; there is no vertical placement variant for this function.
// For vertical positioning see oat.VAnchor (used by Divider).
func (b *Buffer) DrawBorderTitle(borderStyle latte.BorderStyle, title string, titleStyle latte.Style, style latte.Style, anchor Anchor) {
	if borderStyle == latte.BorderNone || borderStyle == latte.BorderExplicitNone || b.clip.Width < 2 || b.clip.Height < 2 {
		return
	}
	runes := borderStyle.Runes()
	bs := b.resolveBorderStyle(style).BorderTcell()

	w := b.clip.Width
	h := b.clip.Height

	// Top and bottom rows
	for x := 1; x < w-1; x++ {
		b.SetCellTcell(x, 0, runes.Top, bs)
		b.SetCellTcell(x, h-1, runes.Bottom, bs)
	}
	// Left and right columns
	for y := 1; y < h-1; y++ {
		b.SetCellTcell(0, y, runes.Left, bs)
		b.SetCellTcell(w-1, y, runes.Right, bs)
	}
	// Corners
	b.SetCellTcell(0, 0, runes.TopLeft, bs)
	b.SetCellTcell(w-1, 0, runes.TopRight, bs)
	b.SetCellTcell(0, h-1, runes.BottomLeft, bs)
	b.SetCellTcell(w-1, h-1, runes.BottomRight, bs)

	// Stamp title into the top border line.
	if title == "" || w < 6 {
		return
	}

	// Build the padded label: " Title "
	label := " " + title + " "
	labelRunes := []rune(label)

	// Available interior width: leave 2 cells for corners + 1 guard on each side.
	maxLen := w - 4
	if len(labelRunes) > maxLen {
		labelRunes = labelRunes[:maxLen]
	}

	ts := b.resolveStyle(titleStyle).ToTcell()
	// If the caller didn't set a title FG, inherit the border FG.
	if titleStyle.FG == latte.ColorDefault && titleStyle.BG == latte.ColorDefault &&
		!titleStyle.Bold && !titleStyle.Italic {
		ts = bs
	}

	// Compute startX based on anchor.
	// Interior runs from x=1 to x=w-2 (inclusive). Guard of 1 on each side gives
	// usable range [2, w-2-len(label)].
	var startX int
	labelLen := len(labelRunes)
	switch anchor {
	case AnchorRight:
		startX = w - 2 - labelLen // just before the right corner
	case AnchorCenter:
		startX = 1 + (w-2-labelLen)/2
		if startX < 2 {
			startX = 2
		}
	default: // AnchorLeft
		startX = 2 // after ╭─
	}

	for i, r := range labelRunes {
		b.SetCellTcell(startX+i, 0, r, ts)
	}
}

// ShowCursor positions the terminal cursor at (x, y) within this buffer.
// Used by EditText to show the insertion point.
func (b *Buffer) ShowCursor(x, y int) {
	b.screen.ShowCursor(b.originX+x, b.originY+y)
}

// HideCursor hides the terminal cursor.
func (b *Buffer) HideCursor() {
	b.screen.HideCursor()
}
