// Package layout provides container components for the oat-latte TUI framework.
// Layouts position and size their children — they never render content themselves.
package layout

import (
	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
)

// childSlot pairs a Component with its computed flex weight for the box layouts.
type childSlot struct {
	component oat.Component
	flex      int // 0 = fixed/auto; >0 = flex weight for remaining space
}

// ---- VBox -----------------------------------------------------------------

// VBox stacks children vertically, top to bottom.
// Children with flex > 0 share the remaining vertical space proportionally.
type VBox struct {
	oat.BaseComponent
	slots  []childSlot
	gap    int        // rows of empty space between children
	hAlign oat.HAlign // default horizontal alignment for children (HAlignFill = fill, unchanged)
}

// NewVBox creates a VBox with optional children.
// Children that implement FlexSpacer (e.g. VFill, FlexChild) are automatically
// promoted to flex slots, matching the behaviour of AddChild.
func NewVBox(children ...oat.Component) *VBox {
	v := &VBox{}
	for _, c := range children {
		v.AddChild(c)
	}
	return v
}

// WithStyle sets the style for this VBox.
func (v *VBox) WithStyle(s latte.Style) *VBox { v.Style = s; return v }

// WithGap sets the number of empty rows between children.
func (v *VBox) WithGap(n int) *VBox { v.gap = n; return v }

// AsScrollView wraps this VBox in a ScrollView, making its content vertically
// scrollable. This is a convenience shorthand for layout.NewScrollView(vbox).
//
// Example:
//
//	panel := layout.NewBorder(
//	    layout.NewVBox(items...).AsScrollView().WithScrollBar(true),
//	).WithTitle("Results")
func (v *VBox) AsScrollView() *ScrollView { return NewScrollView(v) }

// WithHAlign sets the default horizontal alignment applied to every child in
// this VBox that does not carry its own alignment preference.
// Variadic so WithHAlign() with no argument resets to HAlignFill (full-width, unchanged).
func (v *VBox) WithHAlign(a ...oat.HAlign) *VBox {
	if len(a) > 0 {
		v.hAlign = a[0]
	} else {
		v.hAlign = oat.HAlignFill
	}
	return v
}

// ApplyTheme is a no-op for VBox — it carries no semantic role in the theme.
// The canvas tree-walk will recurse into children automatically.
func (v *VBox) ApplyTheme(_ latte.Theme) {}

// AddChild appends a child with flex weight 0 (fixed/auto size).
// If the child implements FlexSpacer (e.g. VFill), it is automatically
// promoted to a flex slot using the spacer's declared weight.
func (v *VBox) AddChild(c oat.Component) {
	if fs, ok := c.(FlexSpacer); ok {
		v.slots = append(v.slots, childSlot{c, fs.FlexWeight()})
		return
	}
	v.slots = append(v.slots, childSlot{c, 0})
}

// AddFlexChild appends a child that participates in flex space distribution.
func (v *VBox) AddFlexChild(c oat.Component, flex int) {
	if flex < 1 {
		flex = 1
	}
	v.slots = append(v.slots, childSlot{c, flex})
}

// Children satisfies oat.Layout.
func (v *VBox) Children() []oat.Component {
	out := make([]oat.Component, len(v.slots))
	for i, s := range v.slots {
		out[i] = s.component
	}
	return out
}

// Measure computes the VBox's desired size.
// Fixed/auto children contribute their measured height; flex children
// contribute zero (their height is only known at Render time).
func (v *VBox) Measure(c oat.Constraint) oat.Size {
	padded := c.Shrink(toOatInsets(v.Style.Padding))
	totalH := 0
	maxW := 0
	gaps := (len(v.slots) - 1) * v.gap
	if gaps < 0 {
		gaps = 0
	}
	totalH += gaps

	for _, slot := range v.slots {
		if slot.flex > 0 {
			// Flex children claim remaining space at render time; skip their height here.
			s := slot.component.Measure(oat.Constraint{MaxWidth: padded.MaxWidth, MaxHeight: -1})
			if s.Width > maxW {
				maxW = s.Width
			}
			continue
		}
		s := slot.component.Measure(padded)
		if s.Width > maxW {
			maxW = s.Width
		}
		totalH += s.Height
	}

	pad := v.Style.Padding
	return oat.Size{
		Width:  clamp(maxW+pad.Left+pad.Right, 0, c.MaxWidth),
		Height: clamp(totalH+pad.Top+pad.Bottom, 0, c.MaxHeight),
	}
}

