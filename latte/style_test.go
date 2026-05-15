package latte

import "testing"

func TestStyleMergeFG(t *testing.T) {
	base := Style{FG: ColorRed}
	got := base.Merge(Style{FG: ColorBlue})
	if got.FG != ColorBlue {
		t.Errorf("Merge FG: got %v, want ColorBlue", got.FG)
	}
}

func TestStyleMergeDefaultFGPreservesBase(t *testing.T) {
	base := Style{FG: ColorRed}
	got := base.Merge(Style{FG: ColorDefault})
	if got.FG != ColorRed {
		t.Errorf("Merge with ColorDefault FG should preserve base, got %v", got.FG)
	}
}

func TestStyleMergeBoolFlags(t *testing.T) {
	base := Style{}
	got := base.Merge(Style{Bold: true, Italic: true, Underline: true, Blink: true, Reverse: true})
	if !got.Bold {
		t.Error("Merge should propagate Bold")
	}
	if !got.Italic {
		t.Error("Merge should propagate Italic")
	}
	if !got.Underline {
		t.Error("Merge should propagate Underline")
	}
	if !got.Blink {
		t.Error("Merge should propagate Blink")
	}
	if !got.Reverse {
		t.Error("Merge should propagate Reverse")
	}
}

func TestStyleMergeBorderOverrides(t *testing.T) {
	base := Style{Border: BorderDouble}
	got := base.Merge(Style{Border: BorderRounded})
	if got.Border != BorderRounded {
		t.Errorf("Merge Border: got %v, want BorderRounded", got.Border)
	}
}

func TestStyleMergeBorderNoneDoesNotOverride(t *testing.T) {
	base := Style{Border: BorderDouble}
	got := base.Merge(Style{Border: BorderNone})
	if got.Border != BorderDouble {
		t.Errorf("Merge BorderNone should not change existing border, got %v", got.Border)
	}
}

func TestStyleMergeBorderExplicitNoneClears(t *testing.T) {
	base := Style{Border: BorderDouble, BorderFG: ColorRed, BorderBG: ColorBlue}
	got := base.Merge(Style{Border: BorderExplicitNone})
	if got.Border != BorderNone {
		t.Errorf("Merge BorderExplicitNone should clear Border, got %v", got.Border)
	}
	if got.BorderFG != ColorDefault {
		t.Errorf("Merge BorderExplicitNone should clear BorderFG, got %v", got.BorderFG)
	}
	if got.BorderBG != ColorDefault {
		t.Errorf("Merge BorderExplicitNone should clear BorderBG, got %v", got.BorderBG)
	}
}

func TestStyleMergeTextAlign(t *testing.T) {
	base := Style{TextAlign: AlignStart}
	got := base.Merge(Style{TextAlign: AlignCenter})
	if got.TextAlign != AlignCenter {
		t.Errorf("Merge TextAlign: got %v, want AlignCenter", got.TextAlign)
	}
}

func TestStyleMergeTextAlignStartDoesNotOverride(t *testing.T) {
	base := Style{TextAlign: AlignEnd}
	got := base.Merge(Style{TextAlign: AlignStart})
	if got.TextAlign != AlignEnd {
		t.Errorf("Merge AlignStart should not override existing TextAlign, got %v", got.TextAlign)
	}
}

func TestStyleMergePadding(t *testing.T) {
	base := Style{Padding: Insets{1, 2, 3, 4}}
	other := Style{Padding: Insets{5, 5, 5, 5}}
	got := base.Merge(other)
	if got.Padding != (Insets{5, 5, 5, 5}) {
		t.Errorf("Merge Padding: got %+v, want {5 5 5 5}", got.Padding)
	}
}

func TestStyleMergeZeroPaddingDoesNotOverride(t *testing.T) {
	base := Style{Padding: Insets{2, 2, 2, 2}}
	got := base.Merge(Style{})
	if got.Padding != (Insets{2, 2, 2, 2}) {
		t.Errorf("Merge zero Padding should not override, got %+v", got.Padding)
	}
}

func TestStyleMergeMargin(t *testing.T) {
	base := Style{Margin: Insets{1, 1, 1, 1}}
	got := base.Merge(Style{Margin: Insets{3, 3, 3, 3}})
	if got.Margin != (Insets{3, 3, 3, 3}) {
		t.Errorf("Merge Margin: got %+v, want {3 3 3 3}", got.Margin)
	}
}

