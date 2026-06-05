package widget

import (
	"strconv"
	"strings"
	"unicode"

	oat "github.com/antoniocali/oat-latte"
	"github.com/antoniocali/oat-latte/latte"
	"github.com/gdamore/tcell/v2"
)

// Language identifies the programming language used for syntax highlighting.
type Language int

const (
	// LangAuto attempts to detect the language from content heuristics.
	LangAuto Language = iota
	// LangGo highlights Go source files.
	LangGo
	// LangPython highlights Python source files.
	LangPython
	// LangJavaScript highlights JavaScript and TypeScript source files.
	LangJavaScript
	// LangRust highlights Rust source files.
	LangRust
	// LangShell highlights shell (bash/sh) scripts.
	LangShell
	// LangText applies no syntax highlighting.
	LangText
)

// tokenKind classifies a code token for colour mapping.
type tokenKind int

const (
	tokenNormal  tokenKind = iota
	tokenKeyword           // language keyword
	tokenString            // string literal
	tokenComment           // inline comment
	tokenNumber            // numeric literal
)

// codeToken is a single syntax-coloured fragment of a source line.
type codeToken struct {
	text string
	kind tokenKind
}

// keyword sets for each supported language.
var keywordsGo = map[string]bool{
	"func": true, "var": true, "const": true, "type": true, "struct": true,
	"interface": true, "return": true, "if": true, "else": true, "for": true,
	"range": true, "package": true, "import": true, "go": true, "defer": true,
	"chan": true, "select": true, "case": true, "default": true, "break": true,
	"continue": true, "nil": true, "true": true, "false": true, "error": true,
	"map": true, "switch": true, "fallthrough": true, "goto": true, "make": true,
	"new": true, "delete": true, "len": true, "cap": true, "append": true,
}

var keywordsPython = map[string]bool{
	"def": true, "class": true, "return": true, "if": true, "elif": true,
	"else": true, "for": true, "while": true, "import": true, "from": true,
	"as": true, "with": true, "try": true, "except": true, "finally": true,
	"raise": true, "pass": true, "None": true, "True": true, "False": true,
	"and": true, "or": true, "not": true, "in": true, "is": true,
	"lambda": true, "yield": true, "global": true, "nonlocal": true, "del": true,
	"break": true, "continue": true, "assert": true, "async": true, "await": true,
}

var keywordsJS = map[string]bool{
	"function": true, "const": true, "let": true, "var": true, "return": true,
	"if": true, "else": true, "for": true, "while": true, "class": true,
	"import": true, "export": true, "from": true, "new": true, "this": true,
	"typeof": true, "instanceof": true, "null": true, "undefined": true,
	"true": true, "false": true, "async": true, "await": true, "=>": true,
	"switch": true, "case": true, "break": true, "continue": true, "default": true,
	"try": true, "catch": true, "finally": true, "throw": true, "delete": true,
	"in": true, "of": true, "do": true, "extends": true, "super": true,
}

var keywordsRust = map[string]bool{
	"fn": true, "let": true, "mut": true, "const": true, "struct": true,
	"enum": true, "impl": true, "trait": true, "pub": true, "use": true,
	"mod": true, "crate": true, "self": true, "super": true, "return": true,
	"if": true, "else": true, "match": true, "for": true, "while": true,
	"loop": true, "break": true, "continue": true, "true": true, "false": true,
	"None": true, "Some": true, "Ok": true, "Err": true, "where": true,
	"type": true, "dyn": true, "async": true, "await": true, "move": true,
	"ref": true, "in": true, "as": true,
}

var keywordsShell = map[string]bool{
	"if": true, "then": true, "else": true, "elif": true, "fi": true,
	"for": true, "while": true, "do": true, "done": true, "case": true,
	"esac": true, "function": true, "return": true, "local": true,
	"export": true, "echo": true, "exit": true, "in": true, "shift": true,
	"source": true, "readonly": true, "unset": true,
}

// CodeView is a syntax-highlighted, scrollable code viewer.
//
// Default keybindings (when focused):
//   - ↑ / ↓      Scroll up / down one line
//   - PgUp / PgDn  Scroll half a page
type CodeView struct {
	oat.BaseComponent
	oat.FocusBehavior

	lines      []string
	lang       Language
	scrollOff  int
	cursorLine int
	lineNums   bool

	// styles
	style        latte.Style
	keywordStyle latte.Style
	stringStyle  latte.Style
	commentStyle latte.Style
	numberStyle  latte.Style
	lineNumStyle latte.Style
}

// NewCodeView creates a CodeView displaying the given source code.
// Language is set to LangAuto by default.
func NewCodeView(code string) *CodeView {
	cv := &CodeView{lang: LangAuto}
	cv.EnsureID()
	cv.SetCode(code)
	cv.rebuildDefaultStyles()
	return cv
}

// WithLanguage sets the syntax highlighting language explicitly.
func (cv *CodeView) WithLanguage(lang Language) *CodeView { cv.lang = lang; return cv }