// Render draws all children top-to-bottom within region.
func (v *VBox) Render(buf *oat.Buffer, region oat.Region) {
	inner := region.Inner(toOatInsets(v.Style.Padding))
	sub := buf.Sub(inner)

	// First pass: measure all fixed children to find remaining space for flex.
	totalFixed := 0
	totalFlex := 0
	fixedHeights := make([]int, len(v.slots))
	for i, slot := range v.slots {
		if slot.flex == 0 {
			s := slot.component.Measure(inner.ToConstraint())
			fixedHeights[i] = s.Height
			totalFixed += s.Height
		} else {
			totalFlex += slot.flex
		}
	}

	gaps := (len(v.slots) - 1) * v.gap
	if gaps < 0 {
		gaps = 0
	}
	remaining := inner.Height - totalFixed - gaps
	if remaining < 0 {
		remaining = 0
	}

	// Second pass: assign heights and render.
	y := 0
	for i, slot := range v.slots {
		h := fixedHeights[i]
		if slot.flex > 0 && totalFlex > 0 {
			h = (remaining * slot.flex) / totalFlex
			// Respect a VFill's optional max-size cap.
			if vf, ok := slot.component.(*VFill); ok && vf.maxSize > 0 && h > vf.maxSize {
				h = vf.maxSize
			}
		}

		// Resolve horizontal (cross-axis) alignment for this child.
		effectiveHAlign := resolveHAlign(slot.component, v.hAlign)
		if effectiveHAlign == oat.HAlignFill {
			childRegion := oat.Region{X: 0, Y: y, Width: inner.Width, Height: h}
			slot.component.Render(sub, childRegion)
		} else {
			desired := slot.component.Measure(oat.Constraint{MaxWidth: inner.Width, MaxHeight: h})
			w := desired.Width
			if w > inner.Width {
				w = inner.Width
			}
			x := 0
			switch effectiveHAlign {
			case oat.HAlignLeft:
				x = 0
			case oat.HAlignCenter:
				x = (inner.Width - w) / 2
			case oat.HAlignRight:
				x = inner.Width - w
			}
			childRegion := oat.Region{X: x, Y: y, Width: w, Height: h}
			slot.component.Render(sub, childRegion)
		}

		y += h
		if i < len(v.slots)-1 {
			y += v.gap
		}
	}
}

// ---- HBox -----------------------------------------------------------------

// HBox places children side by side horizontally, left to right.
// Children with flex > 0 share the remaining horizontal space proportionally.
type HBox struct {
	oat.BaseComponent
	slots  []childSlot
	gap    int        // columns of empty space between children
	vAlign oat.VAlign // default vertical alignment for children (VAlignFill = fill, unchanged)
}

// NewHBox creates an HBox with optional children.
// Children that implement FlexSpacer (e.g. HFill, FlexChild) are automatically
// promoted to flex slots, matching the behaviour of AddChild.
func NewHBox(children ...oat.Component) *HBox {
	h := &HBox{}
	for _, c := range children {
		h.AddChild(c)
	}
	return h
}

// WithStyle sets the style for this HBox.
func (h *HBox) WithStyle(s latte.Style) *HBox { h.Style = s; return h }

// WithGap sets the number of empty columns between children.
func (h *HBox) WithGap(n int) *HBox { h.gap = n; return h }

// AsScrollView wraps this HBox in a ScrollView, making its content vertically
// scrollable. This is a convenience shorthand for layout.NewScrollView(hbox).
//
// Note: HBox lays out children horizontally. AsScrollView is useful when the
// HBox is taller than the viewport (e.g. an HBox containing tall column
// panels), not for horizontal scrolling within a single row.
func (h *HBox) AsScrollView() *ScrollView { return NewScrollView(h) }

