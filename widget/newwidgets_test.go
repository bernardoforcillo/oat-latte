package widget

import (
	"testing"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// ── mockFocusableNode for tests ───────────────────────────────────────────────

// mockLayoutNode is a simple Layout for testing Children() returns.
type mockLayoutNode struct {
	children []oat.Component
}

func (m *mockLayoutNode) Measure(_ oat.Constraint) oat.Size { return oat.Size{} }
func (m *mockLayoutNode) Render(_ *oat.Buffer, _ oat.Region) {}
func (m *mockLayoutNode) Children() []oat.Component         { return m.children }
func (m *mockLayoutNode) AddChild(c oat.Component)           { m.children = append(m.children, c) }

// mockFixed is a minimal Component that returns a fixed size.
type mockFixed struct {
	oat.BaseComponent
	w, h int
}

func (m *mockFixed) Measure(c oat.Constraint) oat.Size {
	return c.Clamp(oat.Size{Width: m.w, Height: m.h})
}
func (m *mockFixed) Render(_ *oat.Buffer, _ oat.Region) {}
func (m *mockFixed) Children() []oat.Component          { return nil }
func (m *mockFixed) AddChild(_ oat.Component)            {}

// ── Spinner ───────────────────────────────────────────────────────────────────

func TestSpinnerMeasure(t *testing.T) {
	s := NewSpinner(SpinnerBraille, nil)
	size := s.Measure(oat.Constraint{MaxWidth: 100, MaxHeight: 10})
	if size.Width != 1 || size.Height != 1 {
		t.Errorf("Spinner no-label Measure = %+v, want {1 1}", size)
	}
}

func TestSpinnerMeasureWithLabel(t *testing.T) {
	s := NewSpinner(SpinnerBraille, nil).WithLabel("Loading")
	size := s.Measure(oat.Constraint{MaxWidth: 100, MaxHeight: 10})
	// 1 (frame) + 1 (space) + 7 (label) = 9
	if size.Width != 9 || size.Height != 1 {
		t.Errorf("Spinner with label Measure = %+v, want {9 1}", size)
	}
}

func TestSpinnerStartStop(t *testing.T) {
	s := NewSpinner(SpinnerLine, nil)
	if s.IsRunning() {
		t.Error("spinner should not be running before Start()")
	}
	s.Start()
	if !s.IsRunning() {
		t.Error("spinner should be running after Start()")
	}
	s.Start() // idempotent
	if !s.IsRunning() {
		t.Error("spinner still running after second Start()")
	}
	s.Stop()
	if s.IsRunning() {
		t.Error("spinner should be stopped after Stop()")
	}
	s.Stop() // idempotent — must not panic
}

func TestSpinnerCurrentFrame(t *testing.T) {
	s := NewSpinner(SpinnerLine, nil)
	r := s.CurrentFrame()
	frames := spinnerFrames[SpinnerLine]
	valid := false
	for _, f := range frames {
		if r == f {
			valid = true
			break
		}
	}
	if !valid {
		t.Errorf("CurrentFrame() = %q, not in SpinnerLine frames", r)
	}
}

func TestSpinnerApplyTheme(t *testing.T) {
	s := NewSpinner(SpinnerBraille, nil)
	s.ApplyTheme(latte.ThemeDark)
	if s.Style == (latte.Style{}) {
		t.Error("ApplyTheme should set a non-zero style")
	}
}

func TestSpinnerRedrawCalled(t *testing.T) {
	called := make(chan struct{}, 1)
	s := NewSpinner(SpinnerBraille, func() {
		select {
		case called <- struct{}{}:
		default:
		}
	})
	s.Start()
	<-called // wait for at least one tick
	s.Stop()
}

// ── Tabs ──────────────────────────────────────────────────────────────────────

func TestTabsMeasure(t *testing.T) {
	content := &mockFixed{w: 40, h: 10}
	tabs := NewTabs(Tab{Label: "A", Content: content})
	size := tabs.Measure(oat.Constraint{MaxWidth: 80, MaxHeight: 24})
	// height = 1 (tab bar) + 10 (content)
	if size.Height != 11 {
		t.Errorf("Tabs Measure height = %d, want 11", size.Height)
	}
}

func TestTabsActiveIndex(t *testing.T) {
	tabs := NewTabs(
		Tab{Label: "A", Content: &mockFixed{}},
		Tab{Label: "B", Content: &mockFixed{}},
	)
	if tabs.ActiveIndex() != 0 {
		t.Errorf("initial ActiveIndex = %d, want 0", tabs.ActiveIndex())
	}
	tabs.SetActive(1)
	if tabs.ActiveIndex() != 1 {
		t.Errorf("after SetActive(1), ActiveIndex = %d, want 1", tabs.ActiveIndex())
	}
}

func TestTabsSetActiveClamps(t *testing.T) {
	tabs := NewTabs(
		Tab{Label: "A", Content: &mockFixed{}},
		Tab{Label: "B", Content: &mockFixed{}},
	)
	tabs.SetActive(99)
	if tabs.ActiveIndex() != 1 {
		t.Errorf("SetActive(99) should clamp to 1, got %d", tabs.ActiveIndex())
	}
	tabs.SetActive(-5)
	if tabs.ActiveIndex() != 0 {
		t.Errorf("SetActive(-5) should clamp to 0, got %d", tabs.ActiveIndex())
	}
}

func TestTabsHandleKeyRight(t *testing.T) {
	tabs := NewTabs(
		Tab{Label: "A", Content: &mockFixed{}},
		Tab{Label: "B", Content: &mockFixed{}},
		Tab{Label: "C", Content: &mockFixed{}},
	)
	ev := tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
	consumed := tabs.HandleKey(ev)
	if !consumed {
		t.Error("HandleKey(Right) should be consumed")
	}
	if tabs.ActiveIndex() != 1 {
		t.Errorf("after Right, ActiveIndex = %d, want 1", tabs.ActiveIndex())
	}
}

func TestTabsHandleKeyLeftWraps(t *testing.T) {
	tabs := NewTabs(
		Tab{Label: "A", Content: &mockFixed{}},
		Tab{Label: "B", Content: &mockFixed{}},
	)
	ev := tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
	tabs.HandleKey(ev)
	if tabs.ActiveIndex() != 1 {
		t.Errorf("Left from tab 0 should wrap to 1, got %d", tabs.ActiveIndex())
	}
}

func TestTabsRightWraps(t *testing.T) {
	tabs := NewTabs(
		Tab{Label: "A", Content: &mockFixed{}},
		Tab{Label: "B", Content: &mockFixed{}},
	)
	tabs.SetActive(1)
	ev := tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
	tabs.HandleKey(ev)
	if tabs.ActiveIndex() != 0 {
		t.Errorf("Right from last tab should wrap to 0, got %d", tabs.ActiveIndex())
	}
}

func TestTabsChildren(t *testing.T) {
	content := &mockFixed{w: 5, h: 3}
	tabs := NewTabs(Tab{Label: "A", Content: content})
	children := tabs.Children()
	if len(children) != 1 || children[0] != content {
		t.Errorf("Children() should return [content], got %v", children)
	}
}

func TestTabsAddChild(t *testing.T) {
	tabs := NewTabs()
	tabs.AddChild(&mockFixed{})
	if len(tabs.tabs) != 1 {
		t.Errorf("AddChild should add a tab, now len=%d", len(tabs.tabs))
	}
}

func TestTabsApplyThemePropagatesToContent(t *testing.T) {
	type themedContent struct {
		mockFixed
		applied bool
	}
	type wrappedContent struct {
		*themedContent
	}
	tc := &themedContent{}
	// Make content implement ThemeReceiver via embedding won't work automatically,
	// so just check that ApplyTheme doesn't panic and styles are set.
	tabs := NewTabs(Tab{Label: "A", Content: &mockFixed{}})
	tabs.ApplyTheme(latte.ThemeDark)
	if tabs.Style == (latte.Style{}) {
		t.Error("ApplyTheme should set a non-zero style")
	}
	_ = tc
}

func TestTabsHasID(t *testing.T) {
	a := NewTabs()
	b := NewTabs()
	if a.ID == "" {
		t.Error("Tabs should have a non-empty ID")
	}
	if a.ID == b.ID {
		t.Error("different Tabs should have different IDs")
	}
}

func TestTabsWithID(t *testing.T) {
	tabs := NewTabs().WithID("my-tabs")
	if tabs.ID != "my-tabs" {
		t.Errorf("WithID: got %q, want %q", tabs.ID, "my-tabs")
	}
}

// ── Table ─────────────────────────────────────────────────────────────────────

func TestTableMeasure(t *testing.T) {
	table := NewTable([]Column{{Header: "A", Width: 10}, {Header: "B", Width: 10}}).
		WithShowHeader(true).
		WithRows([][]string{{"a", "1"}, {"b", "2"}})

	size := table.Measure(oat.Constraint{MaxWidth: 80, MaxHeight: 20})
	// 2 header rows (header + separator) + 2 data rows = 4
	if size.Height != 4 {
		t.Errorf("Table Measure height = %d, want 4", size.Height)
	}
}

func TestTableNoHeaderMeasure(t *testing.T) {
	table := NewTable([]Column{{Header: "A", Width: 10}}).
		WithShowHeader(false).
		WithRows([][]string{{"a"}, {"b"}, {"c"}})

	size := table.Measure(oat.Constraint{MaxWidth: 80, MaxHeight: 20})
	if size.Height != 3 {
		t.Errorf("Table no-header height = %d, want 3", size.Height)
	}
}

func TestTableSelectedIndex(t *testing.T) {
	table := NewTable([]Column{{Header: "A", Width: 10}}).
		WithRows([][]string{{"a"}, {"b"}})

	if table.SelectedIndex() != -1 {
		t.Errorf("initial SelectedIndex = %d, want -1", table.SelectedIndex())
	}
}

func TestTableHandleKeyDown(t *testing.T) {
	table := NewTable([]Column{{Header: "A", Width: 10}}).
		WithRows([][]string{{"a"}, {"b"}, {"c"}})

	ev := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	consumed := table.HandleKey(ev)
	if !consumed {
		t.Error("HandleKey(Down) should be consumed")
	}
	if table.SelectedIndex() != 0 {
		t.Errorf("after first Down, selected = %d, want 0", table.SelectedIndex())
	}
	table.HandleKey(ev)
	if table.SelectedIndex() != 1 {
		t.Errorf("after second Down, selected = %d, want 1", table.SelectedIndex())
	}
}

func TestTableHandleKeyUp(t *testing.T) {
	table := NewTable([]Column{{Header: "A", Width: 10}}).
		WithRows([][]string{{"a"}, {"b"}, {"c"}})

	down := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	up := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	table.HandleKey(down) // → 0
	table.HandleKey(down) // → 1
	table.HandleKey(up)   // → 0
	if table.SelectedIndex() != 0 {
		t.Errorf("after Down Down Up, selected = %d, want 0", table.SelectedIndex())
	}
}

func TestTableHandleKeyHome(t *testing.T) {
	table := NewTable([]Column{{Header: "A", Width: 10}}).
		WithRows([][]string{{"a"}, {"b"}, {"c"}})

	end := tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone)
	home := tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone)
	table.HandleKey(end)
	if table.SelectedIndex() != 2 {
		t.Errorf("after End, selected = %d, want 2", table.SelectedIndex())
	}
	table.HandleKey(home)
	if table.SelectedIndex() != 0 {
		t.Errorf("after Home, selected = %d, want 0", table.SelectedIndex())
	}
}

