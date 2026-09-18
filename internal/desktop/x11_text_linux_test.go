//go:build linux

package desktop

import (
	"strings"
	"testing"
)

func TestX11TextBytesKeepsPlainASCIIUnchanged(t *testing.T) {
	input := "Ghost FTP 0.0.7 - Settings"
	if got := string(x11TextBytes(input)); got != input {
		t.Fatalf("x11TextBytes(%q) = %q", input, got)
	}
}

func TestX11TextBytesUsesReadableFallbacksForProductPunctuation(t *testing.T) {
	cases := map[string]string{
		"● DISCONNECTED":       "* DISCONNECTED",
		"Ghost FTP — Settings": "Ghost FTP - Settings",
		"Parallel (1–8)":       "Parallel (1-8)",
		"Upload ↑":             "Upload ^",
		"↓ Download":           "v Download",
		"ready · idle":         "ready | idle",
		"Wait…":                "Wait...",
		"10 • 20":              "10 * 20",
		"−1":                   "-1",
		"A → B ← C":            "A > B < C",
		"Confirm ✓":            "Confirm [x]",
		"Confirm ○":            "Confirm [ ]",
		"Disabled ✕":           "Disabled x",
	}
	for input, want := range cases {
		if got := string(x11TextBytes(input)); got != want {
			t.Errorf("x11TextBytes(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestX11TextBytesUsesReadableEuropeanLocaleFallbacks(t *testing.T) {
	assertFallback := func(input, want string) {
		t.Helper()
		if got := string(x11TextBytes(input)); got != want {
			t.Errorf("x11TextBytes(%q) = %q, want %q", input, got, want)
		}
	}

	assertFallback("Čuvanje žarišta", "Cuvanje zarista")
	assertFallback("Übergrößenträger", "Ubergrossentrager")
	assertFallback("Połączenie i żądanie", "Polaczenie i zadanie")
	assertFallback("Ștergere și înlocuire", "Stergere si inlocuire")
	assertFallback("Bağlantı ölçümü", "Baglanti olcumu")
	assertFallback("Σύνδεση ασφαλείας", "Syndesi asfaleias")
	assertFallback("Подключение безопасно", "Podklyuchenie bezopasno")
	assertFallback("Підключення захищене", "Pidklyuchennya zakhishchene")
}

func TestX11TextBytesKeepsExplicitFallbackForUnsupportedScripts(t *testing.T) {
	if got := string(x11TextBytes("中文")); got != "??" {
		t.Fatalf("unsupported core-font fallback = %q, want ??", got)
	}
}

func TestX11TextBytesPreservesProtocolLengthLimit(t *testing.T) {
	got := x11TextBytes(strings.Repeat("a", 300))
	if len(got) > 240 {
		t.Fatalf("encoded X11 text length = %d, want <= 240", len(got))
	}
}

func TestX11TextBytesDoesNotSplitExpandedFallbackAtLimit(t *testing.T) {
	input := strings.Repeat("a", 239) + "…"
	got := string(x11TextBytes(input))
	if got != strings.Repeat("a", 239) {
		t.Fatalf("unexpected boundary expansion: length=%d suffix=%q", len(got), got[max(0, len(got)-8):])
	}
}
