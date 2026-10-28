package main

import (
	"strings"
	"testing"
)

func TestHighlightMatches(t *testing.T) {
	got := highlightMatches("\x1b[31mMARCO ANTONIO\x1b[0m x", "marco")
	want := "\x1b[31m" + matchStyleOn + "MARCO" + "\x1b[0m\x1b[31m" + " ANTONIO\x1b[0m x"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if stripAnsi(got) != "MARCO ANTONIO x" {
		t.Fatalf("visible text changed: %q", stripAnsi(got))
	}
	split := highlightMatches("MA\x1b[32mRC\x1b[0mO", "marco")
	if !strings.HasPrefix(split, matchStyleOn) || stripAnsi(split) != "MARCO" {
		t.Fatalf("split match not highlighted: %q", split)
	}
	if highlightMatches("nothing", "zzz") != "nothing" {
		t.Fatal("no-match must be unchanged")
	}
}
