package layout

import (
	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
)

// breakpoint pairs a minimum-width threshold with the component to show
// when the terminal is at least that wide.
type breakpoint struct {
	minWidth  int
	component oat.Component
}

// Responsive renders different components depending on the current terminal
// width. Breakpoints are evaluated in descending order; the first one whose
// minWidth is ≤ the available width wins. If no breakpoint matches, the
// fallback component is used (or nothing is rendered if fallback is nil).
//
// Usage:
//
//	layout.NewResponsive().
//	    Above(120, threeColumnLayout).   // ≥120 cols → three columns
//	    Above(80,  twoColumnLayout).     // ≥80  cols → two columns
//	    Below(80,  singleColumnLayout)   // <80  cols → single column
type Responsive struct {
	oat.BaseComponent
	breakpoints []breakpoint // sorted descending by minWidth
	fallback    oat.Component
	theme       *latte.Theme
}

// NewResponsive creates an empty Responsive layout.
func NewResponsive() *Responsive {
	r := &Responsive{}
	r.EnsureID()
	return r
}

// WithID sets a user-defined identifier.
func (r *Responsive) WithID(id string) *Responsive { r.ID = id; return r }

// Above registers c to be shown when the terminal width is ≥ minWidth.
// Multiple Above calls are allowed; the widest matching breakpoint wins.
func (r *Responsive) Above(minWidth int, c oat.Component) *Responsive {
	r.breakpoints = append(r.breakpoints, breakpoint{minWidth: minWidth, component: c})
	sortBreakpoints(r.breakpoints)
	return r
}

// Below registers c as the fallback when no Above breakpoint matches.
// Equivalent to Above(0, c) — it always matches as the lowest-priority option.
func (r *Responsive) Below(minWidth int, c oat.Component) *Responsive {
	// Store as a breakpoint with the given minWidth so it participates in the
	// normal selection order, giving the caller precise control.
	r.breakpoints = append(r.breakpoints, breakpoint{minWidth: minWidth, component: c})
	sortBreakpoints(r.breakpoints)
	return r
}

// WithFallback sets the component rendered when no breakpoint matches.
func (r *Responsive) WithFallback(c oat.Component) *Responsive { r.fallback = c; return r }

// ApplyTheme propagates the theme to all registered components.
func (r *Responsive) ApplyTheme(t latte.Theme) {
	r.theme = &t
	for _, bp := range r.breakpoints {
		if tr, ok := bp.component.(oat.ThemeReceiver); ok {
			tr.ApplyTheme(t)
		}
	}
	if r.fallback != nil {
		if tr, ok := r.fallback.(oat.ThemeReceiver); ok {
			tr.ApplyTheme(t)
		}
	}
}

// pick returns the component to render for the given available width.
func (r *Responsive) pick(width int) oat.Component {
	// Breakpoints are sorted descending; first match wins.
	for _, bp := range r.breakpoints {
		if width >= bp.minWidth {
			return bp.component
		}
	}
	return r.fallback
}

// Measure delegates to the selected component for the given constraint.
func (r *Responsive) Measure(c oat.Constraint) oat.Size {
	w := c.MaxWidth
	if w < 0 {
		w = 0
	}
	if comp := r.pick(w); comp != nil {
		return comp.Measure(c)
	}
	return oat.Size{}
}

// Render draws the selected component into region.
func (r *Responsive) Render(buf *oat.Buffer, region oat.Region) {
	comp := r.pick(region.Width)
	if comp == nil {
		return
	}
	comp.Render(buf, region)
}

// Children returns all registered components so theme propagation and focus
// walks can reach them even before a render determines which one is active.
func (r *Responsive) Children() []oat.Component {
	out := make([]oat.Component, 0, len(r.breakpoints)+1)
	for _, bp := range r.breakpoints {
		out = append(out, bp.component)
	}
	if r.fallback != nil {
		out = append(out, r.fallback)
	}
	return out
}

// AddChild appends c as the lowest-priority breakpoint (minWidth=0).
func (r *Responsive) AddChild(c oat.Component) {
	r.Below(0, c)
}

// sortBreakpoints sorts breakpoints in descending order of minWidth so that
// pick() can return the first (widest) matching entry.
func sortBreakpoints(bps []breakpoint) {
	// Insertion sort — breakpoint lists are tiny in practice.
	for i := 1; i < len(bps); i++ {
		for j := i; j > 0 && bps[j].minWidth > bps[j-1].minWidth; j-- {
			bps[j], bps[j-1] = bps[j-1], bps[j]
		}
	}
}
