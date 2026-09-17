//go:build linux

package desktop

import (
	"testing"
	"unicode/utf8"
)

func TestLinuxTrimForUIIsRuneAware(t *testing.T) {
	got := linuxTrimForUI("Čuvanje postavki", 8)
	if got != "Čuvan..." {
		t.Fatalf("trimmed UTF-8 label = %q, want %q", got, "Čuvan...")
	}
	if !utf8.ValidString(got) {
		t.Fatalf("trimmed UTF-8 label is invalid: %q", got)
	}

	got = linuxTrimForUI("简体中文设置", 4)
	if got != "简..." {
		t.Fatalf("trimmed CJK label = %q, want %q", got, "简...")
	}
	if !utf8.ValidString(got) {
		t.Fatalf("trimmed CJK label is invalid: %q", got)
	}
}

func TestLinuxButtonLabelLimitTracksControlWidth(t *testing.T) {
	rail := linuxRectWH(0, 0, 166, 30)
	wide := linuxRectWH(0, 0, 280, 30)
	if got := linuxButtonLabelLimit(rail); got != 25 {
		t.Fatalf("rail button label limit = %d, want 25", got)
	}
	if got := linuxButtonLabelLimit(wide); got != 44 {
		t.Fatalf("wide button label limit = %d, want 44", got)
	}
	if linuxButtonLabelLimit(wide) <= linuxButtonLabelLimit(rail) {
		t.Fatal("wide button did not receive a larger label budget")
	}
}

func TestLinuxKeysymTextAcceptsLatin1AndUnicodeKeysyms(t *testing.T) {
	cases := map[uint32]string{
		'a':        "a",
		0x00e9:     "é",
		0x0100010d: "č",
		0x01000416: "Ж",
	}
	for sym, want := range cases {
		got, ok := linuxKeysymText(sym)
		if !ok || got != want {
			t.Errorf("linuxKeysymText(%#x) = %q, %v; want %q, true", sym, got, ok, want)
		}
	}
	for _, sym := range []uint32{x11KeyLeft, x11KeyEscape, 0x0100000a, 0x01110000} {
		if got, ok := linuxKeysymText(sym); ok {
			t.Errorf("linuxKeysymText(%#x) unexpectedly accepted %q", sym, got)
		}
	}
}

func TestLinuxFieldTextEditingPreservesUTF8Boundaries(t *testing.T) {
	if got := linuxDeleteLastRune("server-č"); got != "server-" {
		t.Fatalf("delete last Croatian rune = %q", got)
	}
	if got := linuxDeleteLastRune("路径"); got != "路" {
		t.Fatalf("delete last CJK rune = %q", got)
	}
	if got := linuxTruncateUTF8Bytes("abcč", 4); got != "abc" {
		t.Fatalf("UTF-8 truncation = %q, want abc", got)
	}
	if got := linuxTruncateUTF8Bytes("čč", 3); got != "č" {
		t.Fatalf("UTF-8 truncation = %q, want č", got)
	}
}

