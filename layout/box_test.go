package layout_test

import (
	"testing"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/layout"
)

// fixedComponent returns a fixed Size from Measure, regardless of constraint.
// It intentionally does not clamp — it models a component that always wants
// exactly (w, h) and trusts its parent to respect the allocation.
type fixedComponent struct{ w, h int }

func (f *fixedComponent) Measure(_ oat.Constraint) oat.Size { return oat.Size{Width: f.w, Height: f.h} }
func (f *fixedComponent) Render(_ *oat.Buffer, _ oat.Region) {}

// constrainedComponent clamps its declared size to the given constraint.
// Use this when the test needs a child that properly respects limits.
type constrainedComponent struct{ w, h int }

func (c *constrainedComponent) Measure(con oat.Constraint) oat.Size {
	w, h := c.w, c.h
	if con.MaxWidth >= 0 && w > con.MaxWidth {
		w = con.MaxWidth
	}
	if con.MaxHeight >= 0 && h > con.MaxHeight {
		h = con.MaxHeight
	}
	return oat.Size{Width: w, Height: h}
}
func (c *constrainedComponent) Render(_ *oat.Buffer, _ oat.Region) {}

// --- VBox tests --------------------------------------------------------------

func TestVBoxMeasureStacksHeights(t *testing.T) {
	vbox := layout.NewVBox(
		&fixedComponent{w: 10, h: 3},
		&fixedComponent{w: 5, h: 2},
	)
	size := vbox.Measure(oat.Constraint{MaxWidth: 100, MaxHeight: 100})
	if size.Height != 5 {
		t.Errorf("VBox height: got %d, want 5 (3+2)", size.Height)
	}
}

func TestVBoxMeasureMaxWidth(t *testing.T) {
	vbox := layout.NewVBox(
		&fixedComponent{w: 10, h: 1},
		&fixedComponent{w: 5, h: 1},
	)
	size := vbox.Measure(oat.Constraint{MaxWidth: 100, MaxHeight: 100})
	if size.Width != 10 {
		t.Errorf("VBox width should be max child width: got %d, want 10", size.Width)
	}
}

func TestVBoxMeasureRespectsConstraint(t *testing.T) {
	vbox := layout.NewVBox(
		&fixedComponent{w: 50, h: 30},
	)
	size := vbox.Measure(oat.Constraint{MaxWidth: 20, MaxHeight: 10})
	if size.Width > 20 {
		t.Errorf("VBox width %d exceeds MaxWidth 20", size.Width)
	}
	if size.Height > 10 {
		t.Errorf("VBox height %d exceeds MaxHeight 10", size.Height)
	}
}

func TestVBoxMeasureEmpty(t *testing.T) {
	vbox := layout.NewVBox()
	size := vbox.Measure(oat.Constraint{MaxWidth: 100, MaxHeight: 100})
	if size.Width != 0 || size.Height != 0 {
		t.Errorf("empty VBox should measure {0 0}, got %+v", size)
	}
}

func TestVBoxMeasureWithGap(t *testing.T) {
	vbox := layout.NewVBox(
		&fixedComponent{w: 5, h: 2},
		&fixedComponent{w: 5, h: 2},
	).WithGap(1)
	size := vbox.Measure(oat.Constraint{MaxWidth: 100, MaxHeight: 100})
	// 2 + 1 (gap) + 2 = 5
	if size.Height != 5 {
		t.Errorf("VBox with gap=1: height = %d, want 5 (2+1+2)", size.Height)
	}
}

func TestVBoxMeasureSingleChild(t *testing.T) {
	vbox := layout.NewVBox(&fixedComponent{w: 7, h: 4})
	size := vbox.Measure(oat.Constraint{MaxWidth: 100, MaxHeight: 100})
	if size.Width != 7 || size.Height != 4 {
		t.Errorf("VBox single child: got %+v, want {7 4}", size)
	}
}

func TestVBoxChildrenReturnsAll(t *testing.T) {
	a := &fixedComponent{w: 1, h: 1}
	b := &fixedComponent{w: 2, h: 2}
	vbox := layout.NewVBox(a, b)
	children := vbox.Children()
	if len(children) != 2 {
		t.Errorf("VBox.Children() len = %d, want 2", len(children))
	}
}