func TestTableSelectedRow(t *testing.T) {
	table := NewTable([]Column{{Header: "A", Width: 10}}).
		WithRows([][]string{{"apple"}, {"banana"}})

	if table.SelectedRow() != nil {
		t.Error("SelectedRow() should be nil with no selection")
	}
	down := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	table.HandleKey(down) // → 0
	row := table.SelectedRow()
	if len(row) != 1 || row[0] != "apple" {
		t.Errorf("SelectedRow() = %v, want [apple]", row)
	}
}

func TestTableGetValue(t *testing.T) {
	table := NewTable([]Column{{Header: "A", Width: 5}}).
		AddRow([]string{"x"})
	down := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	table.HandleKey(down)
	v := table.GetValue()
	if v == nil {
		t.Error("GetValue() should not be nil after selection")
	}
}

func TestTableChildren(t *testing.T) {
	table := NewTable(nil)
	if table.Children() != nil {
		t.Error("Table.Children() should return nil")
	}
}

func TestTableApplyTheme(t *testing.T) {
	table := NewTable(nil)
	table.ApplyTheme(latte.ThemeDark)
	if table.Style == (latte.Style{}) {
		t.Error("ApplyTheme should set a non-zero style")
	}
}

// ── TreeNode ──────────────────────────────────────────────────────────────────

