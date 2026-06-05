// Package oattest provides lightweight testing utilities for oat-latte components.
//
// Because oat.Buffer is unexported, this package cannot call component.Render
// directly. Instead it provides:
//
//   - MockComponent: a simple oat.Component implementation for layout testing.
//   - NewKeyEvent: constructs *tcell.EventKey values (= *oat.KeyEvent) for
//     simulating key presses in unit tests.
//   - Assertion helpers: MustContain / MustNotContain operate on []string rows
//     (e.g. produced by capturing terminal output via a higher-level harness).
//   - ContainsString: the pure predicate underlying the assertion helpers.
//
// Rendering tests that need to inspect actual pixel output should use
// tcell.SimulationScreen together with an oat.Canvas or a thin wrapper
// exported from the application under test.
package oattest

import (
	"strings"
	"time"

	oat "github.com/antoniocali/oat-latte"
	"github.com/gdamore/tcell/v2"
)

// MockComponent is a minimal oat.Component whose desired size is controlled by
// its exported W and H fields.  Render marks the component as having been
// rendered by setting Rendered = true.
//
// Use MockComponent to test layout logic without depending on real widgets:
//
//	a := &oattest.MockComponent{W: 20, H: 5}
//	b := &oattest.MockComponent{W: 20, H: 5}
//	box := layout.NewHBox(a, b)
//	size := box.Measure(oat.Constraint{MaxWidth: 80, MaxHeight: 24})
type MockComponent struct {
	// W is the desired width reported by Measure.
	W int
	// H is the desired height reported by Measure.
	H int
	// Rendered is set to true the first time Render is called.
	Rendered bool

	base oat.BaseComponent
}

// Measure returns the desired size clamped to c.
func (m *MockComponent) Measure(c oat.Constraint) oat.Size {
	s := oat.Size{Width: m.W, Height: m.H}
	return c.Clamp(s)
}

// Render marks the component as rendered (sets Rendered = true).
// It does not write any cells because oat.Buffer is not accessible from
// external packages.
func (m *MockComponent) Render(_ *oat.Buffer, _ oat.Region) {
	m.Rendered = true
}

// GetID satisfies oat.IDer.  The ID is auto-generated on first call.
func (m *MockComponent) GetID() string {
	m.base.EnsureID()
	return m.base.ID
}

// Children satisfies oat.Layout (MockComponent is a leaf node).
func (m *MockComponent) Children() []oat.Component { return nil }

// AddChild is a no-op.
func (m *MockComponent) AddChild(_ oat.Component) {}

// ─── Key-event builder ───────────────────────────────────────────────────────

// NewKeyEvent constructs a *tcell.EventKey (which is the same type as
// *oat.KeyEvent) for use in HandleKey tests.
//
//	ev := oattest.NewKeyEvent(tcell.KeyEnter, 0, tcell.ModNone)
//	consumed := myWidget.HandleKey(ev)
func NewKeyEvent(key tcell.Key, r rune, mod tcell.ModMask) *tcell.EventKey {
	return tcell.NewEventKey(key, r, mod)
}

// NewRuneEvent is a convenience wrapper that builds a key event for a printable
// rune (sets key = tcell.KeyRune automatically).
//
//	ev := oattest.NewRuneEvent('a', tcell.ModNone)
func NewRuneEvent(r rune, mod tcell.ModMask) *tcell.EventKey {
	return NewKeyEvent(tcell.KeyRune, r, mod)
}

// ─── Assertion helpers ───────────────────────────────────────────────────────

// ContainsString reports whether any element of rows contains sub as a
// substring.  It is the pure predicate used by MustContain / MustNotContain.
func ContainsString(rows []string, sub string) bool {
	for _, row := range rows {
		if strings.Contains(row, sub) {
			return true
		}
	}
	return false
}

// MustContain asserts that at least one row in rows contains substring.
// It calls t.Fatalf if the substring is not found.
//
//	oattest.MustContain(t, rows, "Hello")
func MustContain(t interface {
	Fatalf(format string, args ...interface{})
}, rows []string, substring string) {
	if !ContainsString(rows, substring) {
		t.Fatalf("oattest.MustContain: %q not found in rendered output:\n%s",
			substring, strings.Join(rows, "\n"))
	}
}

