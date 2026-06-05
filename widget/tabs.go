package widget

import (
	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// TabPosition controls whether the tab bar is rendered above or below the
// content area.
type TabPosition int

const (
	TabsTop    TabPosition = iota // tab bar above content (default)
	TabsBottom                    // tab bar below content
)

// Tab is a single tab entry: a string label and an associated content component.
type Tab struct {
	Label   string
	Content oat.Component
}

// Tabs is a focusable tab container with labeled panels.
//
// Default keybindings (when focused):
//   - ← / Shift+Tab  Previous — move to the previous tab (wraps)
//   - → / Tab        Next — move to the next tab (wraps)
type Tabs struct {
	oat.BaseComponent
	oat.FocusBehavior
	oat.BaseHitRegion

	tabs     []Tab
	active   int
	position TabPosition

	accentStyle latte.Style
	mutedStyle  latte.Style

	callerStyle      latte.Style
	callerFocusStyle latte.Style
}

// NewTabs creates a Tabs widget pre-populated with the given tabs.
func NewTabs(tabs ...Tab) *Tabs {
	t := &Tabs{
		tabs:   append([]Tab{}, tabs...),
		active: 0,
	}
	t.EnsureID()
	return t
}

// WithID sets a user-defined identifier on this component.
func (t *Tabs) WithID(id string) *Tabs { t.ID = id; return t }

// WithStyle sets the base display style for the Tabs widget.
func (t *Tabs) WithStyle(s latte.Style) *Tabs {
	t.Style = s
	t.callerStyle = s
	return t
}

// WithPosition sets whether the tab bar appears above (TabsTop) or below
// (TabsBottom) the content area.
func (t *Tabs) WithPosition(p TabPosition) *Tabs { t.position = p; return t }

// AddTab appends a new tab with the given label and content.
func (t *Tabs) AddTab(label string, content oat.Component) *Tabs {
	t.tabs = append(t.tabs, Tab{Label: label, Content: content})
	return t
}

// ActiveIndex returns the index of the currently active tab.
func (t *Tabs) ActiveIndex() int { return t.active }

// SetActive selects the tab at index i, clamped to the valid range.
func (t *Tabs) SetActive(i int) {
	t.active = clamp(i, 0, len(t.tabs)-1)
}

// ActiveContent returns the content component of the currently active tab,
// or nil if there are no tabs.
func (t *Tabs) ActiveContent() oat.Component {
	if t.active < 0 || t.active >= len(t.tabs) {
		return nil
	}
	return t.tabs[t.active].Content
}

// ApplyTheme applies theme tokens to the Tabs widget and propagates the theme
// to all tab content components that implement oat.ThemeReceiver.
func (t *Tabs) ApplyTheme(theme latte.Theme) {
	t.Style = theme.Text.Merge(t.callerStyle)
	t.FocusStyle = latte.Style{BorderFG: theme.FocusBorder}
	t.accentStyle = theme.Accent
	t.mutedStyle = theme.Muted
	for _, tab := range t.tabs {
		if tr, ok := tab.Content.(oat.ThemeReceiver); ok {
			tr.ApplyTheme(theme)
		}
	}
}

// Children implements oat.Layout. Returns the active tab's content component
// so that focus walks and theme propagation descend into it.
func (t *Tabs) Children() []oat.Component {
	c := t.ActiveContent()
	if c == nil {
		return nil
	}
	return []oat.Component{c}
}

// AddChild implements oat.Layout. Appends a tab with an empty label.
func (t *Tabs) AddChild(c oat.Component) {
	t.tabs = append(t.tabs, Tab{Label: "", Content: c})
}

func (t *Tabs) Measure(c oat.Constraint) oat.Size {
	tabBarH := 1
	contentH := 0
	contentW := 0
	if content := t.ActiveContent(); content != nil {
		inner := oat.Constraint{
			MaxWidth:  c.MaxWidth,
			MaxHeight: -1,
		}
		if c.MaxHeight >= 0 {
			inner.MaxHeight = c.MaxHeight - tabBarH
			if inner.MaxHeight < 0 {
				inner.MaxHeight = 0
			}
		}
		sz := content.Measure(inner)
		contentH = sz.Height
		contentW = sz.Width
	}
	w := c.MaxWidth
	if w < 0 {
		w = contentW
	}
	return oat.Size{Width: w, Height: tabBarH + contentH}
}

func (t *Tabs) Render(buf *oat.Buffer, region oat.Region) {
	style := t.EffectiveStyle(t.IsFocused())
	sub := buf.Sub(region)
	t.SetHitRegion(sub.Region())
	sub.FillBG(style)

	if region.Height == 0 {
		return
	}

	tabBarH := 1
	var barY int
	var contentRegion oat.Region

	switch t.position {
	case TabsBottom:
		barY = region.Height - tabBarH
		contentRegion = oat.Region{X: 0, Y: 0, Width: region.Width, Height: region.Height - tabBarH}
	default: // TabsTop
		barY = 0
		contentRegion = oat.Region{X: 0, Y: tabBarH, Width: region.Width, Height: region.Height - tabBarH}
	}

	// Draw the tab bar.
	x := 0
	for i, tab := range t.tabs {
		label := "[ " + tab.Label + " ]"
		labelRunes := []rune(label)

		var tabStyle latte.Style
		if i == t.active {
			tabStyle = t.accentStyle
			tabStyle.Bold = true
			if t.IsFocused() {
				tabStyle = style.Merge(t.accentStyle)
				tabStyle.Bold = true
			}
		} else {
			tabStyle = t.mutedStyle
		}

		for _, r := range labelRunes {
			if x >= region.Width {
				break
			}
			sub.SetCell(x, barY, r, tabStyle)
			x++
		}
		// Separator space between tabs.
		if x < region.Width && i < len(t.tabs)-1 {
			sub.SetCell(x, barY, ' ', style)
			x++
		}
		if x >= region.Width {
			break
		}
	}
	// Fill the remainder of the tab bar row with the base style.
	for ; x < region.Width; x++ {
		sub.SetCell(x, barY, ' ', style)
	}

	// Render active content using the standard sub-buffer pattern.
	if contentRegion.Height > 0 && contentRegion.Width > 0 {
		if content := t.ActiveContent(); content != nil {
			content.Render(sub, contentRegion)
		}
	}
}

func (t *Tabs) HandleKey(ev *oat.KeyEvent) bool {
	switch ev.Key() {
	case tcell.KeyLeft, tcell.KeyBacktab:
		if len(t.tabs) == 0 {
			return true
		}
		t.active = (t.active - 1 + len(t.tabs)) % len(t.tabs)
		return true
	case tcell.KeyRight, tcell.KeyTab:
		if len(t.tabs) == 0 {
			return true
		}
		t.active = (t.active + 1) % len(t.tabs)
		return true
	}
	return false
}

func (t *Tabs) KeyBindings() []oat.KeyBinding {
	return []oat.KeyBinding{
		{Key: tcell.KeyLeft, Label: "←", Description: "Previous"},
		{Key: tcell.KeyRight, Label: "→", Description: "Next"},
	}
}
