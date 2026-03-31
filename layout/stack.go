package layout

import (
	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
)

// ---- Stack ----------------------------------------------------------------

// Stack layers children on top of each other (Z-axis).
// The last child renders on top. Used for modals and overlays.
type Stack struct {
	oat.BaseComponent
	children []oat.Component
}

// NewStack creates a Stack with optional children.
func NewStack(children ...oat.Component) *Stack {
	return &Stack{children: children}
}

// WithStyle sets the style.
func (s *Stack) WithStyle(st latte.Style) *Stack { s.Style = st; return s }

// ApplyTheme is a no-op for Stack — it carries no semantic role in the theme.
func (s *Stack) ApplyTheme(_ latte.Theme) {}

// AddChild appends a child layer.
func (s *Stack) AddChild(c oat.Component) { s.children = append(s.children, c) }

// Children satisfies oat.Layout.
func (s *Stack) Children() []oat.Component { return s.children }

// Measure returns the size of the largest child.
func (s *Stack) Measure(c oat.Constraint) oat.Size {
	maxW, maxH := 0, 0
	for _, child := range s.children {
		sz := child.Measure(c)
		if sz.Width > maxW {
			maxW = sz.Width
		}
		if sz.Height > maxH {
			maxH = sz.Height
		}
	}
	return oat.Size{Width: maxW, Height: maxH}
}

// Render draws each child in the full region (later children paint over earlier ones).
func (s *Stack) Render(buf *oat.Buffer, region oat.Region) {
	for _, child := range s.children {
		child.Render(buf, region)
	}
}
