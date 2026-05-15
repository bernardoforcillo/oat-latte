package oat

import (
	"testing"

	"github.com/gdamore/tcell/v2"
)

// --- minimal mock components -------------------------------------------------

type mockFocusable struct {
	FocusBehavior
	handled bool
}

func (m *mockFocusable) Measure(_ Constraint) Size  { return Size{} }
func (m *mockFocusable) Render(_ *Buffer, _ Region) {}
func (m *mockFocusable) HandleKey(_ *KeyEvent) bool { m.handled = true; return true }

type mockLayout struct {
	children []Component
}

func (m *mockLayout) Measure(_ Constraint) Size      { return Size{} }
func (m *mockLayout) Render(_ *Buffer, _ Region)     {}
func (m *mockLayout) Children() []Component          { return m.children }
func (m *mockLayout) AddChild(c Component)           { m.children = append(m.children, c) }

type guardedFocusable struct {
	FocusBehavior
	allow bool
}

func (g *guardedFocusable) Measure(_ Constraint) Size  { return Size{} }
func (g *guardedFocusable) Render(_ *Buffer, _ Region) {}
func (g *guardedFocusable) HandleKey(_ *KeyEvent) bool { return false }
func (g *guardedFocusable) IsFocusable() bool          { return g.allow }

// --- tests -------------------------------------------------------------------

func TestFocusManagerEmpty(t *testing.T) {
	fm := NewFocusManager()
	if fm.Current() != nil {
		t.Error("empty FocusManager should have nil Current()")
	}
	if len(fm.Nodes()) != 0 {
		t.Error("empty FocusManager should have no nodes")
	}
	// Must not panic on empty
	fm.Next()
	fm.Prev()
}

func TestFocusManagerCollect(t *testing.T) {
	a := &mockFocusable{}
	b := &mockFocusable{}
	root := &mockLayout{children: []Component{a, b}}

	fm := NewFocusManager()
	fm.Collect(root)

	if len(fm.Nodes()) != 2 {
		t.Fatalf("Collect: got %d nodes, want 2", len(fm.Nodes()))
	}
	if !a.IsFocused() {
		t.Error("first focusable should be focused after Collect")
	}
	if b.IsFocused() {
		t.Error("second focusable should not be focused after Collect")
	}
}

func TestFocusManagerCollectNested(t *testing.T) {
	leaf1 := &mockFocusable{}
	leaf2 := &mockFocusable{}
	inner := &mockLayout{children: []Component{leaf1, leaf2}}
	root := &mockLayout{children: []Component{inner}}

	fm := NewFocusManager()
	fm.Collect(root)

	if len(fm.Nodes()) != 2 {
		t.Errorf("nested Collect: got %d nodes, want 2", len(fm.Nodes()))
	}
}

func TestFocusManagerNext(t *testing.T) {
	a := &mockFocusable{}
	b := &mockFocusable{}
	root := &mockLayout{children: []Component{a, b}}

	fm := NewFocusManager()
	fm.Collect(root)

	fm.Next()
	if !b.IsFocused() {
		t.Error("after Next(), second should be focused")
	}
	if a.IsFocused() {
		t.Error("after Next(), first should not be focused")
	}
}

func TestFocusManagerNextWraps(t *testing.T) {
	a := &mockFocusable{}
	b := &mockFocusable{}
	root := &mockLayout{children: []Component{a, b}}

	fm := NewFocusManager()
	fm.Collect(root)

	fm.Next() // b
	fm.Next() // wraps back to a
	if !a.IsFocused() {
		t.Error("Next() should wrap around to first node")
	}
	if b.IsFocused() {
		t.Error("after wrap, second should not be focused")
	}
}

func TestFocusManagerPrevWraps(t *testing.T) {
	a := &mockFocusable{}
	b := &mockFocusable{}
	root := &mockLayout{children: []Component{a, b}}

	fm := NewFocusManager()
	fm.Collect(root)

	fm.Prev() // wraps from a to b
	if !b.IsFocused() {
		t.Error("Prev() from first should wrap to last")
	}
	if a.IsFocused() {
		t.Error("after Prev() wrap, first should not be focused")
	}
}

func TestFocusManagerFocusIndex(t *testing.T) {
	a := &mockFocusable{}
	b := &mockFocusable{}
	c := &mockFocusable{}
	root := &mockLayout{children: []Component{a, b, c}}

	fm := NewFocusManager()
	fm.Collect(root)

	fm.FocusIndex(2)
	if !c.IsFocused() {
		t.Error("FocusIndex(2) should focus the third node")
	}
	if a.IsFocused() || b.IsFocused() {
		t.Error("FocusIndex should defocus all other nodes")
	}
}