func TestStyleBuilderMethods(t *testing.T) {
	s := Style{}.
		WithFG(ColorRed).
		WithBG(ColorBlue).
		WithBold().
		WithItalic().
		WithUnderline().
		WithBlink().
		WithReverse().
		WithBorder(BorderSingle).
		WithBorderColor(ColorGreen).
		WithTextAlign(AlignCenter).
		WithPadding(2).
		WithMargin(1)

	if s.FG != ColorRed {
		t.Error("WithFG failed")
	}
	if s.BG != ColorBlue {
		t.Error("WithBG failed")
	}
	if !s.Bold {
		t.Error("WithBold failed")
	}
	if !s.Italic {
		t.Error("WithItalic failed")
	}
	if !s.Underline {
		t.Error("WithUnderline failed")
	}
	if !s.Blink {
		t.Error("WithBlink failed")
	}
	if !s.Reverse {
		t.Error("WithReverse failed")
	}
	if s.Border != BorderSingle {
		t.Error("WithBorder failed")
	}
	if s.BorderFG != ColorGreen {
		t.Error("WithBorderColor failed")
	}
	if s.TextAlign != AlignCenter {
		t.Error("WithTextAlign failed")
	}
	if s.Padding != (Insets{2, 2, 2, 2}) {
		t.Errorf("WithPadding(2) = %+v, want {2 2 2 2}", s.Padding)
	}
	if s.Margin != (Insets{1, 1, 1, 1}) {
		t.Errorf("WithMargin(1) = %+v, want {1 1 1 1}", s.Margin)
	}
}

func TestStyleBuilderWithPaddingInsets(t *testing.T) {
	insets := Insets{Top: 1, Right: 2, Bottom: 3, Left: 4}
	s := Style{}.WithPaddingInsets(insets)
	if s.Padding != insets {
		t.Errorf("WithPaddingInsets: got %+v, want %+v", s.Padding, insets)
	}
}

func TestStyleBuilderWithMarginInsets(t *testing.T) {
	insets := Insets{Top: 2, Right: 4, Bottom: 2, Left: 4}
	s := Style{}.WithMarginInsets(insets)
	if s.Margin != insets {
		t.Errorf("WithMarginInsets: got %+v, want %+v", s.Margin, insets)
	}
}

func TestBorderStyleRunes(t *testing.T) {
	tests := []struct {
		style       BorderStyle
		topLeft     rune
		topRight    rune
		bottomLeft  rune
		bottomRight rune
	}{
		{BorderSingle, '┌', '┐', '└', '┘'},
		{BorderDouble, '╔', '╗', '╚', '╝'},
		{BorderRounded, '╭', '╮', '╰', '╯'},
		{BorderThick, '┏', '┓', '┗', '┛'},
		{BorderDashed, '┌', '┐', '└', '┘'},
	}
	for _, tt := range tests {
		r := tt.style.Runes()
		if r.TopLeft != tt.topLeft {
			t.Errorf("BorderStyle(%v).Runes().TopLeft = %c, want %c", tt.style, r.TopLeft, tt.topLeft)
		}
		if r.TopRight != tt.topRight {
			t.Errorf("BorderStyle(%v).Runes().TopRight = %c, want %c", tt.style, r.TopRight, tt.topRight)
		}
		if r.BottomLeft != tt.bottomLeft {
			t.Errorf("BorderStyle(%v).Runes().BottomLeft = %c, want %c", tt.style, r.BottomLeft, tt.bottomLeft)
		}
		if r.BottomRight != tt.bottomRight {
			t.Errorf("BorderStyle(%v).Runes().BottomRight = %c, want %c", tt.style, r.BottomRight, tt.bottomRight)
		}
	}
}

func TestUniform(t *testing.T) {
	i := Uniform(3)
	if i.Top != 3 || i.Right != 3 || i.Bottom != 3 || i.Left != 3 {
		t.Errorf("Uniform(3) = %+v, want all 3", i)
	}
}

func TestSymmetric(t *testing.T) {
	i := Symmetric(2, 4)
	if i.Top != 2 || i.Bottom != 2 || i.Left != 4 || i.Right != 4 {
		t.Errorf("Symmetric(2, 4) = %+v, want {2 4 2 4}", i)
	}
}

func TestStyleMergeChaining(t *testing.T) {
	// Verify multiple Merge calls accumulate correctly.
	base := Style{FG: ColorRed, Bold: false}
	step1 := base.Merge(Style{Bold: true})
	step2 := step1.Merge(Style{FG: ColorBlue})
	if step2.FG != ColorBlue {
		t.Errorf("chained Merge FG: got %v, want ColorBlue", step2.FG)
	}
	if !step2.Bold {
		t.Error("chained Merge Bold: should still be true")
	}
}
