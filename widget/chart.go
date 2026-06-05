package widget

import (
	"fmt"
	"math"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
)

// ── SparklineChart ────────────────────────────────────────────────────────────

// sparkBlocks holds the 9-level Unicode block characters used to render
// sparklines. Index 0 is a space (zero height), index 8 is a full block.
var sparkBlocks = []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// SparklineChart is a single-row sparkline widget that uses Unicode block
// characters (▁▂▃▄▅▆▇█) to visualise a series of float64 values.
//
// There are no interactive keybindings; SparklineChart is display-only.
type SparklineChart struct {
	oat.BaseComponent

	data  []float64 // values to display
	max   float64   // 0 = auto-scale from the data
	label string

	// callerStyle preserves the style set explicitly by the caller so that
	// ApplyTheme can replace it cleanly on theme switches.
	callerStyle latte.Style
}

// NewSparklineChart creates a SparklineChart displaying the given data slice.
func NewSparklineChart(data []float64) *SparklineChart {
	c := &SparklineChart{
		data: append([]float64{}, data...),
	}
	c.EnsureID()
	return c
}

// WithLabel sets an optional label displayed above the sparkline row.
func (c *SparklineChart) WithLabel(label string) *SparklineChart {
	c.label = label
	return c
}

// WithMax fixes the scale maximum. Pass 0 to restore auto-scaling.
func (c *SparklineChart) WithMax(max float64) *SparklineChart {
	c.max = max
	return c
}

// WithID sets a user-defined identifier on this component.
func (c *SparklineChart) WithID(id string) *SparklineChart { c.ID = id; return c }

// WithStyle sets the display style for this SparklineChart.
func (c *SparklineChart) WithStyle(s latte.Style) *SparklineChart {
	c.Style = s
	c.callerStyle = s
	return c
}

// ApplyTheme applies the Accent token as the sparkline colour.
func (c *SparklineChart) ApplyTheme(t latte.Theme) {
	c.Style = t.Accent.Merge(c.callerStyle)
}

func (c *SparklineChart) Measure(con oat.Constraint) oat.Size {
	h := 1
	if c.label != "" {
		h++
	}
	w := len(c.data)
	if con.MaxWidth >= 0 && w > con.MaxWidth {
		w = con.MaxWidth
	}
	if w < 1 {
		w = 1
	}
	return oat.Size{Width: w, Height: h}
}

func (c *SparklineChart) Render(buf *oat.Buffer, region oat.Region) {
	sub := buf.Sub(region)
	sub.FillBG(c.Style)

	y := 0
	if c.label != "" {
		labelStyle := c.Style
		labelStyle.Bold = false
		sub.DrawText(0, 0, c.label, labelStyle)
		y++
	}

	if y >= region.Height || len(c.data) == 0 {
		return
	}

	// Determine scale maximum.
	maxVal := c.max
	if maxVal <= 0 {
		for _, v := range c.data {
			if v > maxVal {
				maxVal = v
			}
		}
	}
	if maxVal <= 0 {
		maxVal = 1 // avoid division by zero
	}

	for i, v := range c.data {
		x := i
		if x >= region.Width {
			break
		}
		// Normalise to 0..1 then map to 0..8 block levels.
		norm := v / maxVal
		if norm < 0 {
			norm = 0
		}
		if norm > 1 {
			norm = 1
		}
		level := int(math.Round(norm * float64(len(sparkBlocks)-1)))
		if level < 0 {
			level = 0
		}
		if level >= len(sparkBlocks) {
			level = len(sparkBlocks) - 1
		}
		sub.SetCell(x, y, sparkBlocks[level], c.Style)
	}
}

// ── BarChart ──────────────────────────────────────────────────────────────────

// BarEntry is a single data point in a BarChart.
type BarEntry struct {
	Label string
	Value float64
	Style latte.Style // optional per-bar override (zero = use chart default)
}

// BarChart displays vertical bars for a set of BarEntry values.
// Each bar is 3 characters wide with 1 character of spacing between bars.
//
// There are no interactive keybindings; BarChart is display-only.
type BarChart struct {
	oat.BaseComponent

	data       []BarEntry
	maxVal     float64 // 0 = auto-scale
	showLabels bool

	barStyle latte.Style // default per-bar fill style

	// callerStyle preserves the style set by the caller before any ApplyTheme.
	callerStyle latte.Style
}

