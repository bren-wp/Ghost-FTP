package remote

import "testing"

func TestSFTPQuoteEscapesGlobMetacharacters(t *testing.T) {
	got := sftpQuote(`folder/a[*]?b.txt`)
	want := `"folder/a\[\*\]\?b.txt"`
	if got != want {
		t.Fatalf("sftpQuote() = %q, want %q", got, want)
	}
}

func TestSFTPQuoteDisambiguatesLeadingDashOperand(t *testing.T) {
	got := sftpQuote(`-R`)
	want := `"./-R"`
	if got != want {
		t.Fatalf("sftpQuote() = %q, want %q", got, want)
	}
}

func TestSFTPQuotePreservesLiteralQuoteAndBackslash(t *testing.T) {
	got := sftpQuote(`folder/a\b"c.txt`)
	want := `"folder/a\\b\"c.txt"`
	if got != want {
		t.Fatalf("sftpQuote() = %q, want %q", got, want)
	}
}
