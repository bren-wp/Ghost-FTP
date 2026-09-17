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