// NewBarChart creates an empty BarChart.
func NewBarChart() *BarChart {
	c := &BarChart{
		showLabels: true,
	}
	c.EnsureID()
	return c
}

// WithEntry appends an entry to the chart with the given label and value.
func (c *BarChart) WithEntry(label string, value float64) *BarChart {
	c.data = append(c.data, BarEntry{Label: label, Value: value})
	return c
}

// WithBarStyle sets the default style for bars that do not carry their own Style.
func (c *BarChart) WithBarStyle(s latte.Style) *BarChart {
	c.barStyle = s
	return c
}

// WithShowLabels controls whether entry labels are rendered below each bar.
func (c *BarChart) WithShowLabels(show bool) *BarChart {
	c.showLabels = show
	return c
}

// WithID sets a user-defined identifier on this component.
func (c *BarChart) WithID(id string) *BarChart { c.ID = id; return c }

// WithStyle sets the base display style for this BarChart.
func (c *BarChart) WithStyle(s latte.Style) *BarChart {
	c.Style = s
	c.callerStyle = s
	return c
}

// ApplyTheme applies theme tokens to the BarChart.
func (c *BarChart) ApplyTheme(t latte.Theme) {
	c.Style = t.Text.Merge(c.callerStyle)
	c.barStyle = t.Accent
}

func (c *BarChart) Measure(con oat.Constraint) oat.Size {
	const barW = 3
	w := len(c.data) * (barW + 1)
	if w < 1 {
		w = 1
	}
	if con.MaxWidth >= 0 && w > con.MaxWidth {
		w = con.MaxWidth
	}
	h := 10
	if con.MaxHeight >= 0 && con.MaxHeight > 0 {
		h = con.MaxHeight
	}
	return oat.Size{Width: w, Height: h}
}

func (c *BarChart) Render(buf *oat.Buffer, region oat.Region) {
	sub := buf.Sub(region)
	sub.FillBG(c.Style)

	if len(c.data) == 0 || region.Height == 0 {
		return
	}

	// Determine scale maximum.
	maxVal := c.maxVal
	if maxVal <= 0 {
		for _, e := range c.data {
			if e.Value > maxVal {
				maxVal = e.Value
			}
		}
	}
	if maxVal <= 0 {
		maxVal = 1
	}

	// Reserve the bottom row for labels when enabled.
	labelH := 0
	if c.showLabels {
		labelH = 1
	}
	availH := region.Height - labelH
	if availH < 1 {
		availH = 1
	}

	const barW = 3

	for i, entry := range c.data {
		x := i * (barW + 1)
		if x >= region.Width {
			break
		}

		// Determine per-bar fill style.
		fillStyle := c.barStyle
		if entry.Style != (latte.Style{}) {
			fillStyle = entry.Style
		}
		if fillStyle == (latte.Style{}) {
			fillStyle = c.Style
		}

		// Calculate bar height in cells.
		norm := entry.Value / maxVal
		if norm < 0 {
			norm = 0
		}
		if norm > 1 {
			norm = 1
		}
		barH := int(math.Round(float64(availH) * norm))
		if barH < 0 {
			barH = 0
		}
		if barH > availH {
			barH = availH
		}

		// Draw bar from the bottom up.
		for row := 0; row < barH; row++ {
			y := availH - 1 - row
			for col := 0; col < barW; col++ {
				cx := x + col
				if cx >= region.Width {
					break
				}
				sub.SetCell(cx, y, '█', fillStyle)
			}
		}

		// Draw label below bar.
		if c.showLabels && labelH > 0 {
			labelY := region.Height - 1
			label := entry.Label
			labelRunes := []rune(label)
			if len(labelRunes) > barW {
				labelRunes = labelRunes[:barW]
			}
			for j, r := range labelRunes {
				cx := x + j
				if cx >= region.Width {
					break
				}
				sub.SetCell(cx, labelY, r, c.Style)
			}
		}
	}
}

// ── Gauge ─────────────────────────────────────────────────────────────────────

// Gauge is a single-line horizontal progress gauge with an optional percentage
// label and an optional text label at the start of the bar.
//
// There are no interactive keybindings; Gauge is display-only.
type Gauge struct {
	oat.BaseComponent

	value       float64 // 0.0 .. 1.0
	label       string
	showPercent bool

	filledStyle latte.Style // style for the filled portion (t.Accent)
	emptyStyle  latte.Style // style for the empty portion (t.Muted)

	// callerStyle preserves the style set by the caller before any theme application.
	callerStyle latte.Style
}

