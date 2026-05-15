package oat

import "testing"

// ── ChangeNotifier tests ──────────────────────────────────────────────────────

func TestChangeNotifierNotifiesListener(t *testing.T) {
	var n ChangeNotifier
	called := 0
	n.AddListener(func() { called++ })
	n.NotifyListeners()
	if called != 1 {
		t.Errorf("NotifyListeners: listener called %d times, want 1", called)
	}
}

func TestChangeNotifierMultipleListeners(t *testing.T) {
	var n ChangeNotifier
	a, b := 0, 0
	n.AddListener(func() { a++ })
	n.AddListener(func() { b++ })
	n.NotifyListeners()
	if a != 1 || b != 1 {
		t.Errorf("both listeners should be called once; a=%d b=%d", a, b)
	}
}

func TestChangeNotifierRemoveListener(t *testing.T) {
	var n ChangeNotifier
	called := 0
	remove := n.AddListener(func() { called++ })
	remove()
	n.NotifyListeners()
	if called != 0 {
		t.Errorf("after remove, listener should not be called; called %d times", called)
	}
}

func TestChangeNotifierRemoveOneOfMany(t *testing.T) {
	var n ChangeNotifier
	a, b := 0, 0
	removeA := n.AddListener(func() { a++ })
	n.AddListener(func() { b++ })
	removeA()
	n.NotifyListeners()
	if a != 0 {
		t.Errorf("removed listener a called %d times, want 0", a)
	}
	if b != 1 {
		t.Errorf("remaining listener b called %d times, want 1", b)
	}
}

func TestChangeNotifierRemoveIdempotent(t *testing.T) {
	var n ChangeNotifier
	called := 0
	remove := n.AddListener(func() { called++ })
	remove()
	remove() // calling twice must not panic
	n.NotifyListeners()
	if called != 0 {
		t.Errorf("after double remove, listener called %d times, want 0", called)
	}
}

func TestChangeNotifierNoListeners(t *testing.T) {
	var n ChangeNotifier
	n.NotifyListeners() // must not panic
}

// ── ValueNotifier[T] tests ────────────────────────────────────────────────────

func TestValueNotifierInitialValue(t *testing.T) {
	vn := NewValueNotifier(42)
	if vn.Value() != 42 {
		t.Errorf("Value() = %d, want 42", vn.Value())
	}
}

func TestValueNotifierSet(t *testing.T) {
	vn := NewValueNotifier(0)
	vn.Set(99)
	if vn.Value() != 99 {
		t.Errorf("after Set(99), Value() = %d, want 99", vn.Value())
	}
}

func TestValueNotifierNotifiesOnSet(t *testing.T) {
	vn := NewValueNotifier(0)
	called := 0
	vn.AddListener(func() { called++ })
	vn.Set(1)
	if called != 1 {
		t.Errorf("listener called %d times after Set, want 1", called)
	}
}

func TestValueNotifierStringType(t *testing.T) {
	vn := NewValueNotifier("hello")
	vn.Set("world")
	if vn.Value() != "world" {
		t.Errorf("string ValueNotifier: Value() = %q, want %q", vn.Value(), "world")
	}
}

func TestValueNotifierListenerReceivesUpdates(t *testing.T) {
	vn := NewValueNotifier(0)
	snapshots := []int{}
	vn.AddListener(func() { snapshots = append(snapshots, vn.Value()) })
	vn.Set(1)
	vn.Set(2)
	vn.Set(3)
	if len(snapshots) != 3 || snapshots[0] != 1 || snapshots[1] != 2 || snapshots[2] != 3 {
		t.Errorf("snapshots = %v, want [1 2 3]", snapshots)
	}
}

func TestValueNotifierRemoveListener(t *testing.T) {
	vn := NewValueNotifier(0)
	called := 0
	remove := vn.AddListener(func() { called++ })
	vn.Set(1) // fires
	remove()
	vn.Set(2) // should NOT fire
	if called != 1 {
		t.Errorf("after remove, listener called %d times total, want 1", called)
	}
}

