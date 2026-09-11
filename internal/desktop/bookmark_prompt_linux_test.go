//go:build linux

package desktop

import "testing"

func TestLinuxBookmarkNamePromptClassification(t *testing.T) {
	u := &linuxDesktop{}
	for _, kind := range []int{linuxPromptBookmarkLocalName, linuxPromptBookmarkRemoteName} {
		u.promptKind = kind
		if !u.bookmarkNamePrompt() {
			t.Fatalf("prompt kind %d must be treated as a bookmark naming child", kind)
		}
	}
	for _, kind := range []int{linuxPromptNone, linuxPromptBookmarkManager, linuxPromptLocalRename} {
		u.promptKind = kind
		if u.bookmarkNamePrompt() {
			t.Fatalf("prompt kind %d must not be treated as a bookmark naming child", kind)
		}
	}
}
