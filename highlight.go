package main

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	chromaStyles "github.com/alecthomas/chroma/v2/styles"
	"charm.land/glamour/v2"
	"charm.land/glamour/v2/ansi"
	glamourStyles "charm.land/glamour/v2/styles"
)

// reChromaError matches ANSI sequences that chroma emits for Error tokens
// (they have a red/dark background set by the style, e.g. \e[38;2;Rm\e[48;2;Bm...\e[0m).
// We capture the inner text to recolor it.
var reChromaError = regexp.MustCompile(`(?:\x1b\[[0-9;]*m)+([^\x1b]+)(?:\x1b\[[0-9;]*m)+`)

// packageManagerCmds maps CLI tool names and subcommands to their desired ANSI color.
var packageManagerColors = map[string]string{
	"winget":  "\033[38;2;88;166;255m",  // blue
	"scoop":   "\033[38;2;88;166;255m",  // blue
	"powershell": "\033[38;2;88;166;255m",  // blue
	"nu": "\033[38;2;0;255;0m", // green
	"cargo":   "\033[38;2;88;166;255m",  // blue
	"npm":     "\033[38;2;88;166;255m",  // blue
	"pip":     "\033[38;2;88;166;255m",  // blue
	"brew":    "\033[38;2;88;166;255m",  // blue
	"apt":     "\033[38;2;88;166;255m",  // blue
	"choco":   "\033[38;2;88;166;255m",  // blue
	"install": "\033[38;2;241;250;140m", // yellow
	"add":     "\033[38;2;241;250;140m", // yellow
	"remove":  "\033[38;2;241;250;140m", // yellow
	"update":  "\033[38;2;241;250;140m", // yellow
	"upgrade": "\033[38;2;241;250;140m", // yellow
	"run":     "\033[38;2;241;250;140m", // yellow
	"build":   "\033[38;2;241;250;140m", // yellow
}

// reColorWord matches a whole word for recoloring.
var reColorWord = regexp.MustCompile(`\b(winget|scoop|powershell|nu|cargo|npm|pip|brew|apt|choco|install|add|remove|update|upgrade|run|build)\b`)

// applyPackageManagerColors post-processes a highlighted line to recolor
// known CLI tools and subcommands regardless of what chroma assigned them.
func applyPackageManagerColors(line string) string {
	// Work on the plain text but preserve surrounding ANSI by rebuilding:
	// Split into ANSI-token segments, recolor matching words within plain spans.
	// Simple approach: strip all ANSI, recolor words, then re-emit.
	// This loses chroma colors for other tokens on that line, so we apply
	// selectively only when the line starts with a known package manager command.
	plain := stripANSI(line)
	trimmed := strings.TrimSpace(plain)

	// Check if line begins with a known package manager command.
	firstWord := strings.SplitN(trimmed, " ", 2)[0]
	if _, ok := packageManagerColors[firstWord]; !ok {
		return line
	}

	// Rebuild line from plain text with selective coloring.
	result := reColorWord.ReplaceAllStringFunc(plain, func(word string) string {
		if color, ok := packageManagerColors[word]; ok {
			return color + word + "\033[0m"
		}
		return word
	})
	return result
}

// highlightCodeByLang highlights content using the language name directly (e.g. "powershell", "go").
func highlightCodeByLang(content, lang string) string {
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := chromaStyles.Get("github-dark")
	if style == nil {
		style = chromaStyles.Fallback
	}

	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Get("terminal256")
	}
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return content
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return content
	}

	result := buf.String()
	result = strings.TrimRight(result, "\n")

	// Post-process: recolor package manager commands and subcommands.
	resultLines := strings.Split(result, "\n")
	for i, rl := range resultLines {
		resultLines[i] = applyPackageManagerColors(rl)
	}
	return strings.Join(resultLines, "\n")
}

// splColors maps SPL keywords/commands to the desired ANSI truecolor sequence.
var splColors = map[string]string{
	"IN":       "\033[38;2;232;145;45m", // orange
	"spath":    "\033[38;2;88;166;255m", // blue
	"eval":     "\033[38;2;88;166;255m", // blue
	"sort":     "\033[38;2;88;166;255m", // blue
	"table":    "\033[38;2;88;166;255m", // blue
	"search":   "\033[38;2;88;166;255m", // blue
	"stats":    "\033[38;2;88;166;255m", // blue
	"by":       "\033[38;2;88;166;255m", // blue
	"input":    "\033[38;2;63;185;80m",  // green
	"count":    "\033[38;2;63;185;80m",  // green
	"strftime": "\033[38;2;189;147;249m", // purple
}

