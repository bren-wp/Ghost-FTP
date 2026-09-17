//go:build windows

package desktop

import "testing"

func TestWindowsSystemCommandCloseIsDistinctFromMinimize(t *testing.T) {
	if !isCloseSystemCommand(scClose) {
		t.Fatal("SC_CLOSE must be recognized as a close command")
	}
	if !isCloseSystemCommand(scClose | 0x0003) {
		t.Fatal("SC_CLOSE low-bit invocation metadata must be ignored")
	}
	if isCloseSystemCommand(scMinimize) {
		t.Fatal("SC_MINIMIZE must never be interpreted as close")
	}
	if normalizedSystemCommand(scMinimize|0x0007) != scMinimize {
		t.Fatal("system-command normalization must preserve SC_MINIMIZE")
	}
}
