package main

import (
	"fmt"
	"strings"
	"testing"
)

func TestDebugSPLWidths(t *testing.T) {
	content := `index IN (1003496)
cf_org_name=DFSPrtnPortalOrg source_type="RTR" cf_app_name="dfsleasecontract-fc-notifier"
| spath cf_app_name
| spath msg
| eval Date=strftime(_time, "%m/%d/%Y %H:%M:%S")
| sort - timestamp
| spath input=msg
| table Date,cf_space_name, ApplicationName,msg.TrouxApplicationName,message`

	out := highlightSPL(content)
	lineMaxW := 70
	lines := splitLines(out)
	var rendered []string
	for _, line := range lines {
		rendered = append(rendered, truncateAnsiLine(line, lineMaxW))
	}
	body := joinLines(rendered)
	box := panelStyle.Width(80).Height(20).Render(body)

	for i, l := range strings.Split(box, "\n") {
		plain := stripANSI(l)
		fmt.Printf("row %2d: visiblewidth=%d :: %q\n", i, len([]rune(plain)), plain)
	}
}

func splitLines(s string) []string {
	var lines []string
	cur := ""
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, cur)
			cur = ""
		} else {
			cur += string(r)
		}
	}
	lines = append(lines, cur)
	return lines
}

func joinLines(lines []string) string {
	s := ""
	for i, l := range lines {
		if i > 0 {
			s += "\n"
		}
		s += l
	}
	return s
}