// reSplKeyword matches whole-word SPL keywords/commands for recoloring.
var reSplKeyword = regexp.MustCompile(`\b(IN|spath|eval|sort|table|search|stats|by|input|count|strftime)\b`)

// splAppNameFields lists SPL field names whose quoted value should be
// highlighted as a "project/app name" regardless of what that name is
// (instead of relying on a hardcoded list of known app names).
var splAppNameFields = []string{"cf_app_name", "ApplicationName"}

// reSplAppNameValue matches `<field>="value"` for the fields above, capturing
// the field name and the quoted value separately so the value can be
// recolored no matter what project/app it refers to.
var reSplAppNameValue = regexp.MustCompile(`\b(` + strings.Join(splAppNameFields, "|") + `)(=)"([^"]*)"`)

const splAppNameColor = "\033[38;2;241;250;140m" // yellow

// highlightSPL applies simple whole-word keyword coloring to Splunk SPL (.spl) content.
func highlightSPL(content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		line = reSplAppNameValue.ReplaceAllStringFunc(line, func(m string) string {
			parts := reSplAppNameValue.FindStringSubmatch(m)
			if len(parts) < 4 {
				return m
			}
			return parts[1] + parts[2] + "\"" + splAppNameColor + parts[3] + ansiReset + "\""
		})
		line = reSplKeyword.ReplaceAllStringFunc(line, func(word string) string {
			if color, ok := splColors[word]; ok {
				return color + word + ansiReset
			}
			return word
		})
		lines[i] = line
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n")
}

// expandTabs replaces tab characters with spaces, aligning to tabWidth-wide
// stops. This prevents visual misalignment in the TUI: a raw '\t' renders at
// whatever width the terminal happens to use for its tab stops, which made
// content like SPL pipe ("|") characters appear to jump left/right from one
// line to the next when a file mixed tabs and spaces for indentation.
func expandTabs(content string, tabWidth int) string {
	if tabWidth <= 0 {
		tabWidth = 4
	}
	if !strings.Contains(content, "\t") {
		return content
	}
	lines := strings.Split(content, "\n")
	for li, line := range lines {
		if !strings.Contains(line, "\t") {
			continue
		}
		var b strings.Builder
		col := 0
		for _, r := range line {
			if r == '\t' {
				spaces := tabWidth - (col % tabWidth)
				b.WriteString(strings.Repeat(" ", spaces))
				col += spaces
			} else {
				b.WriteRune(r)
				col++
			}
		}
		lines[li] = b.String()
	}
	return strings.Join(lines, "\n")
}

// normalizeLineEndings converts CRLF and lone-CR line endings to LF. Stray
// '\r' bytes left in preview content reach the terminal as a literal
// carriage return, which snaps the cursor back to column 0 mid-frame and
// corrupts whatever Lipgloss is drawing on that row (borders, adjacent
// panels, etc.) — this is what broke the TUI when opening CRLF files
// (e.g. Windows-authored .spl files).
func normalizeLineEndings(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return content
}

func highlightCode(content, filename string) string {
	if strings.HasSuffix(strings.ToLower(filename), ".spl") {
		return highlightSPL(content)
	}
	lexerName := chromaLexerForFile(filename)
	lexer := lexers.Get(lexerName)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	style := chromaStyles.Get("github-dark")
	if style == nil {
		style = chromaStyles.Fallback
	}

	// terminal16m = truecolor ANSI — renders colours correctly in Windows Terminal
	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Get("terminal256")
	}
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return content
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return content
	}

	result := buf.String()
	result = strings.TrimRight(result, "\n")
	return result
}

// ---------------------------------------------------------------------------
// Keyword colorization
// ---------------------------------------------------------------------------

// reKeywordTodo matches the word TODO (whole word) in raw markdown.
var reKeywordTodo = regexp.MustCompile(`\b(TODO)\b`)

// reKeywordDoing matches DOING or WORKING (whole word) in raw markdown.
var reKeywordDoing = regexp.MustCompile(`\b(DOING|WORKING)\b`)

