package oat

import "testing"

// mockFixed is a non-focusable Component with a fixed size.
// It does not clamp to the constraint — use constrainedMock when that matters.
type mockFixed struct{ w, h int }

func (m *mockFixed) Measure(_ Constraint) Size  { return Size{Width: m.w, Height: m.h} }
func (m *mockFixed) Render(_ *Buffer, _ Region) {}

// ── State[T] tests ────────────────────────────────────────────────────────────

func TestStateGet(t *testing.T) {
	s := NewState(42, nil)
	if s.Get() != 42 {
		t.Errorf("Get() = %d, want 42", s.Get())
	}
}

func TestStateSetState(t *testing.T) {
	s := NewState(0, nil)
	s.SetState(func(v *int) { *v = 10 })
	if s.Get() != 10 {
		t.Errorf("after SetState, Get() = %d, want 10", s.Get())
	}
}

func TestStateIsDirtyAfterInit(t *testing.T) {
	s := NewState(0, nil)
	if !s.isDirty() {
		t.Error("newly created State should be dirty")
	}
}

func TestStateGetAndClearCleansDirty(t *testing.T) {
	s := NewState(5, nil)
	v := s.getAndClear()
	if v != 5 {
		t.Errorf("getAndClear() = %d, want 5", v)
	}
	if s.isDirty() {
		t.Error("after getAndClear, State should not be dirty")
	}
}

func TestStateSetStateMakesDirty(t *testing.T) {
	s := NewState(0, nil)
	s.getAndClear() // consume initial dirty
	if s.isDirty() {
		t.Fatal("expected clean after getAndClear")
	}
	s.SetState(func(v *int) { *v++ })
	if !s.isDirty() {
		t.Error("State should be dirty after SetState")
	}
}

func TestStateRedrawFnCalledOnSetState(t *testing.T) {
	called := 0
	s := NewState(0, func() { called++ })
	s.SetState(func(v *int) { *v++ })
	if called != 1 {
		t.Errorf("redrawFn called %d times, want 1", called)
	}
}

func TestStateRedrawFnNilSafe(t *testing.T) {
	s := NewState(0, nil)
	s.SetState(func(v *int) { *v++ }) // must not panic
}

func TestStateMultipleSetState(t *testing.T) {
	s := NewState(0, nil)
	for i := 0; i < 5; i++ {
		s.SetState(func(v *int) { *v++ })
	}
	if s.Get() != 5 {
		t.Errorf("after 5 increments, Get() = %d, want 5", s.Get())
	}
}

// ── StatelessWidget tests ─────────────────────────────────────────────────────

func TestStatelessWidgetMeasure(t *testing.T) {
	w := NewStatelessWidget(func() Component {
		return &mockFixed{w: 10, h: 5}
	})
	size := w.Measure(Constraint{MaxWidth: 100, MaxHeight: 100})
	if size.Width != 10 || size.Height != 5 {
		t.Errorf("StatelessWidget Measure: got %+v, want {10 5}", size)
	}
}

func TestStatelessWidgetChildren(t *testing.T) {
	inner := &mockFixed{w: 5, h: 3}
	w := NewStatelessWidget(func() Component { return inner })
	children := w.Children()
	if len(children) != 1 {
		t.Fatalf("Children() len = %d, want 1", len(children))
	}
	if children[0] != inner {
		t.Error("Children[0] should be the built inner component")
	}
}

func TestStatelessWidgetCachesFrame(t *testing.T) {
	buildCount := 0
	w := NewStatelessWidget(func() Component {
		buildCount++
		return &mockFixed{w: 1, h: 1}
	})
	w.Measure(Constraint{MaxWidth: 100, MaxHeight: 100})
	w.Render(nil, Region{Width: 1, Height: 1}) // uses cached frame — no rebuild
	if buildCount != 1 {
		t.Errorf("buildCount = %d, want 1 (frame should be cached)", buildCount)
	}
}

func TestStatelessWidgetInvalidateTriggersRebuild(t *testing.T) {
	buildCount := 0
	w := NewStatelessWidget(func() Component {
		buildCount++
		return &mockFixed{w: 1, h: 1}
	})
	w.Measure(Constraint{MaxWidth: 100, MaxHeight: 100})
	w.Invalidate()
	w.Measure(Constraint{MaxWidth: 100, MaxHeight: 100})
	if buildCount != 2 {
		t.Errorf("after Invalidate, buildCount = %d, want 2", buildCount)
	}
}

