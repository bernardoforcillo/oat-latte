package oat

import (
	"sync"

	"github.com/antoniocali/oat-latte/latte"
)

// ── StatelessWidget ───────────────────────────────────────────────────────────

// StatelessWidget wraps a pure build function as a Component.
// The frame is built once and cached; call Invalidate to rebuild on the next
// Measure pass. Use this when your UI is entirely determined by the values
// captured in the closure (no external mutable state).
//
// For components that react to mutable state, use StatefulWidget or Consumer.
//
// Example:
//
//	header := oat.NewStatelessWidget(func() oat.Component {
//	    return widget.NewText("Hello, world!").WithStyle(latte.Header)
//	})
type StatelessWidget struct {
	BaseComponent
	buildFn func() Component
	frame   Component
	theme   *latte.Theme
}

// NewStatelessWidget creates a StatelessWidget from the given build function.
func NewStatelessWidget(build func() Component) *StatelessWidget {
	w := &StatelessWidget{buildFn: build}
	w.EnsureID()
	return w
}

// WithID sets a user-defined identifier on this widget.
func (w *StatelessWidget) WithID(id string) *StatelessWidget { w.ID = id; return w }

// Invalidate clears the cached frame so it is rebuilt on the next Measure pass.
func (w *StatelessWidget) Invalidate() { w.frame = nil }

// ApplyTheme stores the theme and propagates it to the current frame if any.
func (w *StatelessWidget) ApplyTheme(t latte.Theme) {
	w.theme = &t
	if w.frame != nil {
		applyThemeTree(w.frame, t)
	}
}

func (w *StatelessWidget) getOrBuild() Component {
	if w.frame == nil {
		w.frame = w.buildFn()
		if w.theme != nil {
			applyThemeTree(w.frame, *w.theme)
		}
	}
	return w.frame
}

// Measure delegates to the cached (or newly built) frame.
func (w *StatelessWidget) Measure(c Constraint) Size {
	return w.getOrBuild().Measure(c)
}

// Render delegates to the cached frame.
func (w *StatelessWidget) Render(buf *Buffer, region Region) {
	w.getOrBuild().Render(buf, region)
}

// Children exposes the frame as a single child so the focus tree and theme
// propagation can descend transparently into the built component.
func (w *StatelessWidget) Children() []Component {
	return []Component{w.getOrBuild()}
}

// AddChild is a no-op; StatelessWidget children are managed by the build function.
func (w *StatelessWidget) AddChild(_ Component) {}

// ── State[T] ─────────────────────────────────────────────────────────────────

// State is a generic mutable state container for use with StatefulWidget.
//
// SetState mutates the value, marks the associated widget as needing a rebuild,
// and optionally calls a redraw function to wake the Canvas event loop.
//
// State is safe to use from multiple goroutines.
//
// Example:
//
//	type Counter struct{ n int }
//
//	// Key-event-driven mutations don't need a redrawFn; the Canvas re-renders
//	// automatically after every key event. Pass canvas.Redraw for background
//	// goroutine mutations.
//	state := oat.NewState(Counter{}, nil)
//
//	btn := widget.NewButton("+1", func() {
//	    state.SetState(func(s *Counter) { s.n++ })
//	})
type State[T any] struct {
	mu       sync.Mutex
	value    T
	dirty    bool
	redrawFn func()
}

// NewState creates a State with the given initial value.
// redrawFn is called after each SetState call; pass canvas.Redraw when state
// may be mutated from outside the Canvas event loop (e.g., goroutines, timers).
// Pass nil if mutations are always triggered by UI events.
func NewState[T any](initial T, redrawFn func()) *State[T] {
	return &State[T]{value: initial, dirty: true, redrawFn: redrawFn}
}

// Get returns a copy of the current state value. Thread-safe.
func (s *State[T]) Get() T {
	s.mu.Lock()
	v := s.value
	s.mu.Unlock()
	return v
}

// SetState applies fn to the current value, marks the widget as needing a
// rebuild, and calls redrawFn (if set) to trigger a screen refresh.
// Thread-safe.
func (s *State[T]) SetState(fn func(*T)) {
	s.mu.Lock()
	fn(&s.value)
	s.dirty = true
	redraw := s.redrawFn
	s.mu.Unlock()
	if redraw != nil {
		redraw()
	}
}

func (s *State[T]) isDirty() bool {
	s.mu.Lock()
	d := s.dirty
	s.mu.Unlock()
	return d
}

// getAndClear returns the current value and clears the dirty flag atomically.
func (s *State[T]) getAndClear() T {
	s.mu.Lock()
	v := s.value
	s.dirty = false
	s.mu.Unlock()
	return v
}

// ── StatefulWidget[T] ─────────────────────────────────────────────────────────

// StatefulWidget pairs a State[T] with a build function.
//
// The frame is rebuilt whenever State.SetState is called (dirty flag set).
// Between state changes the cached frame is reused, making repeated renders cheap.
//
// For the focus system to pick up any structural changes (new or removed focusable
// widgets), call canvas.InvalidateLayout() after a SetState that alters the tree
// structure. Changes triggered by canvas.Redraw automatically call InvalidateLayout.
//
// Example:
//
//	type Counter struct{ n int }
//
//	state := oat.NewState(Counter{}, canvas.Redraw)
//
//	counter := oat.NewStatefulWidget(state, func(s Counter) oat.Component {
//	    return layout.NewVBox(
//	        widget.NewText(fmt.Sprintf("Count: %d", s.n)),
//	        widget.NewButton("+1", func() {
//	            state.SetState(func(s *Counter) { s.n++ })
//	        }),
//	    )
//	})
type StatefulWidget[T any] struct {
	BaseComponent
	state   *State[T]
	buildFn func(T) Component
	frame   Component
	theme   *latte.Theme
}

// NewStatefulWidget creates a StatefulWidget backed by the given State.
func NewStatefulWidget[T any](state *State[T], build func(T) Component) *StatefulWidget[T] {
	w := &StatefulWidget[T]{state: state, buildFn: build}
	w.EnsureID()
	return w
}

// WithID sets a user-defined identifier on this widget.
func (w *StatefulWidget[T]) WithID(id string) *StatefulWidget[T] { w.ID = id; return w }

// ApplyTheme stores the theme, builds an initial frame (so theme propagates to
// children during Canvas.Run's initial theme pass), and applies the theme.
func (w *StatefulWidget[T]) ApplyTheme(t latte.Theme) {
	w.theme = &t
	w.frame = w.buildFn(w.state.Get())
	applyThemeTree(w.frame, t)
}

func (w *StatefulWidget[T]) ensureFrame() Component {
	if w.frame == nil || w.state.isDirty() {
		val := w.state.getAndClear()
		w.frame = w.buildFn(val)
		if w.theme != nil {
			applyThemeTree(w.frame, *w.theme)
		}
	}
	return w.frame
}

// Measure rebuilds the frame if the state is dirty, then delegates.
func (w *StatefulWidget[T]) Measure(c Constraint) Size {
	return w.ensureFrame().Measure(c)
}

// Render delegates to the current frame (rebuilt by Measure if dirty).
func (w *StatefulWidget[T]) Render(buf *Buffer, region Region) {
	w.ensureFrame().Render(buf, region)
}

// Children exposes the frame as a single child for transparent tree traversal.
func (w *StatefulWidget[T]) Children() []Component {
	return []Component{w.ensureFrame()}
}

// AddChild is a no-op; StatefulWidget children are managed by the build function.
func (w *StatefulWidget[T]) AddChild(_ Component) {}
