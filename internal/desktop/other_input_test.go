//go:build !windows

package desktop

import (
	"bufio"
	"strings"
	"testing"
)

func TestPromptPreservesEdgeWhitespace(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("  value with spaces  \n"))
	got, err := prompt(reader, "Value", "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if got != "  value with spaces  " {
		t.Fatalf("prompt normalized raw input: %q", got)
	}
}

func TestPromptUsesFallbackOnlyForEmptyLine(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader("\n"))
	got, err := prompt(reader, "Value", "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}

	reader = bufio.NewReader(strings.NewReader("   \n"))
	got, err = prompt(reader, "Value", "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if got != "   " {
		t.Fatalf("whitespace-only raw input was normalized: %q", got)
	}
}
