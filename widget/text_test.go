package widget_test

import (
	"testing"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/widget"
)

// --- Text tests --------------------------------------------------------------

func TestTextMeasureSingleLine(t *testing.T) {
	txt := widget.NewText("hello")
	size := txt.Measure(oat.Constraint{MaxWidth: 100, MaxHeight: -1})
	if size.Width != 5 {
		t.Errorf("Text single line width: got %d, want 5", size.Width)
	}
	if size.Height != 1 {
		t.Errorf("Text single line height: got %d, want 1", size.Height)
	}
}

func TestTextMeasureMultiLine(t *testing.T) {
	txt := widget.NewText("line1\nline2\nline3")
	size := txt.Measure(oat.Constraint{MaxWidth: 100, MaxHeight: -1})
	if size.Height != 3 {
		t.Errorf("Text multi-line height: got %d, want 3", size.Height)
	}
}

func TestTextMeasureMaxWidth(t *testing.T) {
	txt := widget.NewText("a very long line that exceeds the constraint")
	size := txt.Measure(oat.Constraint{MaxWidth: 10, MaxHeight: -1})
	if size.Width > 10 {
		t.Errorf("Text width %d exceeds MaxWidth 10", size.Width)
	}
}

func TestTextMeasureMaxHeight(t *testing.T) {
	txt := widget.NewText("line1\nline2\nline3\nline4\nline5")
	size := txt.Measure(oat.Constraint{MaxWidth: 100, MaxHeight: 3})
	if size.Height > 3 {
		t.Errorf("Text height %d exceeds MaxHeight 3", size.Height)
	}
}

func TestTextMeasureEmpty(t *testing.T) {
	txt := widget.NewText("")
	size := txt.Measure(oat.Constraint{MaxWidth: 100, MaxHeight: -1})
	if size.Width != 0 {
		t.Errorf("empty Text width: got %d, want 0", size.Width)
	}
	if size.Height != 1 {
		// An empty string splits into [""] — 1 line.
		t.Errorf("empty Text height: got %d, want 1", size.Height)
	}
}

func TestTextGetValue(t *testing.T) {
	txt := widget.NewText("hello world")
	v := txt.GetValue()
	s, ok := v.(string)
	if !ok {
		t.Fatalf("GetValue() should return string, got %T", v)
	}
	if s != "hello world" {
		t.Errorf("GetValue() = %q, want %q", s, "hello world")
	}
}

func TestTextSetText(t *testing.T) {
	txt := widget.NewText("original")
	txt.SetText("updated")
	if txt.GetText() != "updated" {
		t.Errorf("SetText/GetText: got %q, want %q", txt.GetText(), "updated")
	}
}

func TestTextWordWrapMeasure(t *testing.T) {
	txt := widget.NewText("word1 word2 word3").WithWordWrap(true)
	// With MaxWidth=7 each word is 5 chars; "word1" fits alone (5≤7), then
	// "word2" won't fit on same line as "word1 word2" (11>7), so it wraps.
	size := txt.Measure(oat.Constraint{MaxWidth: 7, MaxHeight: -1})
	if size.Height < 2 {
		t.Errorf("word-wrapped text should have at least 2 lines, got height=%d", size.Height)
	}
}

// --- Title tests -------------------------------------------------------------

func TestTitleMeasureWithSeparator(t *testing.T) {
	title := widget.NewTitle("Hello World")
	size := title.Measure(oat.Constraint{MaxWidth: 100, MaxHeight: -1})
	if size.Width != 11 {
		t.Errorf("Title width: got %d, want 11", size.Width)
	}
	if size.Height != 2 {
		t.Errorf("Title with separator height: got %d, want 2", size.Height)
	}
}

func TestTitleMeasureWithoutSeparator(t *testing.T) {
	title := widget.NewTitle("Hi").WithSeparator(false)
	size := title.Measure(oat.Constraint{MaxWidth: 100, MaxHeight: -1})
	if size.Width != 2 {
		t.Errorf("Title no-separator width: got %d, want 2", size.Width)
	}
	if size.Height != 1 {
		t.Errorf("Title no-separator height: got %d, want 1", size.Height)
	}
}

func TestTitleMeasureRespectsConstraint(t *testing.T) {
	title := widget.NewTitle("A very long title text that exceeds the limit")
	size := title.Measure(oat.Constraint{MaxWidth: 10, MaxHeight: 1})
	if size.Width > 10 {
		t.Errorf("Title width %d exceeds MaxWidth 10", size.Width)
	}
	if size.Height > 1 {
		t.Errorf("Title height %d exceeds MaxHeight 1", size.Height)
	}
}

func TestTitleGetValue(t *testing.T) {
	title := widget.NewTitle("My Title")
	v := title.GetValue()
	s, ok := v.(string)
	if !ok {
		t.Fatalf("Title.GetValue() should return string, got %T", v)
	}
	if s != "My Title" {
		t.Errorf("Title.GetValue() = %q, want %q", s, "My Title")
	}
}

func TestTextHasID(t *testing.T) {
	a := widget.NewText("foo")
	b := widget.NewText("bar")
	if a.ID == "" {
		t.Error("NewText should assign a non-empty ID")
	}
	if a.ID == b.ID {
		t.Error("different Text widgets should have different IDs")
	}
}

func TestTextWithIDSetsCustomID(t *testing.T) {
	txt := widget.NewText("content").WithID("my-text")
	if txt.ID != "my-text" {
		t.Errorf("WithID: got %q, want %q", txt.ID, "my-text")
	}
}