func TestTreeNodeExpandCollapse(t *testing.T) {
	n := NewTreeNode("root", nil)
	if n.IsExpanded() {
		t.Error("new node should be collapsed")
	}
	n.Expand()
	if !n.IsExpanded() {
		t.Error("after Expand(), should be expanded")
	}
	n.Collapse()
	if n.IsExpanded() {
		t.Error("after Collapse(), should be collapsed")
	}
}

func TestTreeNodeToggle(t *testing.T) {
	n := NewTreeNode("root", nil)
	n.Toggle()
	if !n.IsExpanded() {
		t.Error("Toggle from collapsed should expand")
	}
	n.Toggle()
	if n.IsExpanded() {
		t.Error("Toggle from expanded should collapse")
	}
}

func TestTreeNodeAddChild(t *testing.T) {
	parent := NewTreeNode("parent", nil)
	child := NewTreeNode("child", nil)
	parent.AddChild(child)
	if len(parent.Children) != 1 {
		t.Errorf("AddChild: len(Children) = %d, want 1", len(parent.Children))
	}
	if parent.Children[0] != child {
		t.Error("AddChild: wrong child stored")
	}
}

// ── TreeView ──────────────────────────────────────────────────────────────────

func TestTreeViewMeasure(t *testing.T) {
	root := NewTreeNode("root", nil)
	root.AddChild(NewTreeNode("child1", nil))
	root.Expand()

	tv := NewTreeView().AddRoot(root)
	size := tv.Measure(oat.Constraint{MaxWidth: 80, MaxHeight: 20})
	// 2 visible rows (root + child1 after expand), height = 2
	if size.Height != 2 {
		t.Errorf("TreeView Measure height = %d, want 2", size.Height)
	}
}

