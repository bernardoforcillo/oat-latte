package widget

import (
	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// MenuItem is one entry in a menu or submenu.
type MenuItem struct {
	Label    string
	Shortcut string // e.g. "^S", shown right-aligned
	Disabled bool
	Sep      bool     // if true, render as a horizontal separator
	Action   func()
	Sub      []MenuItem // non-empty = submenu
}

// Menu is a top-level entry in the MenuBar.
type Menu struct {
	Label string
	Items []MenuItem
}

// MenuBar renders a horizontal row of menu titles. Activating a menu opens a
// dropdown panel with its items directly below.
//
// Default keybindings (when focused):
//   - ←/→       Move between top-level menus
//   - ↑/↓       Navigate within an open dropdown
//   - Enter      Open menu / trigger item
//   - Esc        Close without selecting
//   - F10 / Alt  Focus/blur the menu bar
//
// Mouse: click a top-level label to open its dropdown; click an item to activate.
type MenuBar struct {
	oat.BaseComponent
	oat.FocusBehavior
	oat.BaseHitRegion

	menus  []Menu
	active int  // index of open menu (-1 = closed)
	cursor int  // highlighted item in open dropdown

	callerStyle   latte.Style
	barStyle      latte.Style
	activeStyle   latte.Style
	dropStyle     latte.Style
	itemStyle     latte.Style
	selStyle      latte.Style
	mutedStyle    latte.Style
	disabledStyle latte.Style

	// Screen positions of each top-level label, set during Render.
	menuX []int
	menuW []int
	barScreenX int
	barScreenY int
}

// NewMenuBar creates a MenuBar with the given menus.
func NewMenuBar(menus ...Menu) *MenuBar {
	mb := &MenuBar{menus: menus, active: -1}
	mb.EnsureID()
	return mb
}

// WithID sets a user-defined identifier.
func (mb *MenuBar) WithID(id string) *MenuBar { mb.ID = id; return mb }

// WithMenus replaces all menus.
func (mb *MenuBar) WithMenus(menus ...Menu) *MenuBar {
	mb.menus = menus
	mb.active = -1
	return mb
}

// ApplyTheme wires theme colours.
func (mb *MenuBar) ApplyTheme(t latte.Theme) {
	mb.Style = t.Text.Merge(mb.callerStyle)
	mb.FocusStyle = latte.Style{BorderFG: t.FocusBorder}
	mb.barStyle = t.Panel
	mb.activeStyle = t.Accent.WithBold()
	mb.dropStyle = t.Panel
	mb.itemStyle = t.Text
	mb.selStyle = t.Accent
	mb.mutedStyle = t.Muted
	mb.disabledStyle = t.Muted
}

// Measure: always 1 row for the bar itself; the dropdown overlays below.
func (mb *MenuBar) Measure(c oat.Constraint) oat.Size {
	return c.Clamp(oat.Size{Width: c.MaxWidth, Height: 1})
}

// Render draws the menu bar and (when a menu is active) its dropdown.
func (mb *MenuBar) Render(buf *oat.Buffer, region oat.Region) {
	style := mb.EffectiveStyle(mb.IsFocused())
	_ = style
	sub := buf.Sub(region)
	mb.SetHitRegion(sub.Region())
	mb.barScreenX = sub.Region().X
	mb.barScreenY = sub.Region().Y
	sub.FillBG(mb.barStyle)

	mb.menuX = make([]int, len(mb.menus))
	mb.menuW = make([]int, len(mb.menus))

	// Draw top-level labels.
	x := 1
	for i, m := range mb.menus {
		label := " " + m.Label + " "
		mb.menuX[i] = x
		mb.menuW[i] = len([]rune(label))

		st := mb.barStyle
		if i == mb.active {
			st = mb.activeStyle
		}
		x += sub.DrawText(x, 0, label, st)
	}

	// Draw the dropdown for the active menu.
	if mb.active >= 0 && mb.active < len(mb.menus) {
		mb.renderDropdown(buf, region, mb.active)
	}
}

func (mb *MenuBar) renderDropdown(buf *oat.Buffer, barRegion oat.Region, menuIdx int) {
	items := mb.menus[menuIdx].Items
	if len(items) == 0 {
		return
	}

	// Compute dropdown dimensions.
	dropW := mb.dropWidth(items)
	dropH := len(items)
	dropX := mb.menuX[menuIdx]
	dropY := 1 // just below the bar

	// Clamp to screen.
	if dropX+dropW > barRegion.Width {
		dropX = barRegion.Width - dropW
	}
	if dropX < 0 {
		dropX = 0
	}
	maxH := barRegion.Height - dropY
	if dropH > maxH {
		dropH = maxH
	}

	sub := buf.Sub(oat.Region{
		X: barRegion.X + dropX, Y: barRegion.Y + dropY,
		Width: dropW, Height: dropH,
	})
	sub.FillBG(mb.dropStyle)

	for row, item := range items {
		if row >= dropH {
			break
		}
		if item.Sep {
			for x := 0; x < dropW; x++ {
				sub.SetCell(x, row, '─', mb.mutedStyle)
			}
			continue
		}
		st := mb.itemStyle
		if item.Disabled {
			st = mb.disabledStyle
		} else if row == mb.cursor {
			st = mb.selStyle
		}
		label := " " + item.Label
		runes := []rune(label)
		for len(runes) < dropW-1 {
			runes = append(runes, ' ')
		}
		if len(runes) > dropW {
			runes = runes[:dropW]
		}
		sub.DrawText(0, row, string(runes), st)
		// Right-align shortcut hint.
		if item.Shortcut != "" {
			hint := item.Shortcut + " "
			hx := dropW - len([]rune(hint))
			if hx > 0 {
				sub.DrawText(hx, row, hint, mb.mutedStyle)
			}
		}
	}
}

func (mb *MenuBar) dropWidth(items []MenuItem) int {
	w := 20
	for _, item := range items {
		n := len([]rune(item.Label)) + 4
		if item.Shortcut != "" {
			n += len([]rune(item.Shortcut)) + 2
		}
		if n > w {
			w = n
		}
	}
	return w
}

// HandleKey processes keyboard input.
func (mb *MenuBar) HandleKey(ev *oat.KeyEvent) bool {
	if mb.active < 0 {
		// Bar is closed — open on Enter/Space/Down.
		switch ev.Key() {
		case tcell.KeyEnter, tcell.KeyDown:
			if len(mb.menus) > 0 {
				mb.open(0)
			}
			return true
		case tcell.KeyRune:
			if ev.Rune() == ' ' {
				if len(mb.menus) > 0 {
					mb.open(0)
				}
				return true
			}
		}
		return false
	}

	switch ev.Key() {
	case tcell.KeyEscape:
		mb.close()
		return true
	case tcell.KeyLeft:
		mb.open((mb.active - 1 + len(mb.menus)) % len(mb.menus))
		return true
	case tcell.KeyRight:
		mb.open((mb.active + 1) % len(mb.menus))
		return true
	case tcell.KeyUp:
		mb.moveCursor(-1)
		return true
	case tcell.KeyDown:
		mb.moveCursor(1)
		return true
	case tcell.KeyEnter:
		mb.activate()
		return true
	}
	return false
}

// HandleMouse handles click on bar labels and dropdown items.
func (mb *MenuBar) HandleMouse(ev *oat.MouseEvent) bool {
	mx, my := ev.Position()

	// Click on the bar row.
	if my == mb.barScreenY {
		if ev.Buttons()&tcell.Button1 == 0 {
			return false
		}
		rx := mx - mb.barScreenX
		for i, x := range mb.menuX {
			if rx >= x && rx < x+mb.menuW[i] {
				if mb.active == i {
					mb.close()
				} else {
					mb.open(i)
				}
				return true
			}
		}
		return false
	}

	// Click inside the dropdown.
	if mb.active >= 0 {
		dropX := mb.barScreenX + mb.menuX[mb.active]
		dropW := mb.dropWidth(mb.menus[mb.active].Items)
		dropY := mb.barScreenY + 1

		if mx >= dropX && mx < dropX+dropW && my >= dropY {
			row := my - dropY
			items := mb.menus[mb.active].Items
			if ev.Buttons()&tcell.Button1 != 0 && row < len(items) {
				item := items[row]
				if !item.Disabled && !item.Sep {
					mb.cursor = row
					mb.activate()
				}
				return true
			}
			return false
		}

		// Click outside dropdown: close.
		if ev.Buttons()&tcell.Button1 != 0 {
			mb.close()
			return true
		}
	}
	return false
}

// KeyBindings advertises shortcuts.
func (mb *MenuBar) KeyBindings() []oat.KeyBinding {
	return []oat.KeyBinding{
		{Key: tcell.KeyLeft, Label: "←→", Description: "Switch menu"},
		{Key: tcell.KeyUp, Label: "↑↓", Description: "Navigate"},
		{Key: tcell.KeyEnter, Label: "Enter", Description: "Select"},
		{Key: tcell.KeyEscape, Label: "Esc", Description: "Close"},
	}
}

// Children satisfies oat.Layout.
func (mb *MenuBar) Children() []oat.Component { return nil }

// AddChild satisfies oat.Layout (no-op).
func (mb *MenuBar) AddChild(_ oat.Component) {}

// ── internal ──────────────────────────────────────────────────────────────────

func (mb *MenuBar) open(idx int) {
	mb.active = idx
	mb.cursor = mb.firstSelectable(mb.menus[idx].Items)
}

func (mb *MenuBar) close() { mb.active = -1 }

func (mb *MenuBar) moveCursor(delta int) {
	if mb.active < 0 {
		return
	}
	items := mb.menus[mb.active].Items
	n := len(items)
	for i := 1; i <= n; i++ {
		next := (mb.cursor + delta*i + n*i) % n
		if !items[next].Sep && !items[next].Disabled {
			mb.cursor = next
			return
		}
	}
}

func (mb *MenuBar) firstSelectable(items []MenuItem) int {
	for i, item := range items {
		if !item.Sep && !item.Disabled {
			return i
		}
	}
	return 0
}

func (mb *MenuBar) activate() {
	if mb.active < 0 {
		return
	}
	items := mb.menus[mb.active].Items
	if mb.cursor < len(items) {
		item := items[mb.cursor]
		mb.close()
		if item.Action != nil {
			item.Action()
		}
	}
}
