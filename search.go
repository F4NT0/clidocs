package main

import (
	"path/filepath"
	"strings"
)

// matchName returns true when name matches the glob/substring pattern.
// Supports:  *.go  (glob),  main.go  (exact),  docker  (substring).
func matchName(name, pattern string) bool {
	lower := strings.ToLower(name)
	pat := strings.ToLower(pattern)
	if strings.ContainsAny(pat, "*?[") {
		matched, _ := filepath.Match(pat, lower)
		return matched
	}
	return strings.Contains(lower, pat)
}
