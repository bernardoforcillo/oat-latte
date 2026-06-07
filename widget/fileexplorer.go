package widget

import (
	"os"
	"path/filepath"
	"strings"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// FileExplorer is a file-system browser widget built on top of TreeView.
// It supports lazy loading of directory contents, hidden-file filtering,
// glob-based file filtering, and a selection callback.
//
// Directory nodes are loaded lazily: the first time the user expands a
// directory, its children are read from disk. A sentinel child node
// (Label="...", Value=nil) marks unexpanded directories so that the
// TreeView shows an expand arrow without reading the full tree upfront.
type FileExplorer struct {
	oat.BaseComponent
	oat.FocusBehavior
	oat.BaseHitRegion

	tree       *TreeView
	root       string // absolute root path
	showHidden bool
	glob       string // if non-empty, filter files by this pattern (e.g. "*.go")
	onSelect   func(path string)

	callerStyle latte.Style
}

// NewFileExplorer creates a FileExplorer rooted at dir.
// The root directory is immediately loaded one level deep; further expansion
// is lazy — children are loaded on first expand.
func NewFileExplorer(dir string) *FileExplorer {
	fe := &FileExplorer{showHidden: false}
	fe.EnsureID()

	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = filepath.Clean(dir)
	}
	fe.root = abs

	fe.tree = NewTreeView()
	fe.tree.WithOnSelect(func(node *TreeNode) {
		if path, ok := node.Value.(string); ok && fe.onSelect != nil {
			fe.onSelect(path)
		}
	})

	fe.reload()
	return fe
}

// WithID sets a user-defined identifier on this component.
func (fe *FileExplorer) WithID(id string) *FileExplorer {
	fe.ID = id
	return fe
}

// WithShowHidden controls whether hidden entries (names beginning with ".")
// are shown. Default is false (hidden entries are not shown).
func (fe *FileExplorer) WithShowHidden(show bool) *FileExplorer {
	fe.showHidden = show
	fe.reload()
	return fe
}

// WithGlob sets a glob pattern used to filter file entries (e.g. "*.go").
// Directory entries are never filtered — they are always included so the tree
// can be navigated. Passing an empty string disables filtering.
func (fe *FileExplorer) WithGlob(pattern string) *FileExplorer {
	fe.glob = pattern
	fe.reload()
	return fe
}

// WithOnSelect registers a callback that is invoked when the user presses
// Enter on a leaf (file) node. The absolute path of the selected file is
// passed to fn.
func (fe *FileExplorer) WithOnSelect(fn func(path string)) *FileExplorer {
	fe.onSelect = fn
	return fe
}

// SelectedPath returns the absolute path of the currently highlighted entry,
// or an empty string if the tree is empty or the highlighted node has no path.
func (fe *FileExplorer) SelectedPath() string {
	node := fe.tree.SelectedNode()
	if node == nil {
		return ""
	}
	path, _ := node.Value.(string)
	return path
}

// ApplyTheme applies theme tokens to the FileExplorer and its inner TreeView.
func (fe *FileExplorer) ApplyTheme(t latte.Theme) {
	fe.Style = t.Text.Merge(fe.callerStyle)
	fe.FocusStyle = latte.Style{BorderFG: t.FocusBorder}
	fe.tree.ApplyTheme(t)
}

// Measure returns the preferred size by delegating to the inner TreeView.
func (fe *FileExplorer) Measure(c oat.Constraint) oat.Size {
	return fe.tree.Measure(c)
}

// Render draws the FileExplorer by delegating to the inner TreeView.
func (fe *FileExplorer) Render(buf *oat.Buffer, region oat.Region) {
	sub := buf.Sub(region)
	fe.SetHitRegion(sub.Region())
	// Propagate focus state to the inner tree so it highlights the cursor row.
	fe.tree.SetFocused(fe.IsFocused())
	fe.tree.Render(buf, region)
}

// HandleKey processes keyboard input. KeyRight and KeyEnter trigger lazy
// loading of directory children before the event is passed to the TreeView.
func (fe *FileExplorer) HandleKey(ev *oat.KeyEvent) bool {
	switch ev.Key() {
	case tcell.KeyRight, tcell.KeyEnter:
		node := fe.tree.SelectedNode()
		if node != nil && !node.IsExpanded() && node.Value != nil {
			path, ok := node.Value.(string)
			if ok {
				info, err := os.Stat(path)
				if err == nil && info.IsDir() {
					fe.expandDir(node)
				}
			}
		}
	}
	return fe.tree.HandleKey(ev)
}

// KeyBindings advertises available shortcuts (forwarded from the inner TreeView).
func (fe *FileExplorer) KeyBindings() []oat.KeyBinding {
	return fe.tree.KeyBindings()
}

// Children satisfies oat.Layout. Returns the inner TreeView as the sole child.
func (fe *FileExplorer) Children() []oat.Component { return []oat.Component{fe.tree} }

// AddChild is a no-op; the FileExplorer manages its own internal tree.
func (fe *FileExplorer) AddChild(_ oat.Component) {}

// reload clears all tree roots and repopulates them from fe.root.
func (fe *FileExplorer) reload() {
	entries, err := os.ReadDir(fe.root)
	if err != nil {
		fe.tree.SetRoots(nil)
		return
	}

	var roots []*TreeNode
	for _, entry := range entries {
		if !fe.showHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		path := filepath.Join(fe.root, entry.Name())

		if entry.IsDir() {
			node := NewTreeNode(entry.Name(), path)
			// Add sentinel child so TreeView renders the expand arrow.
			node.AddChild(NewTreeNode("...", nil))
			roots = append(roots, node)
		} else {
			// Apply glob filter to files (not directories).
			if fe.glob != "" {
				matched, err := filepath.Match(fe.glob, entry.Name())
				if err != nil || !matched {
					continue
				}
			}
			roots = append(roots, NewTreeNode(entry.Name(), path))
		}
	}
	fe.tree.SetRoots(roots)
}

// expandDir loads real children into node, replacing the sentinel child.
// It is called just before the TreeView processes an expand key event,
// so the newly loaded children are immediately visible after expansion.
func (fe *FileExplorer) expandDir(node *TreeNode) {
	dirPath, ok := node.Value.(string)
	if !ok {
		return
	}

	// Only lazy-load when the single sentinel child is present.
	if len(node.Children) != 1 || node.Children[0].Value != nil {
		return
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		node.Children = nil
		node.AddChild(NewTreeNode("(error reading dir)", nil))
		return
	}

	node.Children = nil // clear sentinel
	for _, entry := range entries {
		if !fe.showHidden && strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		childPath := filepath.Join(dirPath, entry.Name())
		if entry.IsDir() {
			child := NewTreeNode(entry.Name(), childPath)
			child.AddChild(NewTreeNode("...", nil)) // sentinel for nested dir
			node.AddChild(child)
		} else {
			if fe.glob != "" {
				matched, _ := filepath.Match(fe.glob, entry.Name())
				if !matched {
					continue
				}
			}
			node.AddChild(NewTreeNode(entry.Name(), childPath))
		}
	}

	if len(node.Children) == 0 {
		node.AddChild(NewTreeNode("(empty)", nil))
	}

	// Mark the tree dirty so its flat list is rebuilt before rendering.
	fe.tree.dirty = true
}