// WithVAlign sets the default vertical alignment applied to every child in
// this HBox that does not carry its own alignment preference.
// Variadic so WithVAlign() with no argument resets to VAlignFill (full-height, unchanged).
func (h *HBox) WithVAlign(a ...oat.VAlign) *HBox {
	if len(a) > 0 {
		h.vAlign = a[0]
	} else {
		h.vAlign = oat.VAlignFill
	}
	return h
}

// ApplyTheme is a no-op for HBox — it carries no semantic role in the theme.
// The canvas tree-walk will recurse into children automatically.
func (h *HBox) ApplyTheme(_ latte.Theme) {}

// AddChild appends a child with flex weight 0.
// If the child implements FlexSpacer (e.g. HFill), it is automatically
// promoted to a flex slot using the spacer's declared weight.
func (h *HBox) AddChild(c oat.Component) {
	if fs, ok := c.(FlexSpacer); ok {
		h.slots = append(h.slots, childSlot{c, fs.FlexWeight()})
		return
	}
	h.slots = append(h.slots, childSlot{c, 0})
}

// AddFlexChild appends a child that participates in flex space distribution.
func (h *HBox) AddFlexChild(c oat.Component, flex int) {
	if flex < 1 {
		flex = 1
	}
	h.slots = append(h.slots, childSlot{c, flex})
}

// Children satisfies oat.Layout.
func (h *HBox) Children() []oat.Component {
	out := make([]oat.Component, len(h.slots))
	for i, s := range h.slots {
		out[i] = s.component
	}
	return out
}

// Measure computes the HBox's desired size.
// Fixed/auto children contribute their measured width; flex children contribute zero.
func (h *HBox) Measure(c oat.Constraint) oat.Size {
	padded := c.Shrink(toOatInsets(h.Style.Padding))
	totalW := 0
	maxH := 0
	gaps := (len(h.slots) - 1) * h.gap
	if gaps < 0 {
		gaps = 0
	}
	totalW += gaps

	for _, slot := range h.slots {
		if slot.flex > 0 {
			s := slot.component.Measure(oat.Constraint{MaxWidth: -1, MaxHeight: padded.MaxHeight})
			if s.Height > maxH {
				maxH = s.Height
			}
			continue
		}
		s := slot.component.Measure(padded)
		if s.Height > maxH {
			maxH = s.Height
		}
		totalW += s.Width
	}

	pad := h.Style.Padding
	return oat.Size{
		Width:  clamp(totalW+pad.Left+pad.Right, 0, c.MaxWidth),
		Height: clamp(maxH+pad.Top+pad.Bottom, 0, c.MaxHeight),
	}
}

// Render draws all children left-to-right within region.
func (h *HBox) Render(buf *oat.Buffer, region oat.Region) {
	inner := region.Inner(toOatInsets(h.Style.Padding))
	sub := buf.Sub(inner)

	// First pass: measure fixed children.
	totalFixed := 0
	totalFlex := 0
	fixedWidths := make([]int, len(h.slots))
	for i, slot := range h.slots {
		if slot.flex == 0 {
			s := slot.component.Measure(inner.ToConstraint())
			fixedWidths[i] = s.Width
			totalFixed += s.Width
		} else {
			totalFlex += slot.flex
		}
	}

	gaps := (len(h.slots) - 1) * h.gap
	if gaps < 0 {
		gaps = 0
	}
	remaining := inner.Width - totalFixed - gaps
	if remaining < 0 {
		remaining = 0
	}

	// Second pass: assign widths and render.
	x := 0
	for i, slot := range h.slots {
		w := fixedWidths[i]
		if slot.flex > 0 && totalFlex > 0 {
			w = (remaining * slot.flex) / totalFlex
			// Respect an HFill's optional max-size cap.
			if hf, ok := slot.component.(*HFill); ok && hf.maxSize > 0 && w > hf.maxSize {
				w = hf.maxSize
			}
		}

		// Resolve vertical (cross-axis) alignment for this child.
		effectiveVAlign := resolveVAlign(slot.component, h.vAlign)
		if effectiveVAlign == oat.VAlignFill {
			childRegion := oat.Region{X: x, Y: 0, Width: w, Height: inner.Height}
			slot.component.Render(sub, childRegion)
		} else {
			desired := slot.component.Measure(oat.Constraint{MaxWidth: w, MaxHeight: inner.Height})
			ch := desired.Height
			if ch > inner.Height {
				ch = inner.Height
			}
			cy := 0
			switch effectiveVAlign {
			case oat.VAlignTop:
				cy = 0
			case oat.VAlignMiddle:
				cy = (inner.Height - ch) / 2
			case oat.VAlignBottom:
				cy = inner.Height - ch
			}
			childRegion := oat.Region{X: x, Y: cy, Width: w, Height: ch}
			slot.component.Render(sub, childRegion)
		}

		x += w
		if i < len(h.slots)-1 {
			x += h.gap
		}
	}
}

