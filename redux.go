package oat

import "sync"

// Store is a Redux-inspired centralized state container.
//
// S is the state type (typically a struct).
// Actions are plain interface{} values dispatched via Store.Dispatch.
// The reducer function maps (currentState, action) → newState.
//
// Usage:
//
//	type AppState struct {
//	    Items   []string
//	    Loading bool
//	}
//
//	type AddItemAction struct{ Item string }
//	type SetLoadingAction struct{ Loading bool }
//
//	store := oat.NewStore(AppState{}, func(s AppState, action interface{}) AppState {
//	    switch a := action.(type) {
//	    case AddItemAction:
//	        return AppState{Items: append(s.Items, a.Item), Loading: s.Loading}
//	    case SetLoadingAction:
//	        return AppState{Items: s.Items, Loading: a.Loading}
//	    }
//	    return s
//	})
//
//	// Subscribe to state changes:
//	remove := store.Subscribe(func(s AppState) { /* update UI */ })
//	defer remove()
//
//	// Dispatch an action:
//	store.Dispatch(AddItemAction{Item: "hello"})
//
//	// Get current state:
//	state := store.GetState()
//
//	// Derived reactive value (auto-updates when store changes):
//	itemCount := oat.Select(store, func(s AppState) int { return len(s.Items) })
//	// itemCount is a *ValueNotifier[int] — wire to canvas.Redraw or Consumer
type Store[S any] struct {
	mu        sync.RWMutex
	state     S
	reducer   func(S, interface{}) S
	listeners []storeListener[S]
	nextID    int
}

type storeListener[S any] struct {
	id int
	fn func(S)
}

// NewStore creates a Store with the given initial state and reducer.
// The reducer must be a pure function: (state, action) → newState.
func NewStore[S any](initial S, reducer func(S, interface{}) S) *Store[S] {
	return &Store[S]{state: initial, reducer: reducer}
}

// Dispatch applies action to the current state via the reducer,
// updates the store's state, and notifies all subscribers.
// Thread-safe.
func (s *Store[S]) Dispatch(action interface{}) {
	s.mu.Lock()
	s.state = s.reducer(s.state, action)
	state := s.state
	// Snapshot listeners under lock to avoid holding lock during callbacks.
	fns := make([]func(S), len(s.listeners))
	for i, l := range s.listeners {
		fns[i] = l.fn
	}
	s.mu.Unlock()
	for _, fn := range fns {
		fn(state)
	}
}

// GetState returns a copy of the current state. Thread-safe.
func (s *Store[S]) GetState() S {
	s.mu.RLock()
	v := s.state
	s.mu.RUnlock()
	return v
}

// Subscribe registers fn to be called after every Dispatch.
// Returns a cancel function that removes the subscription.
// Thread-safe.
func (s *Store[S]) Subscribe(fn func(S)) (remove func()) {
	s.mu.Lock()
	id := s.nextID
	s.nextID++
	s.listeners = append(s.listeners, storeListener[S]{id: id, fn: fn})
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		for i, l := range s.listeners {
			if l.id == id {
				last := len(s.listeners) - 1
				s.listeners[i] = s.listeners[last]
				s.listeners[last] = storeListener[S]{}
				s.listeners = s.listeners[:last]
				return
			}
		}
	}
}

// Select creates a derived *ValueNotifier[T] that updates whenever the store's
// state changes and the selector returns a different value.
//
// T must be comparable — this constraint is required for the != equality check
// that guards unnecessary updates. If you need a non-comparable derived value,
// use Store.Subscribe directly and manage your own ValueNotifier.
//
// The notifier is seeded immediately with the current selected value.
// Use it with Consumer or ValueNotifier.AddListener to wire it to the canvas:
//
//	count := oat.Select(store, func(s AppState) int { return len(s.Items) })
//	count.AddListener(canvas.Redraw)
//
//	display := oat.NewConsumer(count, func(n int) oat.Component {
//	    return widget.NewText(fmt.Sprintf("%d items", n))
//	}, canvas.Redraw)
func Select[S, T comparable](store *Store[S], selector func(S) T) *ValueNotifier[T] {
	initial := selector(store.GetState())
	vn := NewValueNotifier(initial)
	store.Subscribe(func(s S) {
		next := selector(s)
		if next != vn.Value() {
			vn.Set(next)
		}
	})
	return vn
}

// Middleware is a function that wraps Dispatch to add cross-cutting behaviour
// (logging, async actions, batching, etc.).
//
// Usage:
//
//	logged := oat.ApplyMiddleware(store, func(next func(interface{})) func(interface{}) {
//	    return func(action interface{}) {
//	        log.Printf("dispatching: %T", action)
//	        next(action)
//	    }
//	})
//	logged(AddItemAction{Item: "hello"})
type Middleware func(next func(interface{})) func(interface{})

// ApplyMiddleware wraps store.Dispatch with the given middlewares, applied
// left-to-right (first middleware is outermost).
func ApplyMiddleware(store interface{ Dispatch(interface{}) }, middlewares ...Middleware) func(interface{}) {
	dispatch := store.Dispatch
	for i := len(middlewares) - 1; i >= 0; i-- {
		dispatch = middlewares[i](dispatch)
	}
	return dispatch
}