func TestVBoxAddFlexChild(t *testing.T) {
	vbox := layout.NewVBox()
	vbox.AddFlexChild(&fixedComponent{w: 5, h: 1}, 2)
	children := vbox.Children()
	if len(children) != 1 {
		t.Errorf("VBox.AddFlexChild should add one child, got %d", len(children))
	}
}

// --- HBox tests --------------------------------------------------------------

func TestHBoxMeasureSumsWidths(t *testing.T) {
	hbox := layout.NewHBox(
		&fixedComponent{w: 5, h: 3},
		&fixedComponent{w: 7, h: 2},
	)
	size := hbox.Measure(oat.Constraint{MaxWidth: 100, MaxHeight: 100})
	if size.Width != 12 {
		t.Errorf("HBox width: got %d, want 12 (5+7)", size.Width)
	}
}

func TestHBoxMeasureMaxHeight(t *testing.T) {
	hbox := layout.NewHBox(
		&fixedComponent{w: 5, h: 3},
		&fixedComponent{w: 5, h: 7},
	)
	size := hbox.Measure(oat.Constraint{MaxWidth: 100, MaxHeight: 100})
	if size.Height != 7 {
		t.Errorf("HBox height should be max child height: got %d, want 7", size.Height)
	}
}

func TestHBoxMeasureRespectsConstraint(t *testing.T) {
	hbox := layout.NewHBox(
		&fixedComponent{w: 50, h: 30},
	)
	size := hbox.Measure(oat.Constraint{MaxWidth: 20, MaxHeight: 10})
	if size.Width > 20 {
		t.Errorf("HBox width %d exceeds MaxWidth 20", size.Width)
	}
	if size.Height > 10 {
		t.Errorf("HBox height %d exceeds MaxHeight 10", size.Height)
	}
}

func TestHBoxMeasureEmpty(t *testing.T) {
	hbox := layout.NewHBox()
	size := hbox.Measure(oat.Constraint{MaxWidth: 100, MaxHeight: 100})
	if size.Width != 0 || size.Height != 0 {
		t.Errorf("empty HBox should measure {0 0}, got %+v", size)
	}
}

func TestHBoxMeasureWithGap(t *testing.T) {
	hbox := layout.NewHBox(
		&fixedComponent{w: 5, h: 1},
		&fixedComponent{w: 5, h: 1},
	).WithGap(2)
	size := hbox.Measure(oat.Constraint{MaxWidth: 100, MaxHeight: 100})
	// 5 + 2 (gap) + 5 = 12
	if size.Width != 12 {
		t.Errorf("HBox with gap=2: width = %d, want 12 (5+2+5)", size.Width)
	}
}

func TestHBoxChildrenReturnsAll(t *testing.T) {
	a := &fixedComponent{w: 1, h: 1}
	b := &fixedComponent{w: 2, h: 2}
	hbox := layout.NewHBox(a, b)
	children := hbox.Children()
	if len(children) != 2 {
		t.Errorf("HBox.Children() len = %d, want 2", len(children))
	}
}

func TestHBoxAddFlexChild(t *testing.T) {
	hbox := layout.NewHBox()
	hbox.AddFlexChild(&fixedComponent{w: 1, h: 1}, 3)
	children := hbox.Children()
	if len(children) != 1 {
		t.Errorf("HBox.AddFlexChild should add one child, got %d", len(children))
	}
}

// --- VFill / HFill -----------------------------------------------------------

func TestVBoxWithVFillMeasuresZeroHeight(t *testing.T) {
	// VFill reports zero size during Measure; actual height is resolved at render time.
	vbox := layout.NewVBox(
		&fixedComponent{w: 5, h: 3},
		layout.NewVFill(),
	)
	size := vbox.Measure(oat.Constraint{MaxWidth: 100, MaxHeight: 100})
	// Only the fixed child (h=3) contributes; VFill adds 0 in Measure pass.
	if size.Height != 3 {
		t.Errorf("VBox with VFill Measure height: got %d, want 3", size.Height)
	}
}

func TestHBoxWithHFillMeasuresZeroWidth(t *testing.T) {
	hbox := layout.NewHBox(
		&fixedComponent{w: 5, h: 1},
		layout.NewHFill(),
	)
	size := hbox.Measure(oat.Constraint{MaxWidth: 100, MaxHeight: 100})
	// Only the fixed child (w=5) contributes; HFill adds 0 in Measure pass.
	if size.Width != 5 {
		t.Errorf("HBox with HFill Measure width: got %d, want 5", size.Width)
	}
}