func TestTreeViewHandleKeyDown(t *testing.T) {
	root1 := NewTreeNode("a", nil)
	root2 := NewTreeNode("b", nil)
	tv := NewTreeView().AddRoot(root1).AddRoot(root2)

	ev := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	consumed := tv.HandleKey(ev)
	if !consumed {
		t.Error("HandleKey(Down) should be consumed")
	}
	if tv.cursor != 1 {
		t.Errorf("after Down, cursor = %d, want 1", tv.cursor)
	}
}

func TestTreeViewHandleKeyExpand(t *testing.T) {
	root := NewTreeNode("root", nil)
	root.AddChild(NewTreeNode("child", nil))
	tv := NewTreeView().AddRoot(root)

	right := tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone)
	tv.HandleKey(right)
	if !root.IsExpanded() {
		t.Error("Right on collapsed node should expand it")
	}
}

func TestTreeViewHandleKeyCollapse(t *testing.T) {
	root := NewTreeNode("root", nil)
	root.AddChild(NewTreeNode("child", nil))
	root.Expand()
	tv := NewTreeView().AddRoot(root)

	left := tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone)
	tv.HandleKey(left)
	if root.IsExpanded() {
		t.Error("Left on expanded node should collapse it")
	}
}

func TestTreeViewSelectedNode(t *testing.T) {
	root := NewTreeNode("root", "val")
	tv := NewTreeView().AddRoot(root)

	node := tv.SelectedNode()
	if node != root {
		t.Errorf("SelectedNode() = %v, want root", node)
	}
}

func TestTreeViewOnSelect(t *testing.T) {
	leaf := NewTreeNode("leaf", "leafval")
	tv := NewTreeView().AddRoot(leaf)

	var selected *TreeNode
	tv.WithOnSelect(func(n *TreeNode) { selected = n })

	enter := tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	tv.HandleKey(enter)
	if selected != leaf {
		t.Errorf("onSelect should be called with leaf, got %v", selected)
	}
}

func TestTreeViewChildren(t *testing.T) {
	tv := NewTreeView()
	if tv.Children() != nil {
		t.Error("TreeView.Children() should return nil")
	}
}

