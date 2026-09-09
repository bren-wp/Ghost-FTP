//go:build linux

package desktop

import "testing"

func TestRemoteEditClampCursorNeverSplitsUTF8(t *testing.T) {
	text := "až日b"
	for cursor := 0; cursor <= len(text); cursor++ {
		got := remoteEditClampCursor(text, cursor)
		if got < 0 || got > len(text) {
			t.Fatalf("cursor %d clamped outside text: %d", cursor, got)
		}
		if got < len(text) && got > 0 && text[got]&0xc0 == 0x80 {
			t.Fatalf("cursor %d clamped into UTF-8 continuation byte at %d", cursor, got)
		}
	}
}

func TestRemoteEditLineBoundsAndColumns(t *testing.T) {
	text := "first\nžuta日\nlast"
	start, end, ok := remoteEditLineBounds(text, 1)
	if !ok || text[start:end] != "žuta日" {
		t.Fatalf("line bounds = %d:%d %v %q", start, end, ok, text[start:end])
	}
	line := text[start:end]
	if got := remoteEditByteAtColumn(line, 1); got != len("ž") {
		t.Fatalf("column 1 byte offset = %d, want %d", got, len("ž"))
	}
	if got := remoteEditByteAtColumn(line, 5); got != len(line) {
		t.Fatalf("end column byte offset = %d, want %d", got, len(line))
	}
}

func TestRemoteEditCursorLineColumn(t *testing.T) {
	text := "one\nž日x\nthree"
	cursor := len("one\nž日")
	line, column, start := remoteEditCursorLineColumn(text, cursor)
	if line != 1 || column != 2 || start != len("one\n") {
		t.Fatalf("line/column/start = %d/%d/%d", line, column, start)
	}
}

func TestLinuxRemoteEditKeysymTextSupportsUnicode(t *testing.T) {
	if got, ok := linuxRemoteEditKeysymText(0x0100017e); !ok || got != "ž" {
		t.Fatalf("Unicode keysym = %q, %v", got, ok)
	}
	if got, ok := linuxRemoteEditKeysymText('A'); !ok || got != "A" {
		t.Fatalf("ASCII keysym = %q, %v", got, ok)
	}
	if _, ok := linuxRemoteEditKeysymText(x11KeyEscape); ok {
		t.Fatal("Escape must not be emitted as editor text")
	}
}

func TestRemoteEditVisibleSegmentUsesRuneColumns(t *testing.T) {
	if got := remoteEditVisibleSegment("až日bc", 1, 3); got != "ž日b" {
		t.Fatalf("visible segment = %q", got)
	}
}
