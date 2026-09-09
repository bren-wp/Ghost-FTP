package desktop

import (
	"errors"
	"testing"
)

func TestRemoteEditBufferPreservesUniformNewlines(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		edit string
		want string
	}{
		{name: "lf", raw: "a\nb\n", edit: "a\nc\n", want: "a\nc\n"},
		{name: "crlf", raw: "a\r\nb\r\n", edit: "a\nc\n", want: "a\r\nc\r\n"},
		{name: "cr", raw: "a\rb\r", edit: "a\nc\n", want: "a\rc\r"},
		{name: "none", raw: "single line", edit: "single line changed", want: "single line changed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buffer, err := newRemoteEditBuffer(tt.raw)
			if err != nil {
				t.Fatalf("newRemoteEditBuffer: %v", err)
			}
			if got := buffer.Encode(tt.edit); got != tt.want {
				t.Fatalf("Encode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRemoteEditBufferRejectsMixedNewlines(t *testing.T) {
	for _, raw := range []string{"a\r\nb\nc", "a\r\nb\rc", "a\nb\rc"} {
		if _, err := newRemoteEditBuffer(raw); !errors.Is(err, errRemoteEditMixedNewlines) {
			t.Fatalf("newRemoteEditBuffer(%q) error = %v, want mixed-newline refusal", raw, err)
		}
	}
}

func TestNormalizeRemoteEditorTextCanonicalizesControlNewlines(t *testing.T) {
	if got := normalizeRemoteEditorText("a\r\nb\rc\n"); got != "a\nb\nc\n" {
		t.Fatalf("normalizeRemoteEditorText() = %q", got)
	}
}
