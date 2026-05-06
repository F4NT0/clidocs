package main

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	chromaStyles "github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
	glamourStyles "github.com/charmbracelet/glamour/styles"
)

func highlightCode(content, filename string) string {
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
// Math / superscript helpers
// ---------------------------------------------------------------------------

// reMathInline matches $...$ and $$...$$ LaTeX delimiters.
var reMathInline = regexp.MustCompile(`\$\$?([^$\n]+?)\$?\$`)

// reSuper matches base^exp patterns like x^2, n^{k+1}
var reSuper = regexp.MustCompile(`(\w)\^([{(]?[\w\d+\-*/=]+[)}]?)`)

var superMap = map[rune]string{
	'0': "⁰", '1': "¹", '2': "²", '3': "³", '4': "⁴",
	'5': "⁵", '6': "⁶", '7': "⁷", '8': "⁸", '9': "⁹",
	'+': "⁺", '-': "⁻", '=': "⁼", 'n': "ⁿ", 'i': "ⁱ",
	'a': "ᵃ", 'b': "ᵇ", 'c': "ᶜ", 'd': "ᵈ", 'e': "ᵉ",
	'f': "ᶠ", 'g': "ᵍ", 'h': "ʰ", 'j': "ʲ", 'k': "ᵏ",
	'l': "ˡ", 'm': "ᵐ", 'o': "ᵒ", 'p': "ᵖ", 'r': "ʳ",
	's': "ˢ", 't': "ᵗ", 'u': "ᵘ", 'v': "ᵛ", 'w': "ʷ",
	'x': "ˣ", 'y': "ʸ", 'z': "ᶻ",
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

func strPtr(s string) *string { return &s }

// renderMarkdown renders Markdown using glamour with a patched dark style:
//   - fixes red squares on JSON code blocks (Error/Punctuation tokens → neutral)
//   - adds proper table grid lines (CenterSeparator, ColumnSeparator, RowSeparator)
func renderMarkdown(content string, width int) string {
	if width < 40 {
		width = 40
	}
	content = preprocessMath(content)

	// Start from the built-in dark config and patch only what we need.
	cfg := glamourStyles.DarkStyleConfig

	// Fix 1: JSON red squares — Error and Punctuation chroma tokens inherit
	// a red colour in the dark theme. Override them to plain foreground.
	if cfg.CodeBlock.Chroma != nil {
		neutral := strPtr("#e6edf3")
		cfg.CodeBlock.Chroma.Error = ansi.StylePrimitive{Color: neutral, BackgroundColor: strPtr("")}
		cfg.CodeBlock.Chroma.Punctuation = ansi.StylePrimitive{Color: neutral}
	}

	// Bold — orange.
	boldTrue := true
	upperTrue := true
	cfg.Strong = ansi.StylePrimitive{
		Color: strPtr("#e8912d"),
		Bold:  &boldTrue,
	}

	// H1 — keep glamour dark default (no change).

	// H2 — one arrow, uppercase, light-blue.
	cfg.H2 = ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
		Prefix: "\n→ ", Color: strPtr("#79c0ff"), Bold: &boldTrue, Upper: &upperTrue,
	}}
	// H3 — one arrow, uppercase, orange.
	cfg.H3 = ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
		Prefix: "\n→ ", Color: strPtr("#e8912d"), Bold: &boldTrue, Upper: &upperTrue,
	}}
	// H4 — one arrow, green.
	cfg.H4 = ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
		Prefix: "→ ", Color: strPtr("#3fb950"), Bold: &boldTrue, Upper: &upperTrue,
	}}
	// H5/H6 — one arrow, muted.
	cfg.H5 = ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
		Prefix: "→ ", Color: strPtr("#a5d6ff"), Upper: &upperTrue,
	}}
	cfg.H6 = ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{
		Prefix: "→ ", Color: strPtr("#6e7681"), Upper: &upperTrue,
	}}

	// Table — inner separators only; outer box is added by wrapTableBoxes below.
	cfg.Table = ansi.StyleTable{
		StyleBlock:      cfg.Table.StyleBlock,
		CenterSeparator: strPtr("┼"),
		ColumnSeparator: strPtr("│"),
		RowSeparator:    strPtr("─"),
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
	return strings.TrimRight(out, "\n")
}