func TestFocusManagerFocusIndexOutOfRange(t *testing.T) {
	a := &mockFocusable{}
	root := &mockLayout{children: []Component{a}}

	fm := NewFocusManager()
	fm.Collect(root)

	// Out of range index should be a no-op, not a panic
	fm.FocusIndex(-1)
	fm.FocusIndex(99)
	if !a.IsFocused() {
		t.Error("out-of-range FocusIndex should leave current focus unchanged")
	}
}

func TestFocusManagerFocusByRef(t *testing.T) {
	a := &mockFocusable{}
	b := &mockFocusable{}
	c := &mockFocusable{}
	root := &mockLayout{children: []Component{a, b, c}}

	fm := NewFocusManager()
	fm.Collect(root)

	fm.FocusByRef(c)
	if !c.IsFocused() {
		t.Error("FocusByRef should focus the referenced component")
	}
	if a.IsFocused() || b.IsFocused() {
		t.Error("FocusByRef should defocus all other nodes")
	}
}

func TestFocusManagerFocusByRefMissing(t *testing.T) {
	a := &mockFocusable{}
	outsider := &mockFocusable{}
	root := &mockLayout{children: []Component{a}}

	fm := NewFocusManager()
	fm.Collect(root)

	// FocusByRef on a node not in the tree should be a no-op
	fm.FocusByRef(outsider)
	if !a.IsFocused() {
		t.Error("FocusByRef on unknown node should not change focus")
	}
}

func TestFocusManagerDispatch(t *testing.T) {
	a := &mockFocusable{}
	root := &mockLayout{children: []Component{a}}

	fm := NewFocusManager()
	fm.Collect(root)

	ev := tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone)
	consumed := fm.Dispatch(ev)
	if !consumed {
		t.Error("Dispatch should return true when HandleKey consumes the event")
	}
	if !a.handled {
		t.Error("HandleKey should have been called on the focused component")
	}
}

func TestFocusManagerDispatchEmpty(t *testing.T) {
	fm := NewFocusManager()
	ev := tcell.NewEventKey(tcell.KeyRune, 'x', tcell.ModNone)
	if fm.Dispatch(ev) {
		t.Error("Dispatch on empty FocusManager should return false")
	}
}

func TestFocusManagerCollectDefocusesPrevious(t *testing.T) {
	a := &mockFocusable{}
	root := &mockLayout{children: []Component{a}}

	fm := NewFocusManager()
	fm.Collect(root)

	if !a.IsFocused() {
		t.Fatal("expected a to be focused after first Collect")
	}

	empty := &mockLayout{}
	fm.Collect(empty)

	if a.IsFocused() {
		t.Error("Collect on new tree should defocus the previously focused component")
	}
}

func TestFocusManagerFocusGuardSkipsNode(t *testing.T) {
	blocked := &guardedFocusable{allow: false}
	allowed := &mockFocusable{}
	root := &mockLayout{children: []Component{blocked, allowed}}

	fm := NewFocusManager()
	fm.Collect(root)

	if len(fm.Nodes()) != 1 {
		t.Errorf("FocusGuard=false should skip node; got %d nodes, want 1", len(fm.Nodes()))
	}
	if !allowed.IsFocused() {
		t.Error("allowed node should be focused")
	}
}

func TestFocusManagerCurrentKeyBindings(t *testing.T) {
	a := &mockFocusable{}
	root := &mockLayout{children: []Component{a}}

	fm := NewFocusManager()
	fm.Collect(root)

	// mockFocusable doesn't implement Keybinder — should return nil, not panic
	bindings := fm.CurrentKeyBindings()
	if bindings != nil {
		t.Error("CurrentKeyBindings on non-Keybinder should return nil")
	}
}

func TestMatchesBinding(t *testing.T) {
	binding := KeyBinding{Key: tcell.KeyCtrlS, Label: "^S", Description: "Save"}
	ev := tcell.NewEventKey(tcell.KeyCtrlS, 0, tcell.ModNone)
	if !matchesBinding(ev, binding) {
		t.Error("matchesBinding should match KeyCtrlS event against KeyCtrlS binding")
	}

	evWrong := tcell.NewEventKey(tcell.KeyCtrlQ, 0, tcell.ModNone)
	if matchesBinding(evWrong, binding) {
		t.Error("matchesBinding should not match different key")
	}
}

func TestMatchesBindingRune(t *testing.T) {
	binding := BindRune('q', tcell.ModNone, "q", "Quit", nil)
	ev := tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)
	if !matchesBinding(ev, binding) {
		t.Error("matchesBinding should match rune 'q'")
	}

	evWrong := tcell.NewEventKey(tcell.KeyRune, 'w', tcell.ModNone)
	if matchesBinding(evWrong, binding) {
		t.Error("matchesBinding should not match different rune")
	}
}
