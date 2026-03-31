package layout

import (
	oat "github.com/antoniocali/oat-latte"
)

// ---- FlexSpacer -----------------------------------------------------------

// FlexSpacer is an optional interface for components that should be treated as
// flex children when added to a VBox or HBox via AddChild.
// VFill and HFill implement this interface so callers can use the simpler
// AddChild API without calling AddFlexChild manually.
type FlexSpacer interface {
	oat.Component
	// FlexWeight returns the flex weight for this spacer (always ≥ 1).
	FlexWeight() int
}

// ---- VFill ----------------------------------------------------------------

// VFill is a vertical spacer that expands to fill remaining space in a VBox.
// When added to a VBox via AddChild or AddFlexChild, it consumes all remaining
// vertical space and pushes its siblings toward the edges.
//
// Inside a scrollable container VFill degrades gracefully: use WithMaxSize to
// cap the height to a safe bound, or it will simply take zero height when
// remaining space is negative.
//
//	vbox.AddChild(topWidget)
//	vbox.AddChild(layout.NewVFill())   // expands to fill gap
//	vbox.AddChild(bottomWidget)
type VFill struct {
	weight  int
	maxSize int // 0 = uncapped
}

// NewVFill creates a VFill spacer with flex weight 1.
func NewVFill() *VFill { return &VFill{weight: 1} }

// WithWeight sets the flex weight (default 1). Higher weights claim
// proportionally more space when multiple flex children compete.
func (f *VFill) WithWeight(w int) *VFill {
	if w < 1 {
		w = 1
	}
	f.weight = w
	return f
}

// WithMaxSize caps the maximum height this spacer will consume.
// This is recommended when VFill is used inside a Scrollable container,
// where uncapped spacers can produce unexpectedly large content heights.
func (f *VFill) WithMaxSize(n int) *VFill { f.maxSize = n; return f }

// FlexWeight satisfies FlexSpacer.
func (f *VFill) FlexWeight() int { return f.weight }

// Measure returns zero — VFill claims space only at Render time via the flex
// pass in VBox.
func (f *VFill) Measure(_ oat.Constraint) oat.Size { return oat.Size{} }

// Render does nothing; VFill is a pure spacer.
func (f *VFill) Render(_ *oat.Buffer, _ oat.Region) {}

// ---- HFill ----------------------------------------------------------------

// HFill is a horizontal spacer that expands to fill remaining space in an HBox.
// When added to an HBox via AddChild or AddFlexChild, it consumes all remaining
// horizontal space and pushes its siblings toward the edges.
//
//	hbox.AddChild(leftWidget)
//	hbox.AddChild(layout.NewHFill())   // expands to fill gap
//	hbox.AddChild(rightWidget)
type HFill struct {
	weight  int
	maxSize int // 0 = uncapped
}

// NewHFill creates an HFill spacer with flex weight 1.
func NewHFill() *HFill { return &HFill{weight: 1} }

// WithWeight sets the flex weight (default 1).
func (f *HFill) WithWeight(w int) *HFill {
	if w < 1 {
		w = 1
	}
	f.weight = w
	return f
}

// WithMaxSize caps the maximum width this spacer will consume.
func (f *HFill) WithMaxSize(n int) *HFill { f.maxSize = n; return f }

// FlexWeight satisfies FlexSpacer.
func (f *HFill) FlexWeight() int { return f.weight }

// Measure returns zero — HFill claims space only at Render time via the flex
// pass in HBox.
func (f *HFill) Measure(_ oat.Constraint) oat.Size { return oat.Size{} }

// Render does nothing; HFill is a pure spacer.
func (f *HFill) Render(_ *oat.Buffer, _ oat.Region) {}

// ---- FlexChild ------------------------------------------------------------

// FlexChild wraps any Component so it participates in flex space distribution
// when added to a VBox or HBox via AddChild.  It is the component-bearing
// counterpart to VFill/HFill: instead of leaving the allocated space empty it
// renders its inner component into that space.
//
// The axis is determined by the container — FlexChild works equally well in
// both VBox (vertical flex) and HBox (horizontal flex).
//
//	// c2 fills remaining vertical space; c1 and c3 keep their natural heights.
//	vbox := layout.NewVBox(
//	    c1,
//	    layout.NewFlexChild(c2),
//	    c3,
//	)
//
//	// Optional second argument sets the flex weight (default 1).
//	layout.NewFlexChild(c2, 2)
type FlexChild struct {
	oat.BaseComponent
	child  oat.Component
	weight int
}

// NewFlexChild wraps child as a flex-weight component.
// The optional weight argument (default 1) controls how much of the remaining
// space this child claims relative to other flex children in the same box.
func NewFlexChild(child oat.Component, weight ...int) *FlexChild {
	w := 1
	if len(weight) > 0 && weight[0] > 1 {
		w = weight[0]
	}
	return &FlexChild{child: child, weight: w}
}

// FlexWeight satisfies FlexSpacer, causing AddChild to promote this wrapper
// to a flex slot automatically.
func (f *FlexChild) FlexWeight() int { return f.weight }

// AddChild sets the inner component (satisfies oat.Layout).
func (f *FlexChild) AddChild(c oat.Component) { f.child = c }

