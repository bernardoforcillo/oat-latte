package oat

import (
	"sync"

	"github.com/antoniocali/oat-latte/latte"
)

// listenerEntry pairs a unique numeric ID with a listener function.
type listenerEntry struct {
	id int
	fn func()
}

// ── ChangeNotifier ────────────────────────────────────────────────────────────

// ChangeNotifier provides thread-safe listener management.
// Embed it in a custom state struct and call NotifyListeners after mutations.
// This is the oat-latte equivalent of Flutter's ChangeNotifier mixin.
//
// Example:
//
//	type AppState struct {
//	    oat.ChangeNotifier
//	    items []string
//	}
//
//	func (s *AppState) AddItem(item string) {
//	    s.items = append(s.items, item)
//	    s.NotifyListeners()
//	}
//
//	// Wire to canvas so mutations trigger re-renders:
//	remove := state.AddListener(canvas.Redraw)
//	defer remove()
type ChangeNotifier struct {
	mu        sync.Mutex
	listeners []listenerEntry
	nextID    int
}

// AddListener registers fn and returns a cancel function that removes it.
// Safe to call from any goroutine.
func (n *ChangeNotifier) AddListener(fn func()) (remove func()) {
	n.mu.Lock()
	id := n.nextID
	n.nextID++
	n.listeners = append(n.listeners, listenerEntry{id, fn})
	n.mu.Unlock()

	return func() {
		n.mu.Lock()
		defer n.mu.Unlock()
		for i, e := range n.listeners {
			if e.id == id {
				last := len(n.listeners) - 1
				n.listeners[i] = n.listeners[last]
				n.listeners[last] = listenerEntry{}
				n.listeners = n.listeners[:last]
				return
			}
		}
	}
}

// NotifyListeners calls every registered listener outside the lock to prevent
// deadlocks when listeners themselves call AddListener or NotifyListeners.
// Safe to call from any goroutine.
func (n *ChangeNotifier) NotifyListeners() {
	n.mu.Lock()
	fns := make([]func(), len(n.listeners))
	for i, e := range n.listeners {
		fns[i] = e.fn
	}
	n.mu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

// ── ValueNotifier[T] ─────────────────────────────────────────────────────────

// ValueNotifier holds a single value and notifies listeners when it changes.
// This is the oat-latte equivalent of Flutter's ValueNotifier<T>.
//
// Usage:
//
//	count := oat.NewValueNotifier(0)
//	count.AddListener(canvas.Redraw) // re-render whenever value changes
//
//	// From any goroutine or event handler:
//	count.Set(count.Value() + 1)
type ValueNotifier[T any] struct {
	ChangeNotifier
	mu    sync.RWMutex
	value T
}

// NewValueNotifier creates a ValueNotifier with the given initial value.
func NewValueNotifier[T any](initial T) *ValueNotifier[T] {
	return &ValueNotifier[T]{value: initial}
}

// Value returns the current value. Thread-safe.
func (vn *ValueNotifier[T]) Value() T {
	vn.mu.RLock()
	v := vn.value
	vn.mu.RUnlock()
	return v
}

// Set stores val and notifies all listeners. Thread-safe.
func (vn *ValueNotifier[T]) Set(val T) {
	vn.mu.Lock()
	vn.value = val
	vn.mu.Unlock()
	vn.NotifyListeners()
}

// ── Consumer[T] ──────────────────────────────────────────────────────────────

// Consumer builds its component tree from a ValueNotifier value.
// It rebuilds on every Measure pass, always reflecting the latest notifier value.
//
// Pass redrawFn (typically canvas.Redraw) so external Set calls wake the event
// loop. If the notifier is only mutated inside UI event handlers you can pass nil
// — the Canvas re-renders automatically after every key event.
//
// Example:
//
//	count := oat.NewValueNotifier(0)
//	count.AddListener(canvas.Redraw) // or pass canvas.Redraw to NewConsumer
//
//	display := oat.NewConsumer(count, func(n int) oat.Component {
//	    return widget.NewText(fmt.Sprintf("Count: %d", n))
//	}, canvas.Redraw)
//
//	btn := widget.NewButton("+1", func() { count.Set(count.Value() + 1) })
type Consumer[T any] struct {
	BaseComponent
	notifier       *ValueNotifier[T]
	buildFn        func(T) Component
	frame          Component
	theme          *latte.Theme
	removeListener func()
}

// NewConsumer creates a Consumer that rebuilds whenever vn changes.
// redrawFn is wired as a listener on vn; pass nil if not needed.
func NewConsumer[T any](vn *ValueNotifier[T], build func(T) Component, redrawFn func()) *Consumer[T] {
	c := &Consumer[T]{notifier: vn, buildFn: build}
	c.EnsureID()
	if redrawFn != nil {
		c.removeListener = vn.AddListener(redrawFn)
	}
	return c
}

// WithID sets a user-defined identifier on this widget.
func (c *Consumer[T]) WithID(id string) *Consumer[T] { c.ID = id; return c }

// Dispose removes the Consumer's listener from the notifier.
// Call when the Consumer is permanently removed from the component tree.
func (c *Consumer[T]) Dispose() {
	if c.removeListener != nil {
		c.removeListener()
		c.removeListener = nil
	}
}

// ApplyTheme stores the theme and builds an initial frame so the theme
// propagates to children during Canvas.Run's initial theme walk.
func (c *Consumer[T]) ApplyTheme(t latte.Theme) {
	c.theme = &t
	c.frame = c.buildFn(c.notifier.Value())
	applyThemeTree(c.frame, t)
}

// Measure always rebuilds from the latest notifier value, then delegates.
func (c *Consumer[T]) Measure(con Constraint) Size {
	c.frame = c.buildFn(c.notifier.Value())
	if c.theme != nil {
		applyThemeTree(c.frame, *c.theme)
	}
	return c.frame.Measure(con)
}

// Render delegates to the frame built during the most recent Measure pass.
func (c *Consumer[T]) Render(buf *Buffer, region Region) {
	if c.frame == nil {
		c.frame = c.buildFn(c.notifier.Value())
		if c.theme != nil {
			applyThemeTree(c.frame, *c.theme)
		}
	}
	c.frame.Render(buf, region)
}

// Children exposes the frame as a single child for transparent tree traversal.
func (c *Consumer[T]) Children() []Component {
	if c.frame == nil {
		c.frame = c.buildFn(c.notifier.Value())
	}
	return []Component{c.frame}
}

// AddChild is a no-op; Consumer children are managed by the build function.
func (c *Consumer[T]) AddChild(_ Component) {}