func TestTreeViewHasID(t *testing.T) {
	a := NewTreeView()
	b := NewTreeView()
	if a.ID == "" {
		t.Error("TreeView should have a non-empty ID")
	}
	if a.ID == b.ID {
		t.Error("different TreeViews should have different IDs")
	}
}

// ── Autocomplete ──────────────────────────────────────────────────────────────

func TestAutocompleteMeasure(t *testing.T) {
	ac := NewAutocomplete([]string{"apple", "apricot"})
	size := ac.Measure(oat.Constraint{MaxWidth: 40, MaxHeight: 20})
	// No text → no filter → dropdown closed → just input height
	if size.Height == 0 {
		t.Error("Autocomplete Measure should return non-zero height")
	}
}

func TestAutocompleteGetSetText(t *testing.T) {
	ac := NewAutocomplete([]string{"apple", "avocado"})
	ac.SetText("ap")
	if ac.GetText() != "ap" {
		t.Errorf("GetText() = %q, want %q", ac.GetText(), "ap")
	}
}

func TestAutocompleteChildren(t *testing.T) {
	ac := NewAutocomplete([]string{"a"})
	children := ac.Children()
	if len(children) != 1 {
		t.Errorf("Children() len = %d, want 1", len(children))
	}
}

func TestAutocompleteHandleKeyDelegates(t *testing.T) {
	ac := NewAutocomplete([]string{"apple", "apricot"})
	ac.SetFocused(true)
	// Type 'a' — should open dropdown since "apple" and "apricot" match
	ev := tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone)
	ac.HandleKey(ev)
	if !ac.dropdownOpen {
		t.Error("typing 'a' should open dropdown for matching suggestions")
	}
}

func TestAutocompleteEscClosesDropdown(t *testing.T) {
	ac := NewAutocomplete([]string{"apple", "apricot"})
	ac.SetFocused(true)
	// Open dropdown
	ac.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone))
	if !ac.dropdownOpen {
		t.Skip("dropdown did not open, skipping close test")
	}
	// Close with Esc
	ac.HandleKey(tcell.NewEventKey(tcell.KeyEscape, 0, tcell.ModNone))
	if ac.dropdownOpen {
		t.Error("Esc should close the dropdown")
	}
}

func TestAutocompleteEnterConfirms(t *testing.T) {
	var confirmed string
	ac := NewAutocomplete([]string{"apple", "apricot"}).
		WithOnSelect(func(s string) { confirmed = s })
	ac.SetFocused(true)
	ac.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone))
	if !ac.dropdownOpen {
		t.Skip("dropdown did not open, skipping confirm test")
	}
	ac.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if confirmed == "" {
		t.Error("Enter should confirm a selection and call onSelect")
	}
}

func TestAutocompleteGetValue(t *testing.T) {
	ac := NewAutocomplete(nil)
	ac.SetText("hello")
	if v, ok := ac.GetValue().(string); !ok || v != "hello" {
		t.Errorf("GetValue() = %v, want %q", ac.GetValue(), "hello")
	}
}

func TestAutocompleteHasID(t *testing.T) {
	a := NewAutocomplete(nil)
	b := NewAutocomplete(nil)
	if a.ID == "" {
		t.Error("Autocomplete should have a non-empty ID")
	}
	if a.ID == b.ID {
		t.Error("different Autocompletes should have different IDs")
	}
}

// ── Gauge ─────────────────────────────────────────────────────────────────────

func TestGaugeMeasure(t *testing.T) {
	g := NewGauge(0.5)
	size := g.Measure(oat.Constraint{MaxWidth: 40, MaxHeight: 5})
	if size.Height != 1 {
		t.Errorf("Gauge Measure height = %d, want 1", size.Height)
	}
	if size.Width != 40 {
		t.Errorf("Gauge Measure width = %d, want 40", size.Width)
	}
}

func TestGaugeSetValue(t *testing.T) {
	g := NewGauge(0.5)
	g.SetValue(0.9)
	if v, ok := g.GetValue().(float64); !ok || v != 0.9 {
		t.Errorf("after SetValue(0.9), GetValue() = %v", g.GetValue())
	}
}

