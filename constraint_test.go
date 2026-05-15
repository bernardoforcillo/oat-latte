package oat

import "testing"

func TestInsetsHorizontalVertical(t *testing.T) {
	i := Insets{Top: 1, Right: 2, Bottom: 3, Left: 4}
	if got := i.Horizontal(); got != 6 {
		t.Errorf("Horizontal() = %d, want 6", got)
	}
	if got := i.Vertical(); got != 4 {
		t.Errorf("Vertical() = %d, want 4", got)
	}
}

func TestUniform(t *testing.T) {
	i := Uniform(5)
	if i.Top != 5 || i.Right != 5 || i.Bottom != 5 || i.Left != 5 {
		t.Errorf("Uniform(5) = %+v, want all 5", i)
	}
}

func TestSymmetric(t *testing.T) {
	i := Symmetric(2, 4)
	if i.Top != 2 || i.Bottom != 2 || i.Left != 4 || i.Right != 4 {
		t.Errorf("Symmetric(2, 4) = %+v", i)
	}
}

func TestConstraintShrink(t *testing.T) {
	tests := []struct {
		name   string
		c      Constraint
		insets Insets
		want   Constraint
	}{
		{
			name:   "shrink by uniform",
			c:      Constraint{MaxWidth: 10, MaxHeight: 8},
			insets: Uniform(2),
			want:   Constraint{MaxWidth: 6, MaxHeight: 4},
		},
		{
			name:   "unconstrained width preserved",
			c:      Constraint{MaxWidth: -1, MaxHeight: 10},
			insets: Uniform(3),
			want:   Constraint{MaxWidth: -1, MaxHeight: 4},
		},
		{
			name:   "unconstrained height preserved",
			c:      Constraint{MaxWidth: 10, MaxHeight: -1},
			insets: Uniform(3),
			want:   Constraint{MaxWidth: 4, MaxHeight: -1},
		},
		{
			name:   "clamped to zero not negative",
			c:      Constraint{MaxWidth: 2, MaxHeight: 2},
			insets: Uniform(5),
			want:   Constraint{MaxWidth: 0, MaxHeight: 0},
		},
		{
			name:   "asymmetric insets",
			c:      Constraint{MaxWidth: 20, MaxHeight: 15},
			insets: Insets{Top: 1, Right: 2, Bottom: 3, Left: 4},
			want:   Constraint{MaxWidth: 14, MaxHeight: 11},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.c.Shrink(tt.insets)
			if got != tt.want {
				t.Errorf("Shrink() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestConstraintClamp(t *testing.T) {
	t.Run("clamps oversized", func(t *testing.T) {
		c := Constraint{MaxWidth: 10, MaxHeight: 5}
		got := c.Clamp(Size{Width: 20, Height: 10})
		if got.Width != 10 || got.Height != 5 {
			t.Errorf("Clamp() = %+v, want {10 5}", got)
		}
	})

	t.Run("within bounds unchanged", func(t *testing.T) {
		c := Constraint{MaxWidth: 10, MaxHeight: 5}
		got := c.Clamp(Size{Width: 5, Height: 3})
		if got.Width != 5 || got.Height != 3 {
			t.Errorf("Clamp() within bounds = %+v, want {5 3}", got)
		}
	})

	t.Run("unconstrained axis unchanged", func(t *testing.T) {
		c := Constraint{MaxWidth: -1, MaxHeight: -1}
		got := c.Clamp(Size{Width: 999, Height: 888})
		if got.Width != 999 || got.Height != 888 {
			t.Errorf("Clamp() unconstrained = %+v, want {999 888}", got)
		}
	})

	t.Run("only width constrained", func(t *testing.T) {
		c := Constraint{MaxWidth: 10, MaxHeight: -1}
		got := c.Clamp(Size{Width: 20, Height: 999})
		if got.Width != 10 || got.Height != 999 {
			t.Errorf("Clamp() one axis = %+v, want {10 999}", got)
		}
	})
}

func TestRegionInner(t *testing.T) {
	t.Run("uniform inset", func(t *testing.T) {
		r := Region{X: 5, Y: 5, Width: 20, Height: 10}
		inner := r.Inner(Uniform(1))
		if inner.X != 6 || inner.Y != 6 || inner.Width != 18 || inner.Height != 8 {
			t.Errorf("Inner(1) = %+v, want {6 6 18 8}", inner)
		}
	})

	t.Run("clamped to zero when inset exceeds size", func(t *testing.T) {
		r := Region{X: 0, Y: 0, Width: 2, Height: 2}
		inner := r.Inner(Uniform(5))
		if inner.Width != 0 || inner.Height != 0 {
			t.Errorf("Inner too large = %+v, want width=0 height=0", inner)
		}
	})

	t.Run("asymmetric insets shift origin correctly", func(t *testing.T) {
		r := Region{X: 0, Y: 0, Width: 20, Height: 10}
		inner := r.Inner(Insets{Top: 2, Right: 3, Bottom: 1, Left: 4})
		if inner.X != 4 || inner.Y != 2 || inner.Width != 13 || inner.Height != 7 {
			t.Errorf("Inner asymmetric = %+v, want {4 2 13 7}", inner)
		}
	})
}

func TestRegionToConstraint(t *testing.T) {
	r := Region{X: 3, Y: 4, Width: 15, Height: 10}
	c := r.ToConstraint()
	if c.MaxWidth != 15 || c.MaxHeight != 10 {
		t.Errorf("ToConstraint() = %+v, want {15 10}", c)
	}
}

func TestDimensions(t *testing.T) {
	if _, ok := Fixed(10).(FixedDim); !ok {
		t.Error("Fixed() should return FixedDim")
	}
	if Fixed(10).(FixedDim).N != 10 {
		t.Error("Fixed(10).N should be 10")
	}
	if _, ok := Fill.(FillDim); !ok {
		t.Error("Fill should be FillDim")
	}
	if _, ok := Auto.(AutoDim); !ok {
		t.Error("Auto should be AutoDim")
	}
}