func TestStatelessWidgetHasID(t *testing.T) {
	a := NewStatelessWidget(func() Component { return &mockFixed{} })
	b := NewStatelessWidget(func() Component { return &mockFixed{} })
	if a.ID == "" {
		t.Error("StatelessWidget should have a non-empty ID")
	}
	if a.ID == b.ID {
		t.Error("different StatelessWidgets should have different IDs")
	}
}

func TestStatelessWidgetWithID(t *testing.T) {
	w := NewStatelessWidget(func() Component { return &mockFixed{} }).WithID("sw-1")
	if w.ID != "sw-1" {
		t.Errorf("WithID: got %q, want %q", w.ID, "sw-1")
	}
}

// ── StatefulWidget[T] tests ───────────────────────────────────────────────────

func TestStatefulWidgetBuildsFromInitialState(t *testing.T) {
	s := NewState(7, nil)
	w := NewStatefulWidget(s, func(v int) Component {
		return &mockFixed{w: v, h: 1}
	})
	size := w.Measure(Constraint{MaxWidth: 100, MaxHeight: 100})
	if size.Width != 7 {
		t.Errorf("StatefulWidget Measure width: got %d, want 7", size.Width)
	}
}

func TestStatefulWidgetReusesFrameWhenClean(t *testing.T) {
	s := NewState(1, nil)
	buildCount := 0
	w := NewStatefulWidget(s, func(v int) Component {
		buildCount++
		return &mockFixed{w: v, h: 1}
	})

	w.Measure(Constraint{MaxWidth: 100, MaxHeight: 100})
	countAfterFirst := buildCount

	// Second Measure with no SetState — frame should be reused.
	w.Measure(Constraint{MaxWidth: 100, MaxHeight: 100})
	if buildCount != countAfterFirst {
		t.Errorf("second Measure without SetState triggered rebuild (buildCount %d→%d)", countAfterFirst, buildCount)
	}
}

func TestStatefulWidgetRebuildsAfterSetState(t *testing.T) {
	s := NewState(3, nil)
	w := NewStatefulWidget(s, func(v int) Component {
		return &mockFixed{w: v, h: 1}
	})

	w.Measure(Constraint{MaxWidth: 100, MaxHeight: 100}) // consume initial dirty

	s.SetState(func(v *int) { *v = 20 })
	size := w.Measure(Constraint{MaxWidth: 100, MaxHeight: 100})
	if size.Width != 20 {
		t.Errorf("after SetState(20), width = %d, want 20", size.Width)
	}
}

func TestStatefulWidgetChildrenExposesFrame(t *testing.T) {
	s := NewState(0, nil)
	w := NewStatefulWidget(s, func(_ int) Component {
		return &mockFixed{w: 1, h: 1}
	})
	children := w.Children()
	if len(children) != 1 {
		t.Fatalf("Children() len = %d, want 1", len(children))
	}
}

func TestStatefulWidgetHasID(t *testing.T) {
	s := NewState(0, nil)
	a := NewStatefulWidget(s, func(_ int) Component { return &mockFixed{} })
	b := NewStatefulWidget(s, func(_ int) Component { return &mockFixed{} })
	if a.ID == "" {
		t.Error("StatefulWidget should have a non-empty ID")
	}
	if a.ID == b.ID {
		t.Error("different StatefulWidgets should have different IDs")
	}
}

func TestStatefulWidgetWithID(t *testing.T) {
	s := NewState(0, nil)
	w := NewStatefulWidget(s, func(_ int) Component { return &mockFixed{} }).WithID("sf-1")
	if w.ID != "sf-1" {
		t.Errorf("WithID: got %q, want %q", w.ID, "sf-1")
	}
}

// ── Canvas.Redraw tests ───────────────────────────────────────────────────────

func TestCanvasRedrawDoesNotBlock(t *testing.T) {
	cv := NewCanvas()
	// Multiple calls should coalesce without blocking.
	cv.Redraw()
	cv.Redraw()
	cv.Redraw()
}

func TestCanvasRedrawSendsToChannel(t *testing.T) {
	cv := NewCanvas()
	cv.Redraw()
	select {
	case <-cv.redrawCh:
		// received — correct
	default:
		t.Error("Redraw() should have sent to redrawCh")
	}
}