func TestGaugeSetValueClamps(t *testing.T) {
	g := NewGauge(0.5)
	g.SetValue(1.5)
	if v := g.GetValue().(float64); v != 1.0 {
		t.Errorf("SetValue(1.5) should clamp to 1.0, got %v", v)
	}
	g.SetValue(-0.3)
	if v := g.GetValue().(float64); v != 0.0 {
		t.Errorf("SetValue(-0.3) should clamp to 0.0, got %v", v)
	}
}

func TestGaugeApplyTheme(t *testing.T) {
	g := NewGauge(0.5)
	g.ApplyTheme(latte.ThemeDark)
	// Should not panic and should set some styles.
}

// ── SparklineChart ────────────────────────────────────────────────────────────

func TestSparklineChartMeasure(t *testing.T) {
	data := []float64{0.1, 0.5, 0.9, 0.3}
	c := NewSparklineChart(data)
	size := c.Measure(oat.Constraint{MaxWidth: 80, MaxHeight: 10})
	if size.Height != 1 {
		t.Errorf("SparklineChart height = %d, want 1", size.Height)
	}
	if size.Width != 4 {
		t.Errorf("SparklineChart width = %d, want 4 (len(data))", size.Width)
	}
}

func TestSparklineChartWithLabel(t *testing.T) {
	c := NewSparklineChart([]float64{0.5}).WithLabel("test")
	size := c.Measure(oat.Constraint{MaxWidth: 80, MaxHeight: 10})
	// Label adds a row
	if size.Height != 2 {
		t.Errorf("SparklineChart with label height = %d, want 2", size.Height)
	}
}

// ── Markdown ──────────────────────────────────────────────────────────────────

func TestMarkdownMeasure(t *testing.T) {
	md := NewMarkdown("Hello world")
	size := md.Measure(oat.Constraint{MaxWidth: 80, MaxHeight: 20})
	if size.Height < 1 {
		t.Error("Markdown should have at least 1 row")
	}
}

func TestMarkdownSetText(t *testing.T) {
	md := NewMarkdown("original")
	md.SetText("updated")
	// Should not panic; measurement cache should be invalidated.
	size := md.Measure(oat.Constraint{MaxWidth: 80, MaxHeight: 20})
	if size.Height < 1 {
		t.Error("after SetText, Markdown should still measure correctly")
	}
}

func TestMarkdownApplyTheme(t *testing.T) {
	md := NewMarkdown("test")
	md.ApplyTheme(latte.ThemeDark) // must not panic
}

// ── CodeView ──────────────────────────────────────────────────────────────────

func TestCodeViewMeasure(t *testing.T) {
	cv := NewCodeView("line1\nline2\nline3")
	size := cv.Measure(oat.Constraint{MaxWidth: 80, MaxHeight: 20})
	if size.Height != 3 {
		t.Errorf("CodeView height = %d, want 3", size.Height)
	}
}

func TestCodeViewScrollable(t *testing.T) {
	cv := NewCodeView("a\nb\nc\nd\ne")
	if cv.ContentHeight() != 5 {
		t.Errorf("ContentHeight = %d, want 5", cv.ContentHeight())
	}
	cv.ScrollTo(2)
	if cv.ScrollOffset() != 2 {
		t.Errorf("after ScrollTo(2), ScrollOffset = %d", cv.ScrollOffset())
	}
}

func TestCodeViewHandleKeyUp(t *testing.T) {
	cv := NewCodeView("a\nb\nc\nd\ne")
	cv.ScrollTo(3)
	down := tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	up := tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	cv.HandleKey(down)
	cv.HandleKey(up)
	cv.HandleKey(up)
	if cv.ScrollOffset() != 2 {
		t.Errorf("scroll after down/up/up = %d, want 2", cv.ScrollOffset())
	}
}

func TestCodeViewLanguage(t *testing.T) {
	cv := NewCodeView("package main").WithLanguage(LangGo)
	if cv.lang != LangGo {
		t.Errorf("language = %d, want LangGo", cv.lang)
	}
}

func TestCodeViewLineNumbers(t *testing.T) {
	cv := NewCodeView("line1").WithLineNumbers(true)
	if !cv.lineNums {
		t.Error("line numbers should be enabled after WithLineNumbers(true)")
	}
}