// reKeywordDone matches DONE or FINISH (whole word) in raw markdown.
var reKeywordDone = regexp.MustCompile(`\b(DONE|FINISH)\b`)

// reKeywordFail matches FAIL or NOT (whole word) in raw markdown.
var reKeywordFail = regexp.MustCompile(`\b(FAIL|NOT)\b`)

// Sentinel strings — unique tokens that glamour will pass through unchanged.
const (
	sentinelTodo    = "\U000E5100"
	sentinelDoing   = "\U000E5101"
	sentinelDone    = "\U000E5102"
	sentinelFinish  = "\U000E5103"
	sentinelFail    = "\U000E5104"
	sentinelNot     = "\U000E5105"
	sentinelWorking  = "\U000E5106"
	sentinelCheckIcon = "\U000E5107"
	sentinelXIcon     = "\U000E5108"
	sentinelWorkIcon  = "\U000E5109"
)

const (
	ansiReset     = "\033[0m"
	ansiSoftBlue  = "\033[38;2;100;180;255m"
	ansiYellow    = "\033[38;2;255;200;60m"
	ansiSoftGreen = "\033[38;2;80;200;120m"
	ansiRed       = "\033[38;2;220;80;80m"
)

var (
	iconCheckGreen    = "\033[38;2;80;200;120m" + "\u2714" + ansiReset
	iconXRed          = "\033[38;2;220;80;80m" + "\uf52f" + ansiReset
	iconCircleCheck   = "\033[38;2;80;200;120m" + "\uf05d" + ansiReset
	iconWorkYellow    = "\033[38;2;255;200;60m" + "\uea6c" + ansiReset
)

// injectKeywordSentinels replaces status keywords in raw markdown with
// sentinel Unicode chars that glamour passes through unchanged.
func injectKeywordSentinels(s string) string {
	s = reKeywordTodo.ReplaceAllStringFunc(s, func(m string) string {
		switch m {
		case "TODO":
			return sentinelTodo
		}
		return m
	})
	s = reKeywordDoing.ReplaceAllStringFunc(s, func(m string) string {
		switch m {
		case "DOING":
			return sentinelDoing
		case "WORKING":
			return sentinelWorking
		}
		return m
	})
	s = reKeywordDone.ReplaceAllStringFunc(s, func(m string) string {
		switch m {
		case "DONE":
			return sentinelDone
		case "FINISH":
			return sentinelFinish
		}
		return m
	})
	s = reKeywordFail.ReplaceAllStringFunc(s, func(m string) string {
		switch m {
		case "FAIL":
			return sentinelFail
		case "NOT":
			return sentinelNot
		}
		return m
	})
	s = strings.ReplaceAll(s, ":check:", sentinelCheckIcon)
	s = strings.ReplaceAll(s, ":x:", sentinelXIcon)
	s = strings.ReplaceAll(s, ":work:", sentinelWorkIcon)
	return s
}

// restoreKeywordColors replaces sentinels in the rendered output with
// ANSI-colored keyword text (and icons where applicable).
func restoreKeywordColors(s string) string {
	s = strings.ReplaceAll(s, sentinelTodo, ansiSoftBlue+"TODO"+ansiReset)
	s = strings.ReplaceAll(s, sentinelDoing, ansiYellow+"DOING"+ansiReset)
	s = strings.ReplaceAll(s, sentinelWorking, ansiYellow+"WORKING"+ansiReset)
	s = strings.ReplaceAll(s, sentinelDone, ansiSoftGreen+"DONE"+ansiReset)
	s = strings.ReplaceAll(s, sentinelFinish, ansiSoftGreen+"FINISH"+ansiReset)
	s = strings.ReplaceAll(s, sentinelFail, ansiRed+"FAIL"+ansiReset)
	s = strings.ReplaceAll(s, sentinelNot, ansiRed+"NOT"+ansiReset)
	s = strings.ReplaceAll(s, sentinelCheckIcon, iconCircleCheck)
	s = strings.ReplaceAll(s, sentinelXIcon, iconXRed)
	s = strings.ReplaceAll(s, sentinelWorkIcon, iconWorkYellow)
	return s
}

// ---------------------------------------------------------------------------
// Math / superscript helpers
// ---------------------------------------------------------------------------

