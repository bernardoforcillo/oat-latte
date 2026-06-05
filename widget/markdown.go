package widget

import (
	"strings"
	"unicode/utf8"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
)

// mdSpan is a fragment of text with an associated style, produced by parseLine.
type mdSpan struct {
	text  string
	style latte.Style
}

// Markdown renders inline-formatted markdown text in the terminal.
//
// Supported inline markers:
//   - **bold**
//   - *italic*
//   - `code`
//   - ~~strikethrough~~ (rendered with Reverse)
//   - [link text](url)  (rendered as underlined text; URL is discarded)
//
// Block-level constructs (headings, lists, fences) are not parsed — the raw
// text is word-wrapped and each line is scanned for the above inline tokens.
type Markdown struct {
	oat.BaseComponent

	text    string   // raw markdown text
	wrapped []string // word-wrapped lines, rebuilt when measured width changes
	width   int      // last measured width

	style latte.Style // base text style

	// derived styles, built from style in ApplyTheme / constructor
	boldStyle   latte.Style
	italicStyle latte.Style
	codeStyle   latte.Style
	strikeStyle latte.Style
	linkStyle   latte.Style
}

// NewMarkdown creates a Markdown widget displaying the given text.
func NewMarkdown(text string) *Markdown {
	m := &Markdown{text: text}
	m.EnsureID()
	m.rebuildDerivedStyles()
	return m
}

// WithID sets a user-defined identifier on this component.
func (m *Markdown) WithID(id string) *Markdown { m.ID = id; return m }

// SetText replaces the displayed text and invalidates the wrap cache.
func (m *Markdown) SetText(text string) {
	m.text = text
	m.wrapped = nil
	m.width = 0
}

// ApplyTheme applies semantic theme tokens to the Markdown widget.
func (m *Markdown) ApplyTheme(t latte.Theme) {
	m.style = t.Text
	m.rebuildDerivedStyles()
}

// rebuildDerivedStyles derives the inline-format styles from the base style.
func (m *Markdown) rebuildDerivedStyles() {
	m.boldStyle = latte.Style{FG: m.style.FG, BG: m.style.BG, Bold: true}
	m.italicStyle = latte.Style{FG: m.style.FG, BG: m.style.BG, Italic: true}
	m.codeStyle = latte.Style{FG: m.style.FG, BG: m.style.BG, Reverse: true}
	m.strikeStyle = latte.Style{FG: m.style.FG, BG: m.style.BG, Reverse: true}
	m.linkStyle = latte.Style{FG: m.style.FG, BG: m.style.BG, Underline: true}
}

// Measure returns the preferred size given c.
// If MaxWidth differs from the last measurement the word-wrap cache is rebuilt.
func (m *Markdown) Measure(c oat.Constraint) oat.Size {
	maxW := c.MaxWidth
	if maxW < 0 {
		maxW = 0
	}
	if maxW != m.width || m.wrapped == nil {
		m.wrapped = mdWordWrap(m.text, maxW)
		m.width = maxW
	}

	h := len(m.wrapped)
	w := maxW
	if c.MaxWidth < 0 {
		// Unconstrained: use the widest rendered line.
		w = 0
		for _, line := range m.wrapped {
			lw := utf8.RuneCountInString(line)
			if lw > w {
				w = lw
			}
		}
	}
	return c.Clamp(oat.Size{Width: w, Height: h})
}

// Render draws the formatted markdown text into buf within region.
func (m *Markdown) Render(buf *oat.Buffer, region oat.Region) {
	sub := buf.Sub(region)
	sub.FillBG(m.style)

	if m.wrapped == nil || m.width != region.Width {
		m.wrapped = mdWordWrap(m.text, region.Width)
		m.width = region.Width
	}

	styles := map[string]latte.Style{
		"bold":   m.boldStyle,
		"italic": m.italicStyle,
		"code":   m.codeStyle,
		"strike": m.strikeStyle,
		"link":   m.linkStyle,
		"normal": m.style,
	}

	for y, line := range m.wrapped {
		if y >= region.Height {
			break
		}
		renderMarkdownLine(sub, y, line, styles)
	}
}

// Children satisfies oat.Layout; Markdown has no sub-components.
func (m *Markdown) Children() []oat.Component { return nil }

// AddChild is a no-op for Markdown.
func (m *Markdown) AddChild(_ oat.Component) {}