// ── Consumer[T] tests ─────────────────────────────────────────────────────────

func TestConsumerMeasureFromNotifier(t *testing.T) {
	vn := NewValueNotifier(7)
	c := NewConsumer(vn, func(v int) Component {
		return &mockFixed{w: v, h: 2}
	}, nil)
	size := c.Measure(Constraint{MaxWidth: 100, MaxHeight: 100})
	if size.Width != 7 || size.Height != 2 {
		t.Errorf("Consumer Measure: got %+v, want {7 2}", size)
	}
}

func TestConsumerReflectsUpdatedValue(t *testing.T) {
	vn := NewValueNotifier(5)
	c := NewConsumer(vn, func(v int) Component {
		return &mockFixed{w: v, h: 1}
	}, nil)

	vn.Set(15)
	size := c.Measure(Constraint{MaxWidth: 100, MaxHeight: 100})
	if size.Width != 15 {
		t.Errorf("Consumer after Set(15): width = %d, want 15", size.Width)
	}
}

func TestConsumerChildrenExposesFrame(t *testing.T) {
	vn := NewValueNotifier(0)
	c := NewConsumer(vn, func(_ int) Component {
		return &mockFixed{w: 1, h: 1}
	}, nil)
	children := c.Children()
	if len(children) != 1 {
		t.Fatalf("Consumer.Children() len = %d, want 1", len(children))
	}
}

func TestConsumerWiresRedrawListener(t *testing.T) {
	vn := NewValueNotifier(0)
	redraws := 0
	_ = NewConsumer(vn, func(_ int) Component {
		return &mockFixed{}
	}, func() { redraws++ })

	vn.Set(1)
	if redraws != 1 {
		t.Errorf("Consumer redrawFn called %d times on Set, want 1", redraws)
	}
}

func TestConsumerDisposeRemovesListener(t *testing.T) {
	vn := NewValueNotifier(0)
	redraws := 0
	c := NewConsumer(vn, func(_ int) Component {
		return &mockFixed{}
	}, func() { redraws++ })

	vn.Set(1) // fires
	c.Dispose()
	vn.Set(2) // should NOT fire after dispose
	if redraws != 1 {
		t.Errorf("after Dispose, redrawFn fired %d times total, want 1", redraws)
	}
}

func TestConsumerHasID(t *testing.T) {
	vn := NewValueNotifier(0)
	a := NewConsumer(vn, func(_ int) Component { return &mockFixed{} }, nil)
	b := NewConsumer(vn, func(_ int) Component { return &mockFixed{} }, nil)
	if a.ID == "" {
		t.Error("Consumer should have a non-empty ID")
	}
	if a.ID == b.ID {
		t.Error("different Consumers should have different IDs")
	}
}

// ── Integration: StatefulWidget with FocusManager ────────────────────────────

func TestStatefulWidgetFocusWalk(t *testing.T) {
	s := NewState(0, nil)

	focusable1 := &mockFocusable{}
	focusable2 := &mockFocusable{}

	sw := NewStatefulWidget(s, func(_ int) Component {
		return &mockLayout{children: []Component{focusable1, focusable2}}
	})

	root := &mockLayout{children: []Component{sw}}
	fm := NewFocusManager()
	fm.Collect(root)

	if len(fm.Nodes()) != 2 {
		t.Errorf("FocusManager should find 2 focusable nodes through StatefulWidget; got %d", len(fm.Nodes()))
	}
}

func TestStatelessWidgetFocusWalk(t *testing.T) {
	focusable := &mockFocusable{}
	sw := NewStatelessWidget(func() Component {
		return &mockLayout{children: []Component{focusable}}
	})

	root := &mockLayout{children: []Component{sw}}
	fm := NewFocusManager()
	fm.Collect(root)

	if len(fm.Nodes()) != 1 {
		t.Errorf("FocusManager should find 1 focusable node through StatelessWidget; got %d", len(fm.Nodes()))
	}
}