// reMathInline matches $...$ and $$...$$ LaTeX delimiters.
var reMathInline = regexp.MustCompile(`\$\$?([^$\n]+?)\$?\$`)

// reSuper matches base^exp patterns like x^2, n^{k+1}
var reSuper = regexp.MustCompile(`(\w)\^([{(]?[\w\d+\-*/=]+[)}]?)`)

var superMap = map[rune]string{
	'0': "0", '1': "1", '2': "2", '3': "3", '4': "4",
	'5': "5", '6': "6", '7': "7", '8': "8", '9': "9",
	'+': "+", '-': "-", '=': "=", 'n': "n", 'i': "i",
	'a': "a", 'b': "b", 'c': "c", 'd': "d", 'e': "e",
	'f': "f", 'g': "g", 'h': "h", 'j': "j", 'k': "k",
	'l': "l", 'm': "m", 'o': "o", 'p': "p", 'r': "r",
	's': "s", 't': "t", 'u': "u", 'v': "v", 'w': "w",
	'x': "x", 'y': "y", 'z': "z",
}

func superscript(s string) string {
	var b strings.Builder
	for _, r := range s {
		if sup, ok := superMap[r]; ok {
			b.WriteString(sup)
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// preprocessMath converts $x^2$ / x^2 to Unicode superscripts before rendering.
func preprocessMath(content string) string {
	content = reMathInline.ReplaceAllStringFunc(content, func(m string) string {
		parts := reMathInline.FindStringSubmatch(m)
		if len(parts) < 2 {
			return m
		}
		return reSuper.ReplaceAllStringFunc(parts[1], func(mm string) string {
			pp := reSuper.FindStringSubmatch(mm)
			if len(pp) < 3 {
				return mm
			}
			return pp[1] + superscript(strings.Trim(pp[2], "{}()"))
		})
	})
	content = reSuper.ReplaceAllStringFunc(content, func(m string) string {
		parts := reSuper.FindStringSubmatch(m)
		if len(parts) < 3 {
			return m
		}
		return parts[1] + superscript(strings.Trim(parts[2], "{}()"))
	})
	return content
}

func strPtr(s string) *string  { return &s }
func boolPtr(b bool) *bool    { return &b }

// renderMarkdown renders Markdown using glamour with a patched dark style:
//   - fixes red squares on JSON code blocks (Error/Punctuation tokens → neutral)
//   - extracts Markdown tables, renders glamour on the rest, then splices in leaf-style tables
func renderMarkdown(content string, width int) string {
	if width < 40 {
		width = 40
	}
	content = injectKeywordSentinels(content)
	content = preprocessMath(content)

	// Extract tables before glamour, replace with unique placeholders.
	var tableRendered []string
	content, tableRendered = extractAndRenderTables(content, width)

	// Extract fenced code blocks before glamour, replace with unique placeholders.
	var codeRendered []string
	content, codeRendered = extractAndRenderCodeBlocks(content, width)

	// Start from the built-in dark config and patch only what we need.
	cfg := glamourStyles.DarkStyleConfig

	// Code block — no background (transparent like leaf), only syntax colors.
	codeMargin := uint(1)
	codeIndent := uint(1)
	cfg.CodeBlock = ansi.StyleCodeBlock{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color: strPtr("#d4d4d4"),
			},
			Margin: &codeMargin,
			Indent: &codeIndent,
		},
		Theme: "dracula",
		Chroma: &ansi.Chroma{
			Text:                ansi.StylePrimitive{Color: strPtr("#f8f8f2")},
			Error:               ansi.StylePrimitive{Color: strPtr("#f8f8f2")},
			Comment:             ansi.StylePrimitive{Color: strPtr("#6272a4"), Italic: boolPtr(true)},
			CommentPreproc:      ansi.StylePrimitive{Color: strPtr("#ff79c6")},
			Keyword:             ansi.StylePrimitive{Color: strPtr("#50fa7b")},
			KeywordReserved:     ansi.StylePrimitive{Color: strPtr("#50fa7b")},
			KeywordNamespace:    ansi.StylePrimitive{Color: strPtr("#50fa7b")},
			KeywordType:         ansi.StylePrimitive{Color: strPtr("#50fa7b")},
			Operator:            ansi.StylePrimitive{Color: strPtr("#f8f8f2")},
			Punctuation:         ansi.StylePrimitive{Color: strPtr("#f8f8f2")},
			Name:                ansi.StylePrimitive{Color: strPtr("#f8f8f2")},
			NameBuiltin:         ansi.StylePrimitive{Color: strPtr("#8be9fd")},
			NameTag:             ansi.StylePrimitive{Color: strPtr("#ff79c6")},
			NameAttribute:       ansi.StylePrimitive{Color: strPtr("#50fa7b")},
			NameClass:           ansi.StylePrimitive{Color: strPtr("#8be9fd")},
			NameConstant:        ansi.StylePrimitive{Color: strPtr("#bd93f9")},
			NameDecorator:       ansi.StylePrimitive{Color: strPtr("#50fa7b")},
			NameException:       ansi.StylePrimitive{Color: strPtr("#8be9fd")},
			NameFunction:        ansi.StylePrimitive{Color: strPtr("#50fa7b")},
			NameOther:           ansi.StylePrimitive{Color: strPtr("#f8f8f2")},
			Literal:             ansi.StylePrimitive{Color: strPtr("#f1fa8c")},
			LiteralNumber:       ansi.StylePrimitive{Color: strPtr("#bd93f9")},
			LiteralDate:         ansi.StylePrimitive{Color: strPtr("#f1fa8c")},
			LiteralString:       ansi.StylePrimitive{Color: strPtr("#f1fa8c")},
			LiteralStringEscape: ansi.StylePrimitive{Color: strPtr("#ff79c6")},
			GenericDeleted:      ansi.StylePrimitive{Color: strPtr("#ff5555")},
			GenericEmph:         ansi.StylePrimitive{Color: strPtr("#f8f8f2"), Italic: boolPtr(true)},
			GenericInserted:     ansi.StylePrimitive{Color: strPtr("#50fa7b")},
			GenericStrong:       ansi.StylePrimitive{Color: strPtr("#f8f8f2"), Bold: boolPtr(true)},
			GenericSubheading:   ansi.StylePrimitive{Color: strPtr("#bd93f9")},
			Background:          ansi.StylePrimitive{BackgroundColor: strPtr("")},
		},
	}

	// Bold — orange.
	boldTrue := true
	upperTrue := true
	cfg.Strong = ansi.StylePrimitive{
		Color: strPtr("#e8912d"),
		Bold:  &boldTrue,
	}

	// H1 — prominent heading (no underline to avoid dotted decoration lines).
	cfg.H1 = ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
		Color: strPtr("#ffffff"), Bold: &boldTrue,
	}}

	// H2 — green heading (no underline).
	cfg.H2 = ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
		Color: strPtr("#3fb950"), Bold: &boldTrue,
	}}

	// H3 — one arrow, uppercase, orange.
	cfg.H3 = ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
		Color: strPtr("#e8912d"), Bold: &boldTrue,
	}}
	// H4 — one arrow, green.
	cfg.H4 = ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
		Color: strPtr("#a5d6ff"), Bold: &boldTrue,
	}}
	// H5/H6 — one arrow, muted.
	cfg.H5 = ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
		Color: strPtr("#a5d6ff"), Upper: &upperTrue,
	}}
	cfg.H6 = ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
		Color: strPtr("#6e7681"), Upper: &upperTrue,
	}}

	// HorizontalRule — suppress glamour's decorative horizontal lines.
	cfg.HorizontalRule = ansi.StylePrimitive{
		Color: strPtr("#6e7681"),
	}

	// Table — rendered as plain code block (replaceMarkdownTables already converted them).
	// Just hide glamour's default table separators.
	emptyStr := strPtr("")
	cfg.Table = ansi.StyleTable{
		StyleBlock:      ansi.StyleBlock{},
		CenterSeparator: emptyStr,
		ColumnSeparator: emptyStr,
		RowSeparator:    emptyStr,
	}


	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(cfg),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return content
	}
	out, err := r.Render(content)
	if err != nil {
		return content
	}
	out = restoreKeywordColors(out)
	// Remove dotted list decorations (lines with only dots/dashes after list items).
	out = removeListDecorations(out)
	// Restore pre-rendered leaf-style tables.
	for i, rendered := range tableRendered {
		placeholder := fmt.Sprintf("\x00TABLE%d\x00", i)
		out = strings.ReplaceAll(out, placeholder, "\n"+rendered+"\n")
	}
	// Restore pre-rendered code blocks.
	for i, rendered := range codeRendered {
		placeholder := fmt.Sprintf("\x00CODE%d\x00", i)
		out = strings.ReplaceAll(out, placeholder, "\n"+rendered+"\n")
	}
	return strings.TrimRight(out, "\n")
}