// Children satisfies oat.Layout so theme propagation and focus collection
// recurse into the inner component.
func (f *FlexChild) Children() []oat.Component {
	if f.child == nil {
		return nil
	}
	return []oat.Component{f.child}
}

// Measure delegates to the inner component.
// Called by VBox/HBox during their own Measure pass with MaxHeight/MaxWidth
// set to -1 (unconstrained on the flex axis); the width/height contribution
// is used only to size the container on the cross axis.
func (f *FlexChild) Measure(c oat.Constraint) oat.Size {
	if f.child == nil {
		return oat.Size{}
	}
	return f.child.Measure(c)
}

// Render delegates to the inner component using the full allocated region.
func (f *FlexChild) Render(buf *oat.Buffer, region oat.Region) {
	if f.child == nil {
		return
	}
	f.child.Render(buf, region)
}

// ---- AlignChild -----------------------------------------------------------

// AlignChild wraps a single component with explicit cross-axis alignment
// overrides. It is the per-child counterpart to VBox.WithHAlign / HBox.WithVAlign:
//
//   - In a VBox, AlignChild.hAlign overrides the box's default HAlign.
//   - In an HBox, AlignChild.vAlign overrides the box's default VAlign.
//
// AlignChild is NOT a FlexSpacer — it does not claim flex space on its own.
// Use AddFlexChild to make the wrapper participate in flex distribution:
//
//	vbox.AddFlexChild(layout.NewAlignChild(btn, oat.HAlignRight, oat.VAlignFill), 1)
//
// AlignChild implements oat.Layout so theme propagation and focus collection
// recurse into the wrapped component automatically.
type AlignChild struct {
	oat.BaseComponent
	child  oat.Component
	hAlign oat.HAlign
	vAlign oat.VAlign
}

// NewAlignChild wraps child with the given horizontal and vertical alignment.
// The alignment values control how the child is positioned within the
// cross-axis slot allocated by its parent box.
func NewAlignChild(child oat.Component, h oat.HAlign, v oat.VAlign) *AlignChild {
	return &AlignChild{child: child, hAlign: h, vAlign: v}
}

// GetHAlign satisfies oat.AlignProvider.
func (a *AlignChild) GetHAlign() oat.HAlign { return a.hAlign }

// GetVAlign satisfies oat.AlignProvider.
func (a *AlignChild) GetVAlign() oat.VAlign { return a.vAlign }

// AddChild sets the inner component (satisfies oat.Layout).
func (a *AlignChild) AddChild(c oat.Component) { a.child = c }

// Children satisfies oat.Layout so theme propagation and focus collection
// recurse into the inner component.
func (a *AlignChild) Children() []oat.Component {
	if a.child == nil {
		return nil
	}
	return []oat.Component{a.child}
}

// Measure delegates to the inner component.
func (a *AlignChild) Measure(c oat.Constraint) oat.Size {
	if a.child == nil {
		return oat.Size{}
	}
	return a.child.Measure(c)
}

// Render delegates to the inner component using the full allocated region.
func (a *AlignChild) Render(buf *oat.Buffer, region oat.Region) {
	if a.child == nil {
		return
	}
	a.child.Render(buf, region)
}

// ---- VGap -----------------------------------------------------------------

// VGap is a fixed-height inert spacer for use in a VBox.
// Unlike VFill, VGap always occupies exactly n rows regardless of available
// space — it does not participate in flex distribution.
//
// Use VGap when you want a guaranteed fixed gap between two widgets:
//
//	vbox := layout.NewVBox(
//	    titleWidget,
//	    layout.NewVGap(1),   // always exactly 1 row
//	    bodyWidget,
//	)
//
// If the gap should shrink when space is scarce, use VFill.WithMaxSize instead.
type VGap struct {
	n int
}

// NewVGap creates a fixed-height spacer of n rows.
func NewVGap(n int) *VGap {
	if n < 0 {
		n = 0
	}
	return &VGap{n: n}
}

// Measure returns the fixed height and zero width.
func (g *VGap) Measure(_ oat.Constraint) oat.Size { return oat.Size{Width: 0, Height: g.n} }

// Render is a no-op — VGap is a pure spacer.
func (g *VGap) Render(_ *oat.Buffer, _ oat.Region) {}

// ---- HGap -----------------------------------------------------------------

// HGap is a fixed-width inert spacer for use in an HBox.
// Unlike HFill, HGap always occupies exactly n columns regardless of available
// space — it does not participate in flex distribution.
//
// Use HGap when you want a guaranteed fixed gap between two widgets:
//
//	hbox := layout.NewHBox(
//	    leftWidget,
//	    layout.NewHGap(2),   // always exactly 2 columns
//	    rightWidget,
//	)
//
// If the gap should shrink when space is scarce, use HFill.WithMaxSize instead.
type HGap struct {
	n int
}

// NewHGap creates a fixed-width spacer of n columns.
func NewHGap(n int) *HGap {
	if n < 0 {
		n = 0
	}
	return &HGap{n: n}
}

// Measure returns the fixed width and zero height.
func (g *HGap) Measure(_ oat.Constraint) oat.Size { return oat.Size{Width: g.n, Height: 0} }

// Render is a no-op — HGap is a pure spacer.
func (g *HGap) Render(_ *oat.Buffer, _ oat.Region) {}