// NewGauge creates a Gauge with the given initial value (0.0–1.0).
// Values outside the range are clamped on construction.
func NewGauge(value float64) *Gauge {
	g := &Gauge{
		showPercent: true,
	}
	g.EnsureID()
	g.SetValue(value)
	return g
}

// WithLabel sets an optional text label drawn at the very start of the gauge.
func (g *Gauge) WithLabel(label string) *Gauge {
	g.label = label
	return g
}

// WithShowPercent controls whether a "XX%" overlay is drawn in the centre.
func (g *Gauge) WithShowPercent(show bool) *Gauge {
	g.showPercent = show
	return g
}

// WithID sets a user-defined identifier on this component.
func (g *Gauge) WithID(id string) *Gauge { g.ID = id; return g }

// WithStyle sets the base display style for this Gauge.
func (g *Gauge) WithStyle(s latte.Style) *Gauge {
	g.Style = s
	g.callerStyle = s
	return g
}

// SetValue updates the gauge value; it is clamped to [0.0, 1.0].
func (g *Gauge) SetValue(v float64) {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	g.value = v
}

// GetValue implements oat.ValueGetter. Returns the current value as a float64.
func (g *Gauge) GetValue() interface{} { return g.value }

// ApplyTheme applies theme tokens to the Gauge.
// The filled portion uses t.Accent; the empty portion uses t.Muted.
func (g *Gauge) ApplyTheme(t latte.Theme) {
	g.filledStyle = t.Accent.Merge(g.callerStyle)
	g.emptyStyle = t.Muted
}

func (g *Gauge) Measure(c oat.Constraint) oat.Size {
	w := c.MaxWidth
	if w < 0 {
		w = 20
	}
	return oat.Size{Width: w, Height: 1}
}

func (g *Gauge) Render(buf *oat.Buffer, region oat.Region) {
	sub := buf.Sub(region)
	sub.FillBG(g.Style)

	if region.Width == 0 {
		return
	}

	filledStyle := g.filledStyle
	emptyStyle := g.emptyStyle
	// Fallback when no theme has been applied.
	if filledStyle == (latte.Style{}) {
		filledStyle = g.Style
	}
	if emptyStyle == (latte.Style{}) {
		emptyStyle = g.Style
	}

	// Reserve space for the label at the start if set.
	labelRunes := []rune(g.label)
	labelW := len(labelRunes)
	if labelW >= region.Width {
		labelW = region.Width
	}

	barX := labelW
	barW := region.Width - barX
	if barW < 0 {
		barW = 0
	}

	// Number of filled cells.
	filledW := int(float64(barW) * g.value)
	if filledW < 0 {
		filledW = 0
	}
	if filledW > barW {
		filledW = barW
	}

	// Draw filled portion.
	for x := 0; x < filledW; x++ {
		sub.SetCell(barX+x, 0, '█', filledStyle)
	}
	// Draw empty portion.
	for x := filledW; x < barW; x++ {
		sub.SetCell(barX+x, 0, '░', emptyStyle)
	}

	// Overlay the label text at the start.
	if labelW > 0 {
		sub.DrawText(0, 0, string(labelRunes[:labelW]), g.Style)
	}

	// Overlay the percentage label centred over the bar.
	if g.showPercent && barW >= 4 {
		pct := fmt.Sprintf("%d%%", int(g.value*100))
		pctRunes := []rune(pct)
		pctW := len(pctRunes)
		centerX := barX + (barW-pctW)/2
		if centerX < barX {
			centerX = barX
		}
		// Draw each percentage rune with the style of the underlying cell so
		// the text reads clearly against both the filled and empty portions.
		for i, r := range pctRunes {
			cx := centerX + i
			if cx >= region.Width {
				break
			}
			var cellStyle latte.Style
			if cx-barX < filledW {
				cellStyle = filledStyle
			} else {
				cellStyle = emptyStyle
			}
			// Ensure the percentage text stands out by reversing the cell colour.
			cellStyle.Reverse = true
			sub.SetCell(cx, 0, r, cellStyle)
		}
	}
}