// ---- helpers --------------------------------------------------------------

func clamp(v, min, max int) int {
	if max < 0 {
		return v // unconstrained
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// toOatInsets converts a latte.Insets to an oat.Insets.
// The two types are structurally identical; this avoids a circular import.
func toOatInsets(i latte.Insets) oat.Insets {
	return oat.Insets{Top: i.Top, Right: i.Right, Bottom: i.Bottom, Left: i.Left}
}

// containsFocus reports whether c or any of its descendants is focused.
// Used by Border so it can highlight its border when any child is active.
func containsFocus(c oat.Component) bool {
	if f, ok := c.(oat.Focusable); ok && f.IsFocused() {
		return true
	}
	if l, ok := c.(oat.Layout); ok {
		for _, child := range l.Children() {
			if containsFocus(child) {
				return true
			}
		}
	}
	return false
}

// ---- alignment resolution helpers ----------------------------------------

// resolveHAlign returns the effective HAlign for a child component.
// Resolution order:
//  1. AlignChild wrapper — highest priority
//  2. AlignProvider on the child itself (e.g. BaseComponent.HAlign)
//  3. The parent box's boxDefault
//  4. HAlignFill (unchanged behaviour)
//
// If c is a FlexChild, resolution is performed on the inner child instead —
// FlexChild is a transparent flex-weight carrier and carries no alignment
// preference of its own.
func resolveHAlign(c oat.Component, boxDefault oat.HAlign) oat.HAlign {
	// Unwrap FlexChild — it is a transparent flex-weight carrier.
	if fc, ok := c.(*FlexChild); ok && fc.child != nil {
		c = fc.child
	}
	if ac, ok := c.(*AlignChild); ok {
		return ac.GetHAlign()
	}
	if ap, ok := c.(oat.AlignProvider); ok {
		if a := ap.GetHAlign(); a != oat.HAlignFill {
			return a
		}
	}
	return boxDefault
}

// resolveVAlign returns the effective VAlign for a child component.
// Resolution order:
//  1. AlignChild wrapper — highest priority
//  2. AlignProvider on the child itself (e.g. BaseComponent.VAlign)
//  3. The parent box's boxDefault
//  4. VAlignFill (unchanged behaviour)
//
// If c is a FlexChild, resolution is performed on the inner child instead —
// FlexChild is a transparent flex-weight carrier and carries no alignment
// preference of its own.
func resolveVAlign(c oat.Component, boxDefault oat.VAlign) oat.VAlign {
	// Unwrap FlexChild — it is a transparent flex-weight carrier.
	if fc, ok := c.(*FlexChild); ok && fc.child != nil {
		c = fc.child
	}
	if ac, ok := c.(*AlignChild); ok {
		return ac.GetVAlign()
	}
	if ap, ok := c.(oat.AlignProvider); ok {
		if a := ap.GetVAlign(); a != oat.VAlignFill {
			return a
		}
	}
	return boxDefault
}