// renderMarkdownLine parses inline markdown in line and draws the resulting
// spans at row y into sub.
func renderMarkdownLine(sub *oat.Buffer, y int, line string, styles map[string]latte.Style) {
	spans := parseLine(line, styles)
	x := 0
	for _, sp := range spans {
		x = sub.DrawText(x, y, sp.text, sp.style)
	}
}

// parseLine scans line for inline markdown markers and returns a slice of
// mdSpan values with appropriate styles.
//
// Supported tokens (in order of precedence):
//   - **bold**
//   - *italic*
//   - `code`
//   - ~~strikethrough~~
//   - [text](url)
func parseLine(line string, styles map[string]latte.Style) []mdSpan {
	var spans []mdSpan
	normal := styles["normal"]
	bold := styles["bold"]
	italic := styles["italic"]
	code := styles["code"]
	strike := styles["strike"]
	link := styles["link"]

	runes := []rune(line)
	n := len(runes)
	start := 0

	flush := func(end int, style latte.Style) {
		if end > start {
			spans = append(spans, mdSpan{text: string(runes[start:end]), style: style})
		}
	}

	i := 0
	for i < n {
		// Bold: **...**
		if i+1 < n && runes[i] == '*' && runes[i+1] == '*' {
			flush(i, normal)
			close := strings.Index(string(runes[i+2:]), "**")
			if close >= 0 {
				innerEnd := i + 2 + close
				spans = append(spans, mdSpan{text: string(runes[i+2 : innerEnd]), style: bold})
				i = innerEnd + 2
				start = i
				continue
			}
		}
		// Italic: *...*  (single star, not double)
		if runes[i] == '*' && (i+1 >= n || runes[i+1] != '*') {
			flush(i, normal)
			close := -1
			for j := i + 1; j < n; j++ {
				if runes[j] == '*' && (j+1 >= n || runes[j+1] != '*') {
					close = j
					break
				}
			}
			if close >= 0 {
				spans = append(spans, mdSpan{text: string(runes[i+1 : close]), style: italic})
				i = close + 1
				start = i
				continue
			}
		}
		// Inline code: `...`
		if runes[i] == '`' {
			flush(i, normal)
			close := -1
			for j := i + 1; j < n; j++ {
				if runes[j] == '`' {
					close = j
					break
				}
			}
			if close >= 0 {
				spans = append(spans, mdSpan{text: string(runes[i+1 : close]), style: code})
				i = close + 1
				start = i
				continue
			}
		}
		// Strikethrough: ~~...~~
		if i+1 < n && runes[i] == '~' && runes[i+1] == '~' {
			flush(i, normal)
			close := strings.Index(string(runes[i+2:]), "~~")
			if close >= 0 {
				innerEnd := i + 2 + close
				spans = append(spans, mdSpan{text: string(runes[i+2 : innerEnd]), style: strike})
				i = innerEnd + 2
				start = i
				continue
			}
		}
		// Link: [text](url)
		if runes[i] == '[' {
			flush(i, normal)
			closeB := -1
			for j := i + 1; j < n; j++ {
				if runes[j] == ']' {
					closeB = j
					break
				}
			}
			if closeB >= 0 && closeB+1 < n && runes[closeB+1] == '(' {
				closeP := -1
				for j := closeB + 2; j < n; j++ {
					if runes[j] == ')' {
						closeP = j
						break
					}
				}
				if closeP >= 0 {
					linkText := string(runes[i+1 : closeB])
					spans = append(spans, mdSpan{text: linkText, style: link})
					i = closeP + 1
					start = i
					continue
				}
			}
		}
		i++
	}
	// Flush remaining normal text.
	if start < n {
		spans = append(spans, mdSpan{text: string(runes[start:]), style: normal})
	}
	return spans
}

// mdWordWrap splits text (which may contain newlines) into lines of at most
// width runes, breaking on word boundaries where possible.
// A width of 0 means no wrapping.
func mdWordWrap(text string, width int) []string {
	rawLines := strings.Split(text, "\n")
	if width <= 0 {
		return rawLines
	}
	var out []string
	for _, raw := range rawLines {
		out = append(out, mdWrapLine(raw, width)...)
	}
	return out
}

// mdWrapLine wraps a single line.
func mdWrapLine(line string, width int) []string {
	if utf8.RuneCountInString(line) <= width {
		return []string{line}
	}
	words := strings.Fields(line)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	current := ""
	for _, word := range words {
		if current == "" {
			current = word
			continue
		}
		if utf8.RuneCountInString(current)+1+utf8.RuneCountInString(word) <= width {
			current += " " + word
		} else {
			lines = append(lines, current)
			current = word
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