// MustNotContain asserts that no row in rows contains substring.
// It calls t.Fatalf if the substring is found.
//
//	oattest.MustNotContain(t, rows, "Error")
func MustNotContain(t interface {
	Fatalf(format string, args ...interface{})
}, rows []string, substring string) {
	if ContainsString(rows, substring) {
		t.Fatalf("oattest.MustNotContain: %q unexpectedly found in rendered output:\n%s",
			substring, strings.Join(rows, "\n"))
	}
}

// ─── Constraint / Size helpers ───────────────────────────────────────────────

// FixedConstraint returns a Constraint with both axes fixed to the given size.
func FixedConstraint(width, height int) oat.Constraint {
	return oat.Constraint{MaxWidth: width, MaxHeight: height}
}

// MeasureComponent is a convenience wrapper that calls c.Measure with a fixed
// constraint of the given dimensions.
//
//	size := oattest.MeasureComponent(myWidget, 80, 24)
func MeasureComponent(c oat.Component, width, height int) oat.Size {
	return c.Measure(FixedConstraint(width, height))
}

// ─── SimScreen ───────────────────────────────────────────────────────────────

// SimScreen wraps tcell.SimulationScreen to provide a lightweight, headless
// terminal surface for tests that drive a full oat.Canvas.
//
// Because oat.Buffer is unexported, SimScreen cannot call component.Render
// directly.  Instead, build a real oat.Canvas with a SimulationScreen backend.
// The recommended pattern is:
//
//	screen := oattest.NewSimScreen(80, 24)
//	// Use screen.Screen() to pass the underlying tcell.SimulationScreen to
//	// any oat or tcell API that accepts a tcell.Screen.
type SimScreen struct {
	screen tcell.SimulationScreen
	width  int
	height int
}

// NewSimScreen creates and initialises a headless simulation screen of the
// given size.
func NewSimScreen(width, height int) *SimScreen {
	s := tcell.NewSimulationScreen("")
	_ = s.Init()
	s.SetSize(width, height)
	return &SimScreen{screen: s, width: width, height: height}
}

// Screen returns the underlying tcell.SimulationScreen.
// Pass this to any API that accepts a tcell.Screen.
func (s *SimScreen) Screen() tcell.SimulationScreen { return s.screen }

// Width returns the screen width.
func (s *SimScreen) Width() int { return s.width }

// Height returns the screen height.
func (s *SimScreen) Height() int { return s.height }

// Cell returns the rune at position (x, y).
func (s *SimScreen) Cell(x, y int) rune {
	cells, w, _ := s.screen.GetContents()
	if x < 0 || y < 0 || x >= w || y*w+x >= len(cells) {
		return 0
	}
	mainc, _, _, _ := s.screen.GetContent(x, y)
	return mainc
}

// CellStyle returns the tcell.Style at position (x, y).
func (s *SimScreen) CellStyle(x, y int) tcell.Style {
	_, _, style, _ := s.screen.GetContent(x, y)
	return style
}

// Row returns the content of row y as a string (rune-by-rune across the width).
func (s *SimScreen) Row(y int) string {
	var sb strings.Builder
	for x := 0; x < s.width; x++ {
		r := s.Cell(x, y)
		if r == 0 {
			r = ' '
		}
		sb.WriteRune(r)
	}
	return strings.TrimRight(sb.String(), " ")
}

// Rows returns all rows as a string slice (one element per row, trimmed).
func (s *SimScreen) Rows() []string {
	rows := make([]string, s.height)
	for y := 0; y < s.height; y++ {
		rows[y] = s.Row(y)
	}
	return rows
}

// Contains reports whether any row of the current screen contents contains sub.
func (s *SimScreen) Contains(sub string) bool {
	return ContainsString(s.Rows(), sub)
}

// SendKey injects a synthetic key event into the simulation screen's event queue.
func (s *SimScreen) SendKey(key tcell.Key, r rune, mod tcell.ModMask) {
	s.screen.InjectKey(key, r, mod)
}

// Show calls Show on the underlying SimulationScreen, flushing pending updates
// so that Cell / Row queries reflect the most recent render.
func (s *SimScreen) Show() { s.screen.Show() }

// Sync calls Sync on the underlying SimulationScreen.
func (s *SimScreen) Sync() { s.screen.Sync() }

// Fini shuts down the simulation screen.
func (s *SimScreen) Fini() { s.screen.Fini() }

// ─── Internal: ensure tcell is exercised so the import is not dropped ────────

var _ = time.Now // keep time import if needed in future helpers