// reANSI matches ANSI escape sequences.
var reANSI = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// removeListDecorations removes decorative filler lines glamour adds after list items.
// These are lines that consist almost entirely of a single repeated character (─, ., -, ·, etc.)
// with a minimum length of 4 visible characters.
func removeListDecorations(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	for _, line := range lines {
		plain := strings.TrimSpace(stripANSI(line))
		if isFillerLine(plain) {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// isFillerLine returns true if the line is a decorative filler (repeated single char).
func isFillerLine(plain string) bool {
	runes := []rune(plain)
	n := len(runes)
	if n < 4 {
		return false
	}
	// Count occurrences of the first non-space character.
	first := runes[0]

	count := 0
	for _, r := range runes {
		if r == first || r == ' ' {
			count++
		}
	}
	// If 90%+ of the line is the same filler char (+ spaces), it's decorative.
	return float64(count)/float64(n) >= 0.9
}

func stripANSI(s string) string {
	return reANSI.ReplaceAllString(s, "")
}

// extractAndRenderCodeBlocks finds fenced code blocks (```lang ... ```) in markdown,
// renders them as leaf-style boxes with language label and line numbers, and replaces
// them with null-byte placeholders to survive the glamour pass.
func extractAndRenderCodeBlocks(content string, width int) (string, []string) {
	lines := strings.Split(content, "\n")
	var out []string
	var rendered []string
	i := 0
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "```") {
			lang := strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
			i++
			var codeLines []string
			for i < len(lines) {
				if strings.TrimSpace(lines[i]) == "```" {
					i++
					break
				}
				codeLines = append(codeLines, lines[i])
				i++
			}
			boxed := renderBoxCodeBlock(lang, codeLines, width)
			idx := len(rendered)
			rendered = append(rendered, boxed)
			placeholder := fmt.Sprintf("\x00CODE%d\x00", idx)
			out = append(out, "")
			out = append(out, placeholder)
			out = append(out, "")
			continue
		}
		out = append(out, lines[i])
		i++
	}
	return strings.Join(out, "\n"), rendered
}

// renderBoxCodeBlock renders a code block as a leaf-style box:
//
//	┌─ lang ──────────────┐
//	│ 1  code line one    │
//	│ 2  code line two    │
//	└─────────────────────┘
func renderBoxCodeBlock(lang string, codeLines []string, width int) string {
	// Color constants (reuse package-level ansiMuted/ansiReset2).
	const (
		ansiLang    = "\033[38;2;110;118;129m" // muted gray for label
		ansiLineNum = "\033[38;2;110;118;129m" // muted gray for line numbers
		ansiBorder  = "\033[38;2;110;118;129m" // muted gray for box border
	)

	// Apply syntax highlighting to the whole block at once for correct context,
	// then split back into per-line ANSI strings.
	var highlighted []string
	if lang != "" {
		full := highlightCodeByLang(strings.Join(codeLines, "\n")+"\n", lang)
		// Split by newline; highlightCode may emit a trailing empty line.
		hlLines := strings.Split(strings.TrimRight(full, "\n"), "\n")
		// Align with codeLines length (pad or truncate if chroma changes count).
		highlighted = make([]string, len(codeLines))
		for idx := range codeLines {
			if idx < len(hlLines) {
				highlighted[idx] = hlLines[idx]
			} else {
				highlighted[idx] = codeLines[idx]
			}
		}
	} else {
		highlighted = make([]string, len(codeLines))
		copy(highlighted, codeLines)
	}

	// Determine inner width: width minus 2 for side borders, minus some indent.
	innerW := width - 6
	if innerW < 20 {
		innerW = 20
	}

	// Build the label line: ┌─ lang ──────...─┐
	labelText := " " + lang + " "
	if lang == "" {
		labelText = ""
	}
	dashCount := innerW - len(labelText)
	if dashCount < 2 {
		dashCount = 2
	}
	topLine := ansiBorder + "┌─" + ansiLang + labelText + ansiBorder + strings.Repeat("─", dashCount) + "─┐" + ansiReset2

	bottomLine := ansiBorder + "└" + strings.Repeat("─", innerW+2) + "┘" + ansiReset2

	// Max line number width.
	numW := len(fmt.Sprintf("%d", len(codeLines)))

	var sb strings.Builder
	sb.WriteString(topLine + "\n")
	for idx, hl := range highlighted {
		lineNum := fmt.Sprintf("%*d", numW, idx+1)
		plain := stripANSI(hl)
		// Pad to innerW based on visible (plain) length.
		padNeeded := innerW - numW - 2 - len([]rune(plain))
		if padNeeded < 0 {
			padNeeded = 0
		}
		sb.WriteString(ansiBorder + "│" + ansiReset2 +
			" " + ansiLineNum + lineNum + ansiReset2 + "  " +
			hl + strings.Repeat(" ", padNeeded) +
			" " + ansiBorder + "│" + ansiReset2 + "\n")
	}
	sb.WriteString(bottomLine)
	return sb.String()
}

// extractAndRenderTables finds GFM pipe tables in markdown source, replaces each
// with a null-byte placeholder, and returns the rendered leaf-style table strings.
// After glamour renders the rest, placeholders are replaced with the rendered tables.
func extractAndRenderTables(content string, width int) (string, []string) {
	lines := strings.Split(content, "\n")
	var out []string
	var rendered []string
	i := 0
	for i < len(lines) {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "|") {
			start := i
			for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "|") {
				i++
			}
			tableLines := lines[start:i]
			boxed := renderLeafTable(tableLines, width)
			idx := len(rendered)
			rendered = append(rendered, boxed)
			placeholder := fmt.Sprintf("\x00TABLE%d\x00", idx)
			out = append(out, "")
			out = append(out, placeholder)
			out = append(out, "")
			continue
		}
		out = append(out, lines[i])
		i++
	}
	return strings.Join(out, "\n"), rendered
}