// WithLineNumbers controls whether line numbers are shown in the gutter.
func (cv *CodeView) WithLineNumbers(show bool) *CodeView { cv.lineNums = show; return cv }

// WithID sets a user-defined identifier on this component.
func (cv *CodeView) WithID(id string) *CodeView { cv.ID = id; return cv }

// SetCode replaces the displayed source code.
func (cv *CodeView) SetCode(code string) {
	cv.lines = strings.Split(code, "\n")
	cv.scrollOff = 0
	cv.cursorLine = 0
}

// ApplyTheme applies semantic theme tokens to CodeView styles.
func (cv *CodeView) ApplyTheme(t latte.Theme) {
	cv.style = t.Text
	cv.keywordStyle = latte.Style{FG: t.Accent.FG, BG: t.Text.BG, Bold: true}
	cv.stringStyle = latte.Style{FG: t.Success.FG, BG: t.Text.BG}
	cv.commentStyle = latte.Style{FG: t.Muted.FG, BG: t.Text.BG, Italic: true}
	cv.numberStyle = latte.Style{FG: t.Warning.FG, BG: t.Text.BG}
	cv.lineNumStyle = t.Muted
}

// rebuildDefaultStyles sets neutral default styles for use before a theme is applied.
func (cv *CodeView) rebuildDefaultStyles() {
	cv.keywordStyle = latte.Style{FG: latte.ColorBrightCyan, Bold: true}
	cv.stringStyle = latte.Style{FG: latte.ColorBrightGreen}
	cv.commentStyle = latte.Style{FG: latte.ColorBrightBlack, Italic: true}
	cv.numberStyle = latte.Style{FG: latte.ColorBrightYellow}
	cv.lineNumStyle = latte.Style{FG: latte.ColorBrightBlack}
}

// --- oat.Scrollable ---

// ScrollOffset returns the current scroll offset in lines.
func (cv *CodeView) ScrollOffset() int { return cv.scrollOff }

// ContentHeight returns the total number of lines in the code.
func (cv *CodeView) ContentHeight() int { return len(cv.lines) }

// ScrollTo sets the scroll offset, clamped to valid range.
func (cv *CodeView) ScrollTo(off int) {
	max := len(cv.lines) - 1
	if off < 0 {
		off = 0
	}
	if off > max {
		off = max
	}
	cv.scrollOff = off
}

// Measure returns the preferred size.
func (cv *CodeView) Measure(c oat.Constraint) oat.Size {
	h := len(cv.lines)
	if c.MaxHeight >= 0 && h > c.MaxHeight {
		h = c.MaxHeight
	}
	w := c.MaxWidth
	if w < 0 {
		w = 80 // reasonable fallback when unconstrained
	}
	return oat.Size{Width: w, Height: h}
}

// Render draws visible source lines with syntax highlighting into buf.
func (cv *CodeView) Render(buf *oat.Buffer, region oat.Region) {
	sub := buf.Sub(region)
	sub.FillBG(cv.style)

	lineNumW := 0
	if cv.lineNums && len(cv.lines) > 0 {
		lineNumW = len(strconv.Itoa(len(cv.lines))) + 2 // "N │ "
	}

	visibleH := region.Height
	lang := cv.resolvedLang()

	for row := 0; row < visibleH; row++ {
		lineIdx := cv.scrollOff + row
		if lineIdx >= len(cv.lines) {
			break
		}

		x := 0
		if cv.lineNums {
			numStr := strconv.Itoa(lineIdx + 1)
			// Right-align within lineNumW-2 to leave room for " │ ".
			padded := strings.Repeat(" ", lineNumW-2-len(numStr)) + numStr + " │"
			x = sub.DrawText(0, row, padded, cv.lineNumStyle)
			x++ // one space after │
		}

		line := cv.lines[lineIdx]
		tokens := tokenizeLine(line, lang)
		for _, tok := range tokens {
			style := cv.styleForKind(tok.kind)
			// Clamp x so we don't write past the region width.
			if x >= region.Width {
				break
			}
			// Truncate token text if it would overflow.
			runes := []rune(tok.text)
			available := region.Width - x
			if len(runes) > available {
				runes = runes[:available]
			}
			x = sub.DrawText(x, row, string(runes), style)
		}
		_ = lineNumW
	}
}

// HandleKey processes keyboard input for scrolling.
func (cv *CodeView) HandleKey(ev *oat.KeyEvent) bool {
	switch ev.Key() {
	case tcell.KeyUp:
		cv.ScrollTo(cv.scrollOff - 1)
		return true
	case tcell.KeyDown:
		cv.ScrollTo(cv.scrollOff + 1)
		return true
	case tcell.KeyPgUp:
		cv.ScrollTo(cv.scrollOff - 10)
		return true
	case tcell.KeyPgDn:
		cv.ScrollTo(cv.scrollOff + 10)
		return true
	}
	return false
}

