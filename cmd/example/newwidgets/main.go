// Package main demonstrates Phase 1-3 widgets: Spinner, Tabs, Table,
// TreeView, Autocomplete, Charts, Markdown, CodeView, DevTools, SplitPane.
package main

import (
	"fmt"
	"log"
	"math"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/antoniocali/oat-latte/layout"
	"github.com/antoniocali/oat-latte/widget"
)

func main() {
	canvas := oat.NewCanvas()

	// DevTools overlay — toggle with F12.
	devTools := widget.NewDevTools()

	body := buildBody(canvas)
	devTools.SetRoot(body)

	canvas = oat.NewCanvas(
		oat.WithTheme(latte.ThemeDark),
		oat.WithBody(layout.NewScrollView(body)),
		oat.WithAutoStatusBar(widget.NewStatusBar()),
		oat.WithGlobalKeyBinding(devTools.ToKeyBinding()),
	)
	canvas.ShowPersistentOverlay(devTools)

	if err := canvas.Run(); err != nil {
		log.Fatal(err)
	}
}

func buildBody(canvas *oat.Canvas) oat.Component {
	return layout.NewVBox(
		spinnerSection(canvas),
		layout.NewVGap(1),
		tabsSection(),
		layout.NewVGap(1),
		tableSection(),
		layout.NewVGap(1),
		treeSection(),
		layout.NewVGap(1),
		autocompleteSection(),
		layout.NewVGap(1),
		chartsSection(),
		layout.NewVGap(1),
		markdownSection(),
		layout.NewVGap(1),
		codeSection(),
		layout.NewVGap(1),
		splitPaneSection(),
	)
}

// ── Spinner ───────────────────────────────────────────────────────────────────

func spinnerSection(canvas *oat.Canvas) oat.Component {
	spinner := widget.NewSpinner(widget.SpinnerBraille, canvas.Redraw).
		WithLabel("Processing…")
	spinner.Start()

	return layout.NewVBox(
		widget.NewTitle("Spinner").WithSeparator(true),
		layout.NewVGap(1),
		layout.NewHBox(
			spinner,
			layout.NewHGap(2),
			widget.NewSpinner(widget.SpinnerLine, canvas.Redraw).
				WithLabel("Loading"),
		),
	)
}

// ── Tabs ──────────────────────────────────────────────────────────────────────

func tabsSection() oat.Component {
	info := widget.NewMarkdown(`**Tabs** let you switch between panels with ← and →.
Each tab can hold any *component* — including other layouts.`)

	code := widget.NewCodeView(`func buildTabs() *widget.Tabs {
    return widget.NewTabs(
        widget.Tab{Label: "A", Content: widget.NewText("Panel A")},
        widget.Tab{Label: "B", Content: widget.NewText("Panel B")},
    )
}`).WithLanguage(widget.LangGo).WithLineNumbers(true)

	tabs := widget.NewTabs(
		widget.Tab{Label: "Info", Content: info},
		widget.Tab{Label: "Code", Content: code},
		widget.Tab{Label: "Empty", Content: widget.NewText("(Nothing here yet)")},
	)

	return layout.NewVBox(
		widget.NewTitle("Tabs").WithSeparator(true),
		layout.NewVGap(1),
		tabs,
	)
}

// ── Table ─────────────────────────────────────────────────────────────────────

func tableSection() oat.Component {
	cols := []widget.Column{
		{Header: "Framework", Width: 18},
		{Header: "Language", Width: 12},
		{Header: "Paradigm", Width: 16},
		{Header: "Stars", Width: 7},
	}
	table := widget.NewTable(cols).
		WithShowHeader(true).
		WithRows([][]string{
			{"oat-latte", "Go", "Reactive/TUI", "★★★★★"},
			{"Bubble Tea", "Go", "Elm / TUI", "★★★★★"},
			{"tview", "Go", "Imperative", "★★★★☆"},
			{"Textual", "Python", "Reactive/TUI", "★★★★★"},
			{"Ink", "JavaScript", "React/TUI", "★★★★☆"},
			{"blessed", "JavaScript", "Imperative", "★★★☆☆"},
			{"crossterm", "Rust", "Low-level", "★★★★☆"},
		})

	return layout.NewVBox(
		widget.NewTitle("Table  (↑↓ Home End)").WithSeparator(true),
		layout.NewVGap(1),
		table,
	)
}

// ── TreeView ──────────────────────────────────────────────────────────────────

