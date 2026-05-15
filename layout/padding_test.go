package layout_test

import (
	"testing"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/layout"
)

func TestPaddingMeasureAddsInsets(t *testing.T) {
	child := &fixedComponent{w: 10, h: 5}
	p := layout.NewPaddingUniform(child, 2)
	size := p.Measure(oat.Constraint{MaxWidth: 100, MaxHeight: 100})
	// 10 + 2*2 = 14 wide, 5 + 2*2 = 9 tall
	if size.Width != 14 {
		t.Errorf("PaddingUniform(2) width: got %d, want 14", size.Width)
	}
	if size.Height != 9 {
		t.Errorf("PaddingUniform(2) height: got %d, want 9", size.Height)
	}
}

func TestPaddingMeasureAsymmetric(t *testing.T) {
	child := &fixedComponent{w: 10, h: 5}
	insets := oat.Insets{Top: 1, Right: 2, Bottom: 3, Left: 4}
	p := layout.NewPadding(child, insets)
	size := p.Measure(oat.Constraint{MaxWidth: 100, MaxHeight: 100})
	// 10 + 4 + 2 = 16 wide, 5 + 1 + 3 = 9 tall
	if size.Width != 16 {
		t.Errorf("Padding asymmetric width: got %d, want 16", size.Width)
	}
	if size.Height != 9 {
		t.Errorf("Padding asymmetric height: got %d, want 9", size.Height)
	}
}

func TestPaddingMeasureNilChild(t *testing.T) {
	p := layout.NewPaddingUniform(nil, 3)
	size := p.Measure(oat.Constraint{MaxWidth: 100, MaxHeight: 100})
	// nil child → only inset dimensions: 3+3=6
	if size.Width != 6 {
		t.Errorf("Padding nil child width: got %d, want 6", size.Width)
	}
	if size.Height != 6 {
		t.Errorf("Padding nil child height: got %d, want 6", size.Height)
	}
}

func TestPaddingMeasurePassesShrunkConstraintToChild(t *testing.T) {
	// Use a constraint-respecting child to verify Padding shrinks the constraint
	// by its insets before delegating, and then adds them back to the result.
	child := &constrainedComponent{w: 50, h: 50}
	p := layout.NewPaddingUniform(child, 2)
	// MaxWidth=20 → inner MaxWidth=16; child sees 16 → returns 16; +2+2 = 20
	size := p.Measure(oat.Constraint{MaxWidth: 20, MaxHeight: 15})
	if size.Width != 20 {
		t.Errorf("Padding width: got %d, want 20", size.Width)
	}
	if size.Height != 15 {
		t.Errorf("Padding height: got %d, want 15", size.Height)
	}
}

func TestPaddingChildrenReturnsChild(t *testing.T) {
	child := &fixedComponent{w: 5, h: 5}
	p := layout.NewPaddingUniform(child, 1)
	children := p.Children()
	if len(children) != 1 {
		t.Errorf("Padding.Children() len = %d, want 1", len(children))
	}
	if children[0] != child {
		t.Error("Padding.Children()[0] should be the wrapped child")
	}
}

func TestPaddingChildrenNilChild(t *testing.T) {
	p := layout.NewPaddingUniform(nil, 1)
	if children := p.Children(); len(children) != 0 {
		t.Errorf("Padding with nil child should have 0 children, got %d", len(children))
	}
}