// Children satisfies oat.Layout; CodeView has no sub-components.
func (cv *CodeView) Children() []oat.Component { return nil }

// AddChild is a no-op for CodeView.
func (cv *CodeView) AddChild(_ oat.Component) {}

// resolvedLang returns the effective Language, handling LangAuto.
func (cv *CodeView) resolvedLang() Language {
	if cv.lang != LangAuto {
		return cv.lang
	}
	// Simple auto-detect: look for common patterns in the first ~10 lines.
	check := cv.lines
	if len(check) > 10 {
		check = check[:10]
	}
	for _, line := range check {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "package ") || strings.HasPrefix(t, "func ") {
			return LangGo
		}
		if strings.HasPrefix(t, "def ") || strings.HasPrefix(t, "import ") && strings.Contains(t, "from") {
			return LangPython
		}
		if strings.HasPrefix(t, "fn ") || strings.Contains(t, "let mut ") {
			return LangRust
		}
		if strings.HasPrefix(t, "#!/") || strings.HasPrefix(t, "echo ") {
			return LangShell
		}
		if strings.HasPrefix(t, "function ") || strings.HasPrefix(t, "const ") || strings.HasPrefix(t, "import {") {
			return LangJavaScript
		}
	}
	return LangText
}

// styleForKind maps a tokenKind to the corresponding style.
func (cv *CodeView) styleForKind(k tokenKind) latte.Style {
	switch k {
	case tokenKeyword:
		return cv.keywordStyle
	case tokenString:
		return cv.stringStyle
	case tokenComment:
		return cv.commentStyle
	case tokenNumber:
		return cv.numberStyle
	default:
		return cv.style
	}
}

// tokenizeLine splits a single source line into codeTokens for the given language.
// The algorithm is a simple left-to-right scanner; it handles:
//   - Single-line comments (// or #)
//   - String literals delimited by ", ', or `
//   - Keywords (whole-word match)
//   - Decimal/hex/float numeric literals
func tokenizeLine(line string, lang Language) []codeToken {
	var keywords map[string]bool
	commentPrefix := "//"

	switch lang {
	case LangGo:
		keywords = keywordsGo
	case LangPython:
		keywords = keywordsPython
		commentPrefix = "#"
	case LangJavaScript:
		keywords = keywordsJS
	case LangRust:
		keywords = keywordsRust
	case LangShell:
		keywords = keywordsShell
		commentPrefix = "#"
	default:
		// LangText / unknown: no keywords or comments
	}

	runes := []rune(line)
	n := len(runes)
	var tokens []codeToken
	i := 0

	for i < n {
		// Comment: rest of line
		if commentPrefix != "" && strings.HasPrefix(string(runes[i:]), commentPrefix) {
			tokens = append(tokens, codeToken{text: string(runes[i:]), kind: tokenComment})
			return tokens
		}

		// String literal
		if runes[i] == '"' || runes[i] == '\'' || runes[i] == '`' {
			delim := runes[i]
			j := i + 1
			for j < n {
				if runes[j] == '\\' {
					j += 2
					continue
				}
				if runes[j] == delim {
					j++
					break
				}
				j++
			}
			tokens = append(tokens, codeToken{text: string(runes[i:j]), kind: tokenString})
			i = j
			continue
		}

		// Number: leading digit (handles 0x hex, floats, etc.)
		if unicode.IsDigit(runes[i]) {
			j := i + 1
			for j < n && (unicode.IsDigit(runes[j]) || runes[j] == '.' || runes[j] == '_' ||
				runes[j] == 'x' || runes[j] == 'X' ||
				(runes[j] >= 'a' && runes[j] <= 'f') ||
				(runes[j] >= 'A' && runes[j] <= 'F')) {
				j++
			}
			tokens = append(tokens, codeToken{text: string(runes[i:j]), kind: tokenNumber})
			i = j
			continue
		}

		// Identifier or keyword
		if unicode.IsLetter(runes[i]) || runes[i] == '_' {
			j := i + 1
			for j < n && (unicode.IsLetter(runes[j]) || unicode.IsDigit(runes[j]) || runes[j] == '_') {
				j++
			}
			word := string(runes[i:j])
			kind := tokenNormal
			if keywords != nil && keywords[word] {
				kind = tokenKeyword
			}
			tokens = append(tokens, codeToken{text: word, kind: kind})
			i = j
			continue
		}

		// Any other character: accumulate into a normal token.
		j := i + 1
		for j < n {
			r := runes[j]
			// Stop before characters that start a new token type.
			if r == '"' || r == '\'' || r == '`' || unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
				break
			}
			if commentPrefix != "" && strings.HasPrefix(string(runes[j:]), commentPrefix) {
				break
			}
			j++
		}
		tokens = append(tokens, codeToken{text: string(runes[i:j]), kind: tokenNormal})
		i = j
	}
	return tokens
}
