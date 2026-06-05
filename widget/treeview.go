package widget

import (
	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// TreeNode is a single node in a TreeView hierarchy.
// Nodes may have child nodes, which can be expanded or collapsed.
type TreeNode struct {
	Label    string
	Value    interface{}
	Children []*TreeNode
	expanded bool
}

// NewTreeNode creates a new TreeNode with the given label and value.
func NewTreeNode(label string, value interface{}) *TreeNode {
	return &TreeNode{Label: label, Value: value}
}

// AddChild appends child to this node's children and returns n for chaining.
func (n *TreeNode) AddChild(child *TreeNode) *TreeNode {
	n.Children = append(n.Children, child)
	return n
}

// IsExpanded reports whether this node is currently expanded.
func (n *TreeNode) IsExpanded() bool { return n.expanded }

// Expand marks the node as expanded.
func (n *TreeNode) Expand() { n.expanded = true }

// Collapse marks the node as collapsed.
func (n *TreeNode) Collapse() { n.expanded = false }

// Toggle flips the node's expanded state.
func (n *TreeNode) Toggle() { n.expanded = !n.expanded }

// flatEntry is a single visible row in the flattened tree list.
type flatEntry struct {
	node  *TreeNode
	depth int
}

// TreeView is an expandable tree widget.
//
// Default keybindings (when focused):
//   - ↑ / ↓      Navigate up and down
//   - →  / Enter  Expand a collapsed node (with children)
//   - ←  / Enter  Collapse an expanded node (with children)
//   - Enter        Invoke onSelect callback on a leaf node
type TreeView struct {
	oat.BaseComponent
	oat.FocusBehavior
	oat.BaseHitRegion

	roots     []*TreeNode
	flatList  []*flatEntry
	dirty     bool
	cursor    int
	scrollOff int
	onSelect  func(*TreeNode)

	// callerStyle is the style set directly by the caller before theme application.
	callerStyle latte.Style
	// selectedStyle is the row style for the cursor row when focused.
	selectedStyle latte.Style
	// normalStyle is the row style for unfocused rows.
	normalStyle latte.Style
}

// NewTreeView creates an empty, focusable TreeView.
func NewTreeView() *TreeView {
	tv := &TreeView{dirty: true}
	tv.EnsureID()
	return tv
}

// WithID sets a user-defined identifier on this component.
func (tv *TreeView) WithID(id string) *TreeView { tv.ID = id; return tv }

// WithOnSelect registers a callback invoked when a node is selected (Enter on a leaf).
func (tv *TreeView) WithOnSelect(fn func(*TreeNode)) *TreeView {
	tv.onSelect = fn
	return tv
}

// AddRoot appends a root node to the tree and returns tv for chaining.
func (tv *TreeView) AddRoot(node *TreeNode) *TreeView {
	tv.roots = append(tv.roots, node)
	tv.dirty = true
	return tv
}

// SetRoots replaces all root nodes with the given slice.
func (tv *TreeView) SetRoots(nodes []*TreeNode) {
	tv.roots = nodes
	tv.dirty = true
	tv.cursor = 0
	tv.scrollOff = 0
}

// SelectedNode returns the currently highlighted node, or nil if the tree is empty.
func (tv *TreeView) SelectedNode() *TreeNode {
	tv.rebuildIfDirty()
	if tv.cursor < 0 || tv.cursor >= len(tv.flatList) {
		return nil
	}
	return tv.flatList[tv.cursor].node
}

// GetValue implements oat.ValueGetter. Returns SelectedNode().Value or nil.
func (tv *TreeView) GetValue() interface{} {
	n := tv.SelectedNode()
	if n == nil {
		return nil
	}
	return n.Value
}

// ApplyTheme stores the relevant theme styles for rendering.
func (tv *TreeView) ApplyTheme(t latte.Theme) {
	tv.Style = t.Text.Merge(tv.callerStyle)
	tv.FocusStyle = latte.Style{BorderFG: t.FocusBorder}
	tv.selectedStyle = t.ListSelected
	tv.normalStyle = t.Text
}

// Measure returns the preferred size for the TreeView.
func (tv *TreeView) Measure(c oat.Constraint) oat.Size {
	tv.rebuildIfDirty()

	h := len(tv.flatList)
	maxW := 0
	for _, e := range tv.flatList {
		// indent (2 per depth) + prefix (2) + label
		w := e.depth*2 + 2 + len([]rune(e.node.Label))
		if w > maxW {
			maxW = w
		}
	}
	if maxW == 0 {
		maxW = 10
	}

	return c.Clamp(oat.Size{Width: maxW, Height: h})
}

// Render draws the visible tree rows into buf within region.
func (tv *TreeView) Render(buf *oat.Buffer, region oat.Region) {
	tv.rebuildIfDirty()

	style := tv.EffectiveStyle(tv.IsFocused())
	sub := buf.Sub(region)
	tv.SetHitRegion(sub.Region())
	sub.FillBG(style)

	visibleRows := region.Height
	total := len(tv.flatList)

	// Clamp scroll so cursor is always visible.
	if tv.cursor < tv.scrollOff {
		tv.scrollOff = tv.cursor
	}
	if tv.cursor >= tv.scrollOff+visibleRows {
		tv.scrollOff = tv.cursor - visibleRows + 1
	}
	if tv.scrollOff < 0 {
		tv.scrollOff = 0
	}

	for row := 0; row < visibleRows; row++ {
		idx := tv.scrollOff + row
		if idx >= total {
			break
		}
		e := tv.flatList[idx]

		// Build indent string.
		indent := ""
		for i := 0; i < e.depth; i++ {
			indent += "  "
		}

		// Choose prefix based on whether node has children and expanded state.
		var prefix string
		if len(e.node.Children) > 0 {
			if e.node.IsExpanded() {
				prefix = "▼ "
			} else {
				prefix = "▶ "
			}
		} else {
			prefix = "  "
		}

		rowStyle := tv.normalStyle
		if tv.Style != (latte.Style{}) {
			rowStyle = tv.Style
		}
		if idx == tv.cursor && tv.IsFocused() {
			rowStyle = tv.selectedStyle
		}

		line := indent + prefix + e.node.Label

		// Fill row background then draw text.
		for x := 0; x < region.Width; x++ {
			sub.SetCell(x, row, ' ', rowStyle)
		}
		sub.DrawText(0, row, line, rowStyle)
	}
}

// HandleKey processes keyboard input when the TreeView is focused.
func (tv *TreeView) HandleKey(ev *oat.KeyEvent) bool {
	tv.rebuildIfDirty()
	total := len(tv.flatList)
	if total == 0 {
		return false
	}

	switch ev.Key() {
	case tcell.KeyUp:
		if tv.cursor > 0 {
			tv.cursor--
			tv.clampScroll()
		}
		return true

	case tcell.KeyDown:
		if tv.cursor < total-1 {
			tv.cursor++
			tv.clampScroll()
		}
		return true

	case tcell.KeyRight:
		e := tv.flatList[tv.cursor]
		if len(e.node.Children) > 0 && !e.node.IsExpanded() {
			e.node.Expand()
			tv.dirty = true
			tv.rebuildIfDirty()
		}
		return true

	case tcell.KeyLeft:
		e := tv.flatList[tv.cursor]
		if len(e.node.Children) > 0 && e.node.IsExpanded() {
			e.node.Collapse()
			tv.dirty = true
			tv.rebuildIfDirty()
			// Clamp cursor in case it fell outside the new flat list.
			if tv.cursor >= len(tv.flatList) {
				tv.cursor = len(tv.flatList) - 1
			}
		}
		return true

	case tcell.KeyEnter:
		e := tv.flatList[tv.cursor]
		if len(e.node.Children) > 0 {
			// Toggle expand/collapse for branch nodes.
			if e.node.IsExpanded() {
				e.node.Collapse()
			} else {
				e.node.Expand()
			}
			tv.dirty = true
			tv.rebuildIfDirty()
			if tv.cursor >= len(tv.flatList) {
				tv.cursor = len(tv.flatList) - 1
			}
		} else {
			// Leaf node: invoke callback.
			if tv.onSelect != nil {
				tv.onSelect(e.node)
			}
		}
		return true
	}

	return false
}

// KeyBindings advertises available shortcuts to the StatusBar.
func (tv *TreeView) KeyBindings() []oat.KeyBinding {
	return []oat.KeyBinding{
		{Key: tcell.KeyUp, Label: "↑", Description: "Up"},
		{Key: tcell.KeyDown, Label: "↓", Description: "Down"},
		{Key: tcell.KeyRight, Label: "→", Description: "Expand"},
		{Key: tcell.KeyLeft, Label: "←", Description: "Collapse"},
		{Key: tcell.KeyEnter, Label: "Enter", Description: "Select"},
	}
}

// Children satisfies oat.Layout. TreeView has no sub-components.
func (tv *TreeView) Children() []oat.Component { return nil }

// AddChild is a no-op for TreeView; use AddRoot to add tree nodes.
func (tv *TreeView) AddChild(_ oat.Component) {}

// rebuildIfDirty rebuilds the flatList from the current tree state.
func (tv *TreeView) rebuildIfDirty() {
	if !tv.dirty {
		return
	}
	tv.flatList = tv.flatList[:0]
	for _, root := range tv.roots {
		tv.flattenTree(root, 0)
	}
	tv.dirty = false
	// Clamp cursor to new list length.
	if len(tv.flatList) == 0 {
		tv.cursor = 0
	} else if tv.cursor >= len(tv.flatList) {
		tv.cursor = len(tv.flatList) - 1
	}
}

// flattenTree recursively appends visible entries to flatList.
func (tv *TreeView) flattenTree(node *TreeNode, depth int) {
	tv.flatList = append(tv.flatList, &flatEntry{node: node, depth: depth})
	if node.IsExpanded() {
		for _, child := range node.Children {
			tv.flattenTree(child, depth+1)
		}
	}
}

// clampScroll adjusts scrollOff so cursor remains visible.
func (tv *TreeView) clampScroll() {
	// Scroll adjustment is done lazily in Render; this is a no-op placeholder
	// kept so future callers have a well-named hook if needed.
}
