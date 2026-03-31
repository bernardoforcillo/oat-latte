package layout

import (
	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
)

// ---- Border ---------------------------------------------------------------

// Border wraps a single child and draws a border around it.
// It reserves one cell on each side for the border lines.
// An optional Title is stamped into the top border line: ╭─ Title ──╮
type Border struct {
	oat.BaseComponent
	child            oat.Component
	titleStyle       latte.Style // style for the title text in the border line
	titleAnchor      oat.Anchor  // horizontal position of the title (default AnchorLeft)
	roundedCorner    bool        // effective value used at render time
	roundedCornerSet bool        // true when caller called WithRoundedCorner explicitly
}

// NewBorder wraps child with a border using the default (single) border style.
// Use the builder methods (WithStyle, WithTitle, WithTitleStyle) to customise.
func NewBorder(child oat.Component) *Border {
	b := &Border{child: child}
	b.Style.Border = latte.BorderSingle
	return b
}

// WithStyle sets a custom style on this Border.
// If style.Border is BorderNone the border type defaults to BorderSingle.
func (b *Border) WithStyle(s latte.Style) *Border {
	b.Style = s
	if b.Style.Border == latte.BorderNone {
		b.Style.Border = latte.BorderSingle
	}
	return b
}

// WithTitle sets the label stamped into the top border rule.
// anchor is an oat.Anchor (H-axis) and is optional; it defaults to
// oat.AnchorLeft when omitted. Pass oat.AnchorCenter or oat.AnchorRight
// to reposition the title horizontally.
//
// Note: Anchor is the horizontal-axis type. There is no vertical title
// placement for Border — the title always appears in the top border row.
// For vertical positioning see oat.VAnchor (used by Divider).
func (b *Border) WithTitle(title string, anchor ...oat.Anchor) *Border {
	b.Title = title
	if len(anchor) > 0 {
		b.titleAnchor = anchor[0]
	}
	return b
}

// WithTitleStyle sets a custom style for the title text (e.g. bold cyan).
// If not set, the border colour is inherited.
func (b *Border) WithTitleStyle(s latte.Style) *Border {
	b.titleStyle = s
	return b
}

// WithRoundedCorner controls whether the border uses rounded corners (╭╮╰╯)
// instead of the default square ones (┌┐└┘).
//
// Once called, this explicit choice overrides the theme's RoundedCorner
// setting for this Border. If the border's resolved style is incompatible
// with arc corners (BorderDouble, BorderThick, BorderDashed) the setting is
// stored but silently ignored at render time — no panic is raised.
// Arc corner codepoints exist only for light-weight strokes (─ │).
func (b *Border) WithRoundedCorner(rounded bool) *Border {
	b.roundedCorner = rounded
	b.roundedCornerSet = true
	return b
}

// ApplyTheme applies Panel and PanelTitle tokens from the theme to this Border.
// The FocusBorder colour is stored in FocusStyle.BorderFG so the focus-aware
// Render logic picks it up automatically.
// If the caller has not called WithRoundedCorner explicitly, the theme's
// RoundedCorner field drives the effective rounded-corner behaviour (both
// enabling and disabling it on theme switch).
func (b *Border) ApplyTheme(t latte.Theme) {
	// Sync rounded-corner preference from the theme whenever the caller has
	// not pinned it with an explicit WithRoundedCorner call.
	if !b.roundedCornerSet {
		b.roundedCorner = t.RoundedCorner
	}

	// Preserve the border shape if the caller set it explicitly, but apply
	// colours from the theme.
	if b.Style.Border != latte.BorderNone {
		b.Style.BorderFG = t.Panel.BorderFG
		b.Style.BG = t.Panel.BG
		b.Style.BorderBG = t.Panel.BG
	} else {
		b.Style = t.Panel
		b.Style.BorderBG = t.Panel.BG
	}
	b.titleStyle = t.PanelTitle
	b.FocusStyle = latte.Style{BorderFG: t.FocusBorder}
}

// AddChild sets the single child (replaces any existing child).
func (b *Border) AddChild(c oat.Component) { b.child = c }

// Children satisfies oat.Layout.
func (b *Border) Children() []oat.Component {
	if b.child == nil {
		return nil
	}
	return []oat.Component{b.child}
}

// Measure adds 2 to each dimension for the border.
func (b *Border) Measure(c oat.Constraint) oat.Size {
	if b.child == nil {
		return oat.Size{}
	}
	inner := c.Shrink(oat.Insets{Top: 1, Right: 1, Bottom: 1, Left: 1})
	s := b.child.Measure(inner)
	return oat.Size{Width: s.Width + 2, Height: s.Height + 2}
}

// Render draws the border (with optional title) and then the child inside it.
// When any descendant component has focus the border colour is promoted to
// the focused border colour (cyan) so the user always knows which panel is active.
func (b *Border) Render(buf *oat.Buffer, region oat.Region) {
	style := b.Style
	// Resolve effective border shape: upgrade Single→Rounded when roundedCorner
	// is true, downgrade Rounded→Single when false.  Incompatible styles
	// (Double, Thick, Dashed) are left untouched in both directions.
	switch style.Border {
	case latte.BorderSingle:
		if b.roundedCorner {
			style.Border = latte.BorderRounded
		}
	case latte.BorderRounded:
		if !b.roundedCorner {
			style.Border = latte.BorderSingle
		}
	}
	if b.child != nil && containsFocus(b.child) {
		// Override the border colour to the focus highlight colour.
		// If a custom FocusStyle was set on the Border itself, use its BorderFG;
		// otherwise fall back to the framework default (bright cyan).
		fg := b.FocusStyle.BorderFG
		if fg == latte.ColorDefault {
			fg = latte.ColorBrightCyan
		}
		style.BorderFG = fg
	}
	sub := buf.Sub(region)
	// Fill the full region with the panel background before drawing the border
	// runes and child. This ensures the interior padding area and any cells not
	// covered by the child have the correct background colour.
	if style.BG != latte.ColorDefault {
		sub.FillBG(style)
	}
	sub.DrawBorderTitle(style.Border, b.Title, b.titleStyle, style, b.titleAnchor)
	if b.child != nil {
		innerRegion := oat.Region{X: 1, Y: 1, Width: region.Width - 2, Height: region.Height - 2}
		b.child.Render(sub, innerRegion)
	}
}
