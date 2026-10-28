package main

import (
	"strings"
	"unicode"
)

const matchStyleOn = "\x1b[1;38;2;63;185;80m"

// highlightMatches renders every case-insensitive occurrence of query in s
// (which may contain ANSI SGR sequences) in bold green. The original styling
// is restored after each match so the rest of the line keeps its colours.
func highlightMatches(s, query string) string {
	if query == "" || s == "" {
		return s
	}
	qr := []rune(strings.ToLower(query))
	for i, r := range qr {
		qr[i] = unicode.ToLower(r)
	}

	type token struct {
		esc string
		r   rune
	}
	var toks []token
	var vis []rune
	runes := []rune(s)
	for i := 0; i < len(runes); {
		if runes[i] != '\x1b' {
			toks = append(toks, token{r: runes[i]})
			vis = append(vis, unicode.ToLower(runes[i]))
			i++
			continue
		}
		j := i + 1
		if j < len(runes) && runes[j] == '[' {
			j++
			for j < len(runes) && !(runes[j] >= '@' && runes[j] <= '~') {
				j++
			}
		}
		if j < len(runes) {
			j++
		}
		toks = append(toks, token{esc: string(runes[i:j])})
		i = j
	}

	inMatch := make([]bool, len(vis))
	found := false
	for i := 0; i+len(qr) <= len(vis); {
		ok := true
		for k, q := range qr {
			if vis[i+k] != q {
				ok = false
				break
			}
		}
		if ok {
			for k := range qr {
				inMatch[i+k] = true
			}
			found = true
			i += len(qr)
		} else {
			i++
		}
	}
	if !found {
		return s
	}

	var out strings.Builder
	var state []string
	inHL := false
	restore := func() {
		out.WriteString("\x1b[0m")
		for _, e := range state {
			out.WriteString(e)
		}
	}
	vi := 0
	for _, t := range toks {
		if t.esc != "" {
			if strings.HasSuffix(t.esc, "m") && strings.HasPrefix(t.esc, "\x1b[") {
				params := t.esc[2 : len(t.esc)-1]
				if params == "" || params == "0" {
					state = nil
				} else {
					if strings.HasPrefix(params, "0;") {
						state = nil
					}
					state = append(state, t.esc)
				}
				if inHL {
					continue
				}
			}
			out.WriteString(t.esc)
			continue
		}
		m := inMatch[vi]
		vi++
		if m && !inHL {
			out.WriteString(matchStyleOn)
			inHL = true
		} else if !m && inHL {
			restore()
			inHL = false
		}
		out.WriteRune(t.r)
	}
	if inHL {
		restore()
	}
	return out.String()
}
