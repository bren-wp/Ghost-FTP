//go:build linux

package desktop

import (
	"strings"
	"testing"
)

func TestParseTerminalArgsSupportsQuotedPaths(t *testing.T) {
	tests := []struct {
		line string
		want []string
	}{
		{`get "remote file.txt" "/tmp/local file.txt"`, []string{"get", "remote file.txt", "/tmp/local file.txt"}},
		{`rename 'stari naziv.txt' "novi naziv.txt"`, []string{"rename", "stari naziv.txt", "novi naziv.txt"}},
		{`mkdir "Nova mapa"`, []string{"mkdir", "Nova mapa"}},
		{`put C:\Temp\file.txt remote.txt`, []string{"put", `C:\Temp\file.txt`, "remote.txt"}},
		{`get "a\"b.txt" lokalno.txt`, []string{"get", `a"b.txt`, "lokalno.txt"}},
		{`cd ""`, []string{"cd", ""}},
	}
	for _, tc := range tests {
		got, err := parseTerminalArgs(tc.line)
		if err != nil {
			t.Fatalf("parseTerminalArgs(%q): %v", tc.line, err)
		}
		if len(got) != len(tc.want) {
			t.Fatalf("parseTerminalArgs(%q) = %#v, want %#v", tc.line, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("parseTerminalArgs(%q)[%d] = %q, want %q", tc.line, i, got[i], tc.want[i])
			}
		}
	}
}

func TestParseTerminalArgsRejectsMalformedInput(t *testing.T) {
	invalid := []string{
		`get "nezatvoreno`,
		"mkdir bad\x00name",
		"ls foo\rbar",
		strings.Repeat("a", maxTerminalCommandLength+1),
		strings.Repeat("x ", maxTerminalArguments+1),
	}
	for _, line := range invalid {
		if _, err := parseTerminalArgs(line); err == nil {
			t.Fatalf("parseTerminalArgs(%q) unexpectedly succeeded", line)
		}
	}
}

func TestParseTerminalArgsEmptyLine(t *testing.T) {
	got, err := parseTerminalArgs("  \t  \n")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected no arguments, got %#v", got)
	}
}