func treeSection() oat.Component {
	// Build a file-system-like tree.
	src := widget.NewTreeNode("src", nil)
	cmd := widget.NewTreeNode("cmd", nil)
	cmd.AddChild(widget.NewTreeNode("main.go", "file"))
	src.AddChild(cmd)
	pkg := widget.NewTreeNode("pkg", nil)
	pkg.AddChild(widget.NewTreeNode("widget", nil).
		AddChild(widget.NewTreeNode("button.go", "file")).
		AddChild(widget.NewTreeNode("table.go", "file")).
		AddChild(widget.NewTreeNode("treeview.go", "file")))
	pkg.AddChild(widget.NewTreeNode("layout", nil).
		AddChild(widget.NewTreeNode("box.go", "file")).
		AddChild(widget.NewTreeNode("splitpane.go", "file")))
	src.AddChild(pkg)
	src.Expand()

	docs := widget.NewTreeNode("docs", nil)
	docs.AddChild(widget.NewTreeNode("README.md", "file"))
	docs.AddChild(widget.NewTreeNode("CONTRIBUTING.md", "file"))

	tv := widget.NewTreeView().
		AddRoot(src).
		AddRoot(docs).
		WithOnSelect(func(n *widget.TreeNode) {
			_ = fmt.Sprintf("selected: %s", n.Label)
		})

	return layout.NewVBox(
		widget.NewTitle("TreeView  (↑↓ ←→ Enter)").WithSeparator(true),
		layout.NewVGap(1),
		tv,
	)
}

// ── Autocomplete ──────────────────────────────────────────────────────────────

func autocompleteSection() oat.Component {
	ac := widget.NewAutocomplete([]string{
		"go build", "go test", "go run", "go mod tidy",
		"git status", "git commit", "git push", "git pull",
		"make build", "make test", "make lint", "make clean",
	}).WithHint("Command search").
		WithPlaceholder("Type a command…").
		WithMaxItems(6)

	return layout.NewVBox(
		widget.NewTitle("Autocomplete  (type to filter, ↑↓ Enter)").WithSeparator(true),
		layout.NewVGap(1),
		ac,
	)
}

// ── Charts ────────────────────────────────────────────────────────────────────

func chartsSection() oat.Component {
	// Sparkline: cosine wave.
	data := make([]float64, 50)
	for i := range data {
		data[i] = math.Cos(float64(i)*0.3)*0.5 + 0.5
	}
	spark := widget.NewSparklineChart(data).WithLabel("Cosine wave")

	// Bar chart: simulated memory usage.
	bar := widget.NewBarChart().WithShowLabels(true)
	for _, entry := range []struct{ label string; v float64 }{
		{"Go", 42}, {"Py", 78}, {"JS", 65}, {"Rust", 28}, {"Java", 91},
	} {
		bar.WithEntry(entry.label, entry.v)
	}

	// Gauges.
	gaugeRow := layout.NewVBox(
		widget.NewGauge(0.38).WithLabel("CPU ").WithShowPercent(true),
		widget.NewGauge(0.72).WithLabel("RAM ").WithShowPercent(true),
		widget.NewGauge(0.15).WithLabel("Disk").WithShowPercent(true),
	)

	return layout.NewVBox(
		widget.NewTitle("Charts  (SparklineChart, BarChart, Gauge)").WithSeparator(true),
		layout.NewVGap(1),
		spark,
		layout.NewVGap(1),
		bar,
		layout.NewVGap(1),
		gaugeRow,
	)
}

// ── Markdown ──────────────────────────────────────────────────────────────────

func markdownSection() oat.Component {
	md := widget.NewMarkdown(`# widget.Markdown

Render **bold**, *italic*, ` + "`inline code`" + `, ~~strikethrough~~, and [links](url).

> Use Markdown for *rich* documentation panes, changelogs, or **help text**
> without needing a separate HTML renderer.`)

	return layout.NewVBox(
		widget.NewTitle("Markdown").WithSeparator(true),
		layout.NewVGap(1),
		md,
	)
}

// ── CodeView ──────────────────────────────────────────────────────────────────

func codeSection() oat.Component {
	code := `package widget

import (
	"sync"
	"time"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
)

// Spinner is an animated loading indicator.
type Spinner struct {
	oat.BaseComponent
	style    SpinnerStyle
	running  bool
	mu       sync.Mutex
	stopCh   chan struct{}
	redrawFn func()
}

// Start begins the animation loop. Idempotent.
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return  // already running
	}
	s.running = true
	s.stopCh = make(chan struct{})
	stopCh := s.stopCh
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if s.redrawFn != nil {
					s.redrawFn()
				}
			case <-stopCh:
				return
			}
		}
	}()
}`

	cv := widget.NewCodeView(code).
		WithLanguage(widget.LangGo).
		WithLineNumbers(true)

	return layout.NewVBox(
		widget.NewTitle("CodeView  (↑↓ PgUp PgDn)").WithSeparator(true),
		layout.NewVGap(1),
		cv,
	)
}

// ── SplitPane ─────────────────────────────────────────────────────────────────

func splitPaneSection() oat.Component {
	left := layout.NewBorder(
		layout.NewVBox(
			widget.NewText("Left panel").WithStyle(latte.Style{Bold: true}),
			widget.NewText("Alt+← to shrink"),
		),
	).WithTitle("First", oat.AnchorLeft)

	right := layout.NewBorder(
		layout.NewVBox(
			widget.NewText("Right panel").WithStyle(latte.Style{Bold: true}),
			widget.NewText("Alt+→ to grow"),
		),
	).WithTitle("Second", oat.AnchorLeft)

	sp := layout.NewSplitPane(left, right, layout.SplitHorizontal).
		WithRatio(0.4)

	return layout.NewVBox(
		widget.NewTitle("SplitPane  (Alt+←→)").WithSeparator(true),
		layout.NewVGap(1),
		sp,
	)
}
