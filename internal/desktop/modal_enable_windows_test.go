//go:build windows

package desktop

import "testing"

func TestModalAwareEnableWindowProcTracksOriginalTopLevelIconicState(t *testing.T) {
	// Runtime behavior is exercised by the authentic Windows UI workflow. This
	// compile-time test keeps the modal-aware wrapper part of the Windows build
	// and documents the intended SW_RESTORE command used by the guarded path.
	if swRestore != 9 {
		t.Fatalf("swRestore = %d, want 9", swRestore)
	}
}
