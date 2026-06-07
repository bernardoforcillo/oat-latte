package layout

import (
	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// SplitDirection controls whether the SplitPane divides its space horizontally
// (two columns) or vertically (two rows).
type SplitDirection int

const (
	// SplitHorizontal places first on the left and second on the right,
	// separated by a vertical '│' divider.
	SplitHorizontal SplitDirection = iota
	// SplitVertical places first on the top and second on the bottom,
	// separated by a horizontal '─' divider.
	SplitVertical
)

// SplitPane is a two-panel layout with a resizable divider.
// The divider occupies exactly one cell. Panels are sized according to ratio
// (proportion for the first panel) and clamped by minFirst / minSecond.
//
// Default keybindings (when focused):
//   - Alt+←  Shrink first panel (horizontal split)
//   - Alt+→  Grow  first panel (horizontal split)
//   - Alt+↑  Shrink first panel (vertical split)
//   - Alt+↓  Grow  first panel (vertical split)
//
// Mouse: click-drag the divider bar to resize interactively.
type SplitPane struct {
	oat.BaseComponent
	oat.FocusBehavior
	oat.BaseHitRegion

	first     oat.Component
	second    oat.Component
	dir       SplitDirection
	ratio     float64 // proportion for first panel; clamped to [0.1, 0.9]
	minFirst  int     // minimum cells for first panel
	minSecond int     // minimum cells for second panel
	divStyle  latte.Style

	// Screen coords of the divider and the whole region, set during Render.
	divScreenPos    int // absolute X (horizontal) or Y (vertical) of divider
	regionScreenPos int // absolute X (horizontal) or Y (vertical) of region start
	regionSize      int // Width (horizontal) or Height (vertical) of region
	dragging        bool
}

// NewSplitPane creates a SplitPane with the given panels and direction.
// The initial ratio is 0.5 (equal split). Use WithRatio to customise.
func NewSplitPane(first, second oat.Component, dir SplitDirection) *SplitPane {
	sp := &SplitPane{
		first:     first,
		second:    second,
		dir:       dir,
		ratio:     0.5,
		minFirst:  5,
		minSecond: 5,
	}
	sp.EnsureID()
	return sp
}

// WithRatio sets the proportion of available space given to the first panel.
// The value is clamped to [0.1, 0.9].
func (sp *SplitPane) WithRatio(r float64) *SplitPane {
	sp.ratio = clampFloat(r, 0.1, 0.9)
	return sp
}

// WithMinSizes sets the minimum cell count for each panel.
func (sp *SplitPane) WithMinSizes(minFirst, minSecond int) *SplitPane {
	sp.minFirst = minFirst
	sp.minSecond = minSecond
	return sp
}

// WithDividerStyle sets the style used to draw the divider line.
func (sp *SplitPane) WithDividerStyle(s latte.Style) *SplitPane {
	sp.divStyle = s
	return sp
}

// ApplyTheme applies the Muted token as the divider style and propagates the
// theme to both child panels.
func (sp *SplitPane) ApplyTheme(t latte.Theme) {
	sp.divStyle = t.Muted
	if tr, ok := sp.first.(oat.ThemeReceiver); ok {
		tr.ApplyTheme(t)
	}
	if tr, ok := sp.second.(oat.ThemeReceiver); ok {
		tr.ApplyTheme(t)
	}
}

// Measure returns the preferred size of the SplitPane.
// For SplitHorizontal the width is the sum of both panels plus 1 (divider),
// the height is the taller of the two.
// For SplitVertical the height is the sum plus 1, the width is the wider.
func (sp *SplitPane) Measure(c oat.Constraint) oat.Size {
	var s1, s2 oat.Size
	if sp.first != nil {
		s1 = sp.first.Measure(c)
	}
	if sp.second != nil {
		s2 = sp.second.Measure(c)
	}

	switch sp.dir {
	case SplitHorizontal:
		w := c.MaxWidth
		if w < 0 {
			w = s1.Width + 1 + s2.Width
		}
		h := s1.Height
		if s2.Height > h {
			h = s2.Height
		}
		if c.MaxHeight >= 0 && h > c.MaxHeight {
			h = c.MaxHeight
		}
		return oat.Size{Width: w, Height: h}

	default: // SplitVertical
		h := c.MaxHeight
		if h < 0 {
			h = s1.Height + 1 + s2.Height
		}
		w := s1.Width
		if s2.Width > w {
			w = s2.Width
		}
		if c.MaxWidth >= 0 && w > c.MaxWidth {
			w = c.MaxWidth
		}
		return oat.Size{Width: w, Height: h}
	}
}

// Render draws both panels and the divider into buf within region.
func (sp *SplitPane) Render(buf *oat.Buffer, region oat.Region) {
	sub := buf.Sub(region)
	sp.SetHitRegion(sub.Region())

	switch sp.dir {
	case SplitHorizontal:
		available := region.Width - 1 // 1 cell for divider
		if available < 0 {
			available = 0
		}
		firstW := int(float64(available) * sp.ratio)
		firstW = clampInt(firstW, sp.minFirst, available-sp.minSecond)
		secondW := available - firstW

		sp.divScreenPos = sub.Region().X + firstW
		sp.regionScreenPos = sub.Region().X
		sp.regionSize = region.Width

		// Render first panel.
		if sp.first != nil && firstW > 0 {
			sp.first.Render(sub, oat.Region{X: 0, Y: 0, Width: firstW, Height: region.Height})
		}

		// Draw vertical divider (highlighted during drag).
		divStyle := sp.divStyle
		if sp.dragging {
			divStyle.Bold = true
		}
		divX := firstW
		for y := 0; y < region.Height; y++ {
			sub.SetCell(divX, y, '│', divStyle)
		}

		// Render second panel.
		if sp.second != nil && secondW > 0 {
			sp.second.Render(sub, oat.Region{X: firstW + 1, Y: 0, Width: secondW, Height: region.Height})
		}

	default: // SplitVertical
		available := region.Height - 1 // 1 cell for divider
		if available < 0 {
			available = 0
		}
		firstH := int(float64(available) * sp.ratio)
		firstH = clampInt(firstH, sp.minFirst, available-sp.minSecond)
		secondH := available - firstH

		sp.divScreenPos = sub.Region().Y + firstH
		sp.regionScreenPos = sub.Region().Y
		sp.regionSize = region.Height

		// Render first panel.
		if sp.first != nil && firstH > 0 {
			sp.first.Render(sub, oat.Region{X: 0, Y: 0, Width: region.Width, Height: firstH})
		}

		// Draw horizontal divider.
		divStyle := sp.divStyle
		if sp.dragging {
			divStyle.Bold = true
		}
		divY := firstH
		for x := 0; x < region.Width; x++ {
			sub.SetCell(x, divY, '─', divStyle)
		}

		// Render second panel.
		if sp.second != nil && secondH > 0 {
			sp.second.Render(sub, oat.Region{X: 0, Y: firstH + 1, Width: region.Width, Height: secondH})
		}
	}
}

// HandleMouse handles drag-resize of the divider.
// Clicking on the divider and then dragging updates the ratio in real time.
func (sp *SplitPane) HandleMouse(ev *oat.MouseEvent) bool {
	mx, my := ev.Position()

	if ev.Buttons()&tcell.Button1 != 0 {
		// Button held — check if on divider or already dragging.
		if sp.dir == SplitHorizontal {
			onDiv := mx == sp.divScreenPos
			if sp.dragging || onDiv {
				sp.dragging = true
				available := sp.regionSize - 1
				if available > 0 {
					newFirst := mx - sp.regionScreenPos
					sp.ratio = clampFloat(float64(newFirst)/float64(available), 0.1, 0.9)
				}
				return true
			}
		} else {
			onDiv := my == sp.divScreenPos
			if sp.dragging || onDiv {
				sp.dragging = true
				available := sp.regionSize - 1
				if available > 0 {
					newFirst := my - sp.regionScreenPos
					sp.ratio = clampFloat(float64(newFirst)/float64(available), 0.1, 0.9)
				}
				return true
			}
		}
	} else {
		// Button released — end drag.
		if sp.dragging {
			sp.dragging = false
			return true
		}
	}
	return false
}

// HandleKey processes keyboard input for resizing the split.
// Alt+← / Alt+→ adjust a horizontal split; Alt+↑ / Alt+↓ adjust a vertical split.
// Tab is never consumed so Canvas can cycle focus normally.
func (sp *SplitPane) HandleKey(ev *oat.KeyEvent) bool {
	alt := ev.Modifiers()&tcell.ModAlt != 0
	if !alt {
		return false
	}

	switch sp.dir {
	case SplitHorizontal:
		switch ev.Key() {
		case tcell.KeyLeft:
			sp.ratio = clampFloat(sp.ratio-0.05, 0.1, 0.9)
			return true
		case tcell.KeyRight:
			sp.ratio = clampFloat(sp.ratio+0.05, 0.1, 0.9)
			return true
		}

	default: // SplitVertical
		switch ev.Key() {
		case tcell.KeyUp:
			sp.ratio = clampFloat(sp.ratio-0.05, 0.1, 0.9)
			return true
		case tcell.KeyDown:
			sp.ratio = clampFloat(sp.ratio+0.05, 0.1, 0.9)
			return true
		}
	}

	return false
}

// KeyBindings advertises the resize shortcuts to the StatusBar.
func (sp *SplitPane) KeyBindings() []oat.KeyBinding {
	if sp.dir == SplitHorizontal {
		return []oat.KeyBinding{
			{Key: tcell.KeyLeft, Mod: tcell.ModAlt, Label: "M-←", Description: "Shrink"},
			{Key: tcell.KeyRight, Mod: tcell.ModAlt, Label: "M-→", Description: "Grow"},
		}
	}
	return []oat.KeyBinding{
		{Key: tcell.KeyUp, Mod: tcell.ModAlt, Label: "M-↑", Description: "Shrink"},
		{Key: tcell.KeyDown, Mod: tcell.ModAlt, Label: "M-↓", Description: "Grow"},
	}
}

// Children satisfies oat.Layout so theme propagation and focus collection
// recurse into both panels.
func (sp *SplitPane) Children() []oat.Component {
	out := make([]oat.Component, 0, 2)
	if sp.first != nil {
		out = append(out, sp.first)
	}
	if sp.second != nil {
		out = append(out, sp.second)
	}
	return out
}

// AddChild sets first if it is nil, otherwise sets second.
func (sp *SplitPane) AddChild(c oat.Component) {
	if sp.first == nil {
		sp.first = c
	} else {
		sp.second = c
	}
}

// ---- helpers ----------------------------------------------------------------

// clampFloat clamps v to [lo, hi].
func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// clampInt clamps v to [lo, hi].
func clampInt(v, lo, hi int) int {
	if hi < lo {
		hi = lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
