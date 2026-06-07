package widget

import (
	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
)

// Badge is a compact inline label with a coloured background, used to display
// counts, status indicators, or short tags.
//
// The badge adds one cell of horizontal padding on each side:
//
//	widget.NewBadge("3 errors").WithStyle(latte.Style{BG: latte.Hex("#cc0000"), FG: latte.Hex("#ffffff")})
//	widget.NewBadge("● live").WithStyle(theme.Accent)
type Badge struct {
	oat.BaseComponent
	text        string
	badgeStyle  latte.Style // explicit badge colour; ApplyTheme sets the default
	callerStyle latte.Style // preserved across theme switches
}

// NewBadge creates a Badge displaying text.
func NewBadge(text string) *Badge {
	b := &Badge{text: text}
	b.EnsureID()
	return b
}

// WithID sets a user-defined identifier.
func (b *Badge) WithID(id string) *Badge { b.ID = id; return b }

// WithStyle sets the badge's colours explicitly.
// This takes precedence over the theme's Accent token.
func (b *Badge) WithStyle(s latte.Style) *Badge {
	b.badgeStyle = s
	b.callerStyle = s
	return b
}

// SetText changes the displayed text at runtime.
func (b *Badge) SetText(text string) { b.text = text }

// Text returns the current badge text.
func (b *Badge) Text() string { return b.text }

// ApplyTheme uses the theme's Accent style as the default badge colour.
// An explicit WithStyle call takes precedence.
func (b *Badge) ApplyTheme(t latte.Theme) {
	b.badgeStyle = t.Accent.Merge(b.callerStyle)
}

// Measure returns the size of the badge: text + 2 padding cells wide, 1 tall.
func (b *Badge) Measure(c oat.Constraint) oat.Size {
	w := len([]rune(b.text)) + 2 // one padding cell on each side
	if w < 2 {
		w = 2
	}
	return c.Clamp(oat.Size{Width: w, Height: 1})
}

// Render draws " text " in the badge style.
func (b *Badge) Render(buf *oat.Buffer, region oat.Region) {
	sub := buf.Sub(region)
	sub.FillBG(b.badgeStyle)
	if region.Width < 2 {
		return
	}
	runes := []rune(" " + b.text + " ")
	if len(runes) > region.Width {
		runes = runes[:region.Width]
	}
	sub.DrawText(0, 0, string(runes), b.badgeStyle)
}

// Children satisfies oat.Layout.
func (b *Badge) Children() []oat.Component { return nil }

// AddChild satisfies oat.Layout (no-op).
func (b *Badge) AddChild(_ oat.Component) {}
