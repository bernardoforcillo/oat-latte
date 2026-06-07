package widget

import (
	"strings"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// BreadcrumbItem is a single segment of a Breadcrumb navigation trail.
type BreadcrumbItem struct {
	Label string
	Value interface{}
}

// Breadcrumb renders a navigation trail like  Home › Projects › Feature
// and fires onChange when the user clicks or selects a segment.
//
// Default keybindings (when focused):
//   - ←/→    Move selection between segments
//   - Enter   Confirm current segment
type Breadcrumb struct {
	oat.BaseComponent
	oat.FocusBehavior
	oat.BaseHitRegion

	items    []BreadcrumbItem
	sep      string // default " › "
	active   int    // currently highlighted segment (-1 = none)
	onChange func(index int, item BreadcrumbItem)

	callerStyle    latte.Style
	segmentStyle   latte.Style
	activeStyle    latte.Style
	sepStyle       latte.Style

	// Screen X positions of each segment's first cell (set during Render).
	segX     []int
	segW     []int
	screenX0 int
}

// NewBreadcrumb creates an empty Breadcrumb with the default separator " › ".
func NewBreadcrumb(items ...BreadcrumbItem) *Breadcrumb {
	b := &Breadcrumb{
		items:  items,
		sep:    " › ",
		active: len(items) - 1,
	}
	b.EnsureID()
	return b
}

// WithID sets a user-defined identifier.
func (b *Breadcrumb) WithID(id string) *Breadcrumb { b.ID = id; return b }

// WithSep sets the separator string between segments.
func (b *Breadcrumb) WithSep(sep string) *Breadcrumb { b.sep = sep; return b }

// WithItems replaces the item list.
func (b *Breadcrumb) WithItems(items ...BreadcrumbItem) *Breadcrumb {
	b.items = items
	b.active = len(items) - 1
	return b
}

// Push appends a new segment (navigation forward).
func (b *Breadcrumb) Push(item BreadcrumbItem) {
	b.items = append(b.items, item)
	b.active = len(b.items) - 1
}

// Pop removes the last segment and returns it (navigation back).
func (b *Breadcrumb) Pop() (BreadcrumbItem, bool) {
	if len(b.items) == 0 {
		return BreadcrumbItem{}, false
	}
	item := b.items[len(b.items)-1]
	b.items = b.items[:len(b.items)-1]
	b.active = len(b.items) - 1
	return item, true
}

// Items returns the current item slice.
func (b *Breadcrumb) Items() []BreadcrumbItem { return b.items }

// WithOnChange registers a callback fired when a segment is selected.
func (b *Breadcrumb) WithOnChange(fn func(index int, item BreadcrumbItem)) *Breadcrumb {
	b.onChange = fn
	return b
}

// ApplyTheme wires theme colours.
func (b *Breadcrumb) ApplyTheme(t latte.Theme) {
	b.Style = t.Text.Merge(b.callerStyle)
	b.FocusStyle = latte.Style{BorderFG: t.FocusBorder}
	b.segmentStyle = t.Text
	b.activeStyle = t.Accent.WithBold()
	b.sepStyle = t.Muted
}

// Measure returns the total width of all segments + separators.
func (b *Breadcrumb) Measure(c oat.Constraint) oat.Size {
	w := b.totalWidth()
	return c.Clamp(oat.Size{Width: w, Height: 1})
}

func (b *Breadcrumb) totalWidth() int {
	w := 0
	for i, item := range b.items {
		w += len([]rune(item.Label))
		if i < len(b.items)-1 {
			w += len([]rune(b.sep))
		}
	}
	return w
}

// Render draws the breadcrumb trail.
func (b *Breadcrumb) Render(buf *oat.Buffer, region oat.Region) {
	style := b.EffectiveStyle(b.IsFocused())
	sub := buf.Sub(region)
	b.SetHitRegion(sub.Region())
	b.screenX0 = sub.Region().X
	sub.FillBG(style)

	b.segX = make([]int, len(b.items))
	b.segW = make([]int, len(b.items))

	x := 0
	for i, item := range b.items {
		b.segX[i] = x
		lw := len([]rune(item.Label))
		b.segW[i] = lw

		st := b.segmentStyle
		if i == b.active {
			st = b.activeStyle
		}
		x += sub.DrawText(x, 0, item.Label, st)

		if i < len(b.items)-1 {
			x += sub.DrawText(x, 0, b.sep, b.sepStyle)
		}
	}
}

// HandleKey navigates between segments.
func (b *Breadcrumb) HandleKey(ev *oat.KeyEvent) bool {
	switch ev.Key() {
	case tcell.KeyLeft:
		if b.active > 0 {
			b.active--
		}
		return true
	case tcell.KeyRight:
		if b.active < len(b.items)-1 {
			b.active++
		}
		return true
	case tcell.KeyEnter:
		b.confirm()
		return true
	}
	return false
}

// HandleMouse handles clicks on segments.
func (b *Breadcrumb) HandleMouse(ev *oat.MouseEvent) bool {
	if ev.Buttons()&tcell.Button1 == 0 {
		return false
	}
	mx, my := ev.Position()
	_ = my
	if !b.ContainsScreenPos(mx, my) {
		return false
	}
	rx := mx - b.screenX0
	for i, x := range b.segX {
		if rx >= x && rx < x+b.segW[i] {
			b.active = i
			b.confirm()
			return true
		}
	}
	return false
}

func (b *Breadcrumb) confirm() {
	if b.active >= 0 && b.active < len(b.items) && b.onChange != nil {
		b.onChange(b.active, b.items[b.active])
	}
}

// KeyBindings advertises shortcuts.
func (b *Breadcrumb) KeyBindings() []oat.KeyBinding {
	return []oat.KeyBinding{
		{Key: tcell.KeyLeft, Label: "←→", Description: "Navigate"},
		{Key: tcell.KeyEnter, Label: "Enter", Description: "Select"},
	}
}

// Children satisfies oat.Layout.
func (b *Breadcrumb) Children() []oat.Component { return nil }

// AddChild satisfies oat.Layout (no-op).
func (b *Breadcrumb) AddChild(_ oat.Component) {}

// String returns the breadcrumb as a plain string.
func (b *Breadcrumb) String() string {
	labels := make([]string, len(b.items))
	for i, item := range b.items {
		labels[i] = item.Label
	}
	return strings.Join(labels, b.sep)
}
