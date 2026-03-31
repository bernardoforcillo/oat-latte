package layout

import (
	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
)

// ---- Padding --------------------------------------------------------------

// Padding wraps a single child and adds configurable space around it.
type Padding struct {
	oat.BaseComponent
	child  oat.Component
	insets oat.Insets
}

// NewPadding wraps child with the given insets.
func NewPadding(child oat.Component, insets oat.Insets) *Padding {
	return &Padding{child: child, insets: insets}
}

// NewPaddingUniform wraps child with uniform padding on all sides.
func NewPaddingUniform(child oat.Component, n int) *Padding {
	return NewPadding(child, oat.Uniform(n))
}

// ApplyTheme is a no-op for Padding — it carries no semantic role in the theme.
func (p *Padding) ApplyTheme(_ latte.Theme) {}

// AddChild sets the single child.
func (p *Padding) AddChild(c oat.Component) { p.child = c }

// Children satisfies oat.Layout.
func (p *Padding) Children() []oat.Component {
	if p.child == nil {
		return nil
	}
	return []oat.Component{p.child}
}

// Measure adds padding to the child's measured size.
func (p *Padding) Measure(c oat.Constraint) oat.Size {
	if p.child == nil {
		return oat.Size{Width: p.insets.Horizontal(), Height: p.insets.Vertical()}
	}
	inner := c.Shrink(p.insets)
	s := p.child.Measure(inner)
	return oat.Size{
		Width:  s.Width + p.insets.Horizontal(),
		Height: s.Height + p.insets.Vertical(),
	}
}

// Render draws the child offset by the padding insets.
func (p *Padding) Render(buf *oat.Buffer, region oat.Region) {
	if p.child == nil {
		return
	}
	innerRegion := region.Inner(p.insets)
	sub := buf.Sub(innerRegion)
	childRegion := oat.Region{Width: innerRegion.Width, Height: innerRegion.Height}
	p.child.Render(sub, childRegion)
}
