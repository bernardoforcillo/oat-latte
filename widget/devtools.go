package widget

import (
	"fmt"
	"reflect"
	"strings"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// DevTools is an F12-toggled overlay panel that displays the component tree,
// component IDs, and basic layout information.
//
// Mount as a persistent overlay and wire up the global key binding:
//
//	dt := widget.NewDevTools()
//	dt.SetRoot(myBody)
//
//	app := oat.NewCanvas(
//	    oat.WithBody(myBody),
//	    oat.WithGlobalKeyBinding(dt.ToKeyBinding()),
//	    oat.WithNotificationManager(/* ... */),
//	)
//	app.ShowPersistentOverlay(dt)
type DevTools struct {
	oat.BaseComponent

	visible   bool
	root      oat.Component
	scrollOff int
	lines     []string // cached component-tree dump

	panelStyle  latte.Style
	titleStyle  latte.Style
	normalStyle latte.Style
	mutedStyle  latte.Style
}

// NewDevTools creates a DevTools overlay (initially hidden).
func NewDevTools() *DevTools {
	dt := &DevTools{}
	dt.EnsureID()
	dt.rebuildDefaultStyles()
	return dt
}

// WithID sets a user-defined identifier on this component.
func (dt *DevTools) WithID(id string) *DevTools { dt.ID = id; return dt }

// Toggle flips the visibility of the DevTools panel.
func (dt *DevTools) Toggle() {
	dt.visible = !dt.visible
	if dt.visible {
		dt.rebuild()
	}
}

// IsVisible reports whether the DevTools panel is currently shown.
func (dt *DevTools) IsVisible() bool { return dt.visible }

// SetRoot registers the application's body component so the DevTools panel
// can walk its subtree to produce the component-tree dump.
func (dt *DevTools) SetRoot(c oat.Component) {
	dt.root = c
	if dt.visible {
		dt.rebuild()
	}
}

// ToKeyBinding returns a KeyBinding that toggles this overlay when F12 is pressed.
// Register it with oat.WithGlobalKeyBinding(dt.ToKeyBinding()).
func (dt *DevTools) ToKeyBinding() oat.KeyBinding {
	return oat.KeyBinding{
		Key:         tcell.KeyF12,
		Label:       "F12",
		Description: "DevTools",
		Handler:     dt.Toggle,
	}
}

// ApplyTheme applies semantic theme tokens to the DevTools panel.
func (dt *DevTools) ApplyTheme(t latte.Theme) {
	dt.panelStyle = latte.Style{
		FG: t.Text.FG,
		BG: t.Muted.BG,
	}
	if dt.panelStyle.BG == latte.ColorDefault {
		dt.panelStyle.BG = t.Text.BG
	}
	dt.titleStyle = latte.Style{FG: t.Accent.FG, BG: dt.panelStyle.BG, Bold: true}
	dt.normalStyle = latte.Style{FG: t.Text.FG, BG: dt.panelStyle.BG}
	dt.mutedStyle = latte.Style{FG: t.Muted.FG, BG: dt.panelStyle.BG}
}

// rebuildDefaultStyles sets neutral fallback styles.
func (dt *DevTools) rebuildDefaultStyles() {
	bg := latte.RGB(30, 30, 50)
	dt.panelStyle = latte.Style{FG: latte.ColorBrightWhite, BG: bg}
	dt.titleStyle = latte.Style{FG: latte.ColorBrightCyan, BG: bg, Bold: true}
	dt.normalStyle = latte.Style{FG: latte.ColorBrightWhite, BG: bg}
	dt.mutedStyle = latte.Style{FG: latte.ColorBrightBlack, BG: bg}
}

// Measure returns {0,0} when hidden, otherwise requests a right-side panel.
func (dt *DevTools) Measure(c oat.Constraint) oat.Size {
	if !dt.visible {
		return oat.Size{}
	}
	w := c.MaxWidth * 2 / 3
	if w < 40 {
		w = 40
	}
	if c.MaxWidth >= 0 && w > c.MaxWidth {
		w = c.MaxWidth
	}
	h := c.MaxHeight
	if h < 0 {
		h = 0
	}
	return oat.Size{Width: w, Height: h}
}

// Render draws the DevTools panel on the right side of the screen when visible.
func (dt *DevTools) Render(buf *oat.Buffer, region oat.Region) {
	if !dt.visible {
		return
	}

	// Panel occupies the right 2/5 of the region (at least 40 columns wide).
	panelW := region.Width * 2 / 5
	if panelW < 40 {
		panelW = 40
	}
	if panelW > region.Width {
		panelW = region.Width
	}
	panelH := region.Height
	startX := region.Width - panelW

	panelRegion := oat.Region{X: startX, Y: 0, Width: panelW, Height: panelH}
	sub := buf.Sub(panelRegion)
	sub.FillBG(dt.panelStyle)
	sub.DrawBorderTitle(latte.BorderSingle, "DevTools (F12)", dt.titleStyle, dt.panelStyle, oat.AnchorLeft)

	// Inner area (inside the border).
	if panelW < 4 || panelH < 4 {
		return
	}
	inner := sub.Sub(oat.Region{X: 1, Y: 1, Width: panelW - 2, Height: panelH - 2})

	if dt.root != nil {
		dt.rebuild()
	}

	for y, line := range dt.lines {
		idx := dt.scrollOff + y
		if idx >= len(dt.lines) {
			break
		}
		if y >= panelH-2 {
			break
		}
		style := dt.normalStyle
		if strings.HasPrefix(strings.TrimSpace(line), "//") {
			style = dt.mutedStyle
		}
		inner.DrawText(0, y, line, style)
	}
}

// Children satisfies oat.Layout; DevTools has no sub-components.
func (dt *DevTools) Children() []oat.Component { return nil }

// AddChild is a no-op for DevTools.
func (dt *DevTools) AddChild(_ oat.Component) {}

// rebuild regenerates the cached component-tree dump from dt.root.
func (dt *DevTools) rebuild() {
	dt.lines = nil
	if dt.root == nil {
		dt.lines = []string{"(no root set)"}
		return
	}
	header := "Component Tree"
	separator := strings.Repeat("─", len(header))
	dt.lines = append(dt.lines, header, separator)
	dt.lines = append(dt.lines, dumpTree(dt.root, 0)...)
}

// dumpTree recursively walks the component subtree rooted at c and returns
// one descriptive string per component.
func dumpTree(c oat.Component, depth int) []string {
	if c == nil {
		return nil
	}
	indent := strings.Repeat("  ", depth)

	// Derive a display name via reflection.
	typeName := "Unknown"
	rv := reflect.ValueOf(c)
	if rv.Kind() == reflect.Ptr && !rv.IsNil() {
		typeName = rv.Elem().Type().Name()
		if typeName == "" {
			typeName = rv.Type().String()
		}
	} else if rv.IsValid() {
		typeName = rv.Type().String()
	}

	// Show ID if the component exposes one.
	idPart := ""
	if ider, ok := c.(oat.IDer); ok {
		if id := ider.GetID(); id != "" {
			idPart = fmt.Sprintf(" id=%s", id)
		}
	}

	line := indent + typeName + idPart
	result := []string{line}

	// Recurse into children.
	if layout, ok := c.(oat.Layout); ok {
		for _, child := range layout.Children() {
			result = append(result, dumpTree(child, depth+1)...)
		}
	}
	return result
}
