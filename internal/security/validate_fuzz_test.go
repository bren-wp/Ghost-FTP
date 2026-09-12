package security

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func FuzzSecurityValidators(f *testing.F) {
	for _, seed := range []string{
		"example.com",
		"[2001:db8::1]",
		"2001:db8::1",
		"public_html/file.txt",
		"../escape",
		"report.txt",
		"CON",
		"secret-value",
		"line\r\nbreak",
		"\x00",
		"",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {
		if err := ValidateRemotePath(value); err == nil {
			if len(value) > 4096 || !utf8.ValidString(value) || strings.ContainsAny(value, "\x00\r\n") {
				t.Fatalf("ValidateRemotePath accepted structurally forbidden input %q", value)
			}
			for _, part := range strings.Split(strings.ReplaceAll(value, "\\", "/"), "/") {
				if part == ".." {
					t.Fatalf("ValidateRemotePath accepted traversal input %q", value)
				}
			}
		}

		if err := ValidateRemoteName(value); err == nil {
			if value == "" || value == "." || value == ".." || len(value) > 1024 || !utf8.ValidString(value) || strings.ContainsAny(value, "\x00\r\n/\\") {
				t.Fatalf("ValidateRemoteName accepted structurally forbidden input %q", value)
			}
		}

		if err := ValidateName(value); err == nil {
			if value == "" || value != strings.TrimSpace(value) || value == "." || value == ".." || len(value) > 255 || !utf8.ValidString(value) || !safeName.MatchString(value) || strings.HasSuffix(value, ".") || strings.HasSuffix(value, " ") {
				t.Fatalf("ValidateName accepted structurally forbidden input %q", value)
			}
		}

		if err := ValidateHost(value); err == nil {
			if value == "" || value != strings.TrimSpace(value) || len(value) > 253 || !utf8.ValidString(value) || strings.ContainsAny(value, "\x00\r\n\t /\\@") {
				t.Fatalf("ValidateHost accepted structurally forbidden input %q", value)
			}
		}

		if err := ValidateSecret(value); err == nil {
			if len(value) > 8192 || !utf8.ValidString(value) || strings.ContainsAny(value, "\x00\r\n") {
				t.Fatalf("ValidateSecret accepted structurally forbidden input %q", value)
			}
		}
	})
}