// ANSI color helpers for leaf-style tables.
const (
	ansiBlue   = "\033[38;2;88;166;255m" // #58a6ff
	ansiMuted  = "\033[38;2;110;118;129m" // #6e7681
	ansiBold   = "\033[1m"
	ansiReset2 = "\033[0m"
)

// renderLeafTable converts GFM pipe table lines into a leaf-style box-drawing table.
// Inline markdown in cells (bold, code) is rendered via glamour before layout.
func renderLeafTable(rows []string, _ int) string {
	// Parse raw cells, skipping the separator row (---|---)
	var rawRows [][]string
	for _, row := range rows {
		row = strings.TrimSpace(row)
		if row == "" {
			continue
		}
		// Skip separator rows like |---|---|
		stripped := strings.ReplaceAll(row, "-", "")
		stripped = strings.ReplaceAll(stripped, ":", "")
		stripped = strings.ReplaceAll(stripped, "|", "")
		stripped = strings.ReplaceAll(stripped, " ", "")
		if stripped == "" {
			continue
		}
		parts := strings.Split(row, "|")
		var cells []string
		for _, p := range parts {
			cells = append(cells, strings.TrimSpace(p))
		}
		if len(cells) > 0 && cells[0] == "" {
			cells = cells[1:]
		}
		if len(cells) > 0 && cells[len(cells)-1] == "" {
			cells = cells[:len(cells)-1]
		}
		if len(cells) > 0 {
			rawRows = append(rawRows, cells)
		}
	}
	if len(rawRows) == 0 {
		return ""
	}

	// Render inline markdown per cell using a tiny glamour pass.
	// We also keep the plain (ANSI-stripped) version to measure column widths.
	nCols := 0
	for _, row := range rawRows {
		if len(row) > nCols {
			nCols = len(row)
		}
	}

	type renderedRow []string // ANSI-rendered cell strings
	var rendered []renderedRow
	widths := make([]int, nCols) // visible (plain) width per column

	for ri, row := range rawRows {
		rr := make(renderedRow, nCols)
		for ci := 0; ci < nCols; ci++ {
			raw := ""
			if ci < len(row) {
				raw = row[ci]
			}
			var ansiCell string
			if ri == 0 {
				// Header: bold blue, no inline rendering needed
				ansiCell = raw
			} else {
				// Data cell: render inline markdown
				ansiCell = renderInlineMarkdown(raw)
			}
			rr[ci] = ansiCell
			plainW := len([]rune(stripANSI(ansiCell)))
			if plainW > widths[ci] {
				widths[ci] = plainW
			}
		}
		rendered = append(rendered, rr)
	}

	// Box-drawing helpers.
	hline := func(left, mid, right, fill string) string {
		var b strings.Builder
		b.WriteString(ansiMuted + left)
		for ci, w := range widths {
			b.WriteString(strings.Repeat(fill, w+2))
			if ci < len(widths)-1 {
				b.WriteString(mid)
			}
		}
		b.WriteString(right + ansiReset2)
		return b.String()
	}

	fmtCell := func(ansiText string, colWidth int, isHeader bool) string {
		plainW := len([]rune(stripANSI(ansiText)))
		pad := strings.Repeat(" ", colWidth-plainW)
		if isHeader {
			return " " + ansiBold + ansiBlue + ansiText + pad + ansiReset2 + " "
		}
		return " " + ansiText + pad + " "
	}

	row2str := func(rr renderedRow, isHeader bool) string {
		var b strings.Builder
		b.WriteString(ansiMuted + "│" + ansiReset2)
		for ci := 0; ci < nCols; ci++ {
			text := ""
			if ci < len(rr) {
				text = rr[ci]
			}
			b.WriteString(fmtCell(text, widths[ci], isHeader))
			b.WriteString(ansiMuted + "│" + ansiReset2)
		}
		return b.String()
	}

	var sb strings.Builder
	sb.WriteString(hline("┌", "┬", "┐", "─") + "\n")
	for ri, rr := range rendered {
		isHeader := ri == 0
		sb.WriteString(row2str(rr, isHeader) + "\n")
		if ri == 0 || ri < len(rendered)-1 {
			sb.WriteString(hline("├", "┼", "┤", "─") + "\n")
		}
	}
	sb.WriteString(hline("└", "┴", "┘", "─"))
	return sb.String()
}

var (
	reInlineBold   = regexp.MustCompile(`\*\*(.+?)\*\*`)
	reInlineCode   = regexp.MustCompile("`([^`]+)`")
	reInlineItalic = regexp.MustCompile(`\*([^*]+)\*`)
)

// renderInlineMarkdown renders inline Markdown (bold, code, italic) to ANSI.
func renderInlineMarkdown(s string) string {
	// **bold** → bold orange (matching cfg.Strong)
	s = reInlineBold.ReplaceAllStringFunc(s, func(m string) string {
		sub := reInlineBold.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		return "\033[1m\033[38;2;232;145;45m" + sub[1] + "\033[0m"
	})
	// `code` → cyan
	s = reInlineCode.ReplaceAllStringFunc(s, func(m string) string {
		sub := reInlineCode.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		return "\033[38;2;78;201;176m" + sub[1] + "\033[0m"
	})
	// *italic* → italic
	s = reInlineItalic.ReplaceAllStringFunc(s, func(m string) string {
		sub := reInlineItalic.FindStringSubmatch(m)
		if len(sub) < 2 {
			return m
		}
		return "\033[3m" + sub[1] + "\033[0m"
	})
	return s
}

func chromaLexerForFile(filename string) string {
	ext := strings.ToLower(filename[strings.LastIndex(filename, "."):])
	m := map[string]string{
		".go":   "go",
		".js":   "javascript",
		".ts":   "typescript",
		".tsx":  "tsx",
		".jsx":  "jsx",
		".py":   "python",
		".rs":   "rust",
		".sh":   "bash",
		".bash": "bash",
		".ps1":  "powershell",
		".json": "json",
		".yaml": "yaml",
		".yml":  "yaml",
		".toml": "toml",
		".md":   "markdown",
		".html": "html",
		".css":  "css",
		".scss": "scss",
		".c":    "c",
		".cpp":  "cpp",
		".cs":   "csharp",
		".java": "java",
		".rb":   "ruby",
		".php":  "php",
		".lua":  "lua",
		".vim":  "vim",
		".sql":  "sql",
		".xml":  "xml",
		".kt":   "kotlin",
		".swift": "swift",
		".r":    "r",
	}
	if l, ok := m[ext]; ok {
		return l
	}
	return "text"
}
