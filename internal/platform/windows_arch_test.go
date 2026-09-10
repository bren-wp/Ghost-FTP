package platform

import "testing"

func TestWindowsArchitectureFromProcessor(t *testing.T) {
	for _, tc := range []struct {
		code uint16
		want string
	}{
		{windowsProcessorArchitectureIntel, "x86"},
		{windowsProcessorArchitectureAMD64, "x64"},
	} {
		got, err := windowsArchitectureFromProcessor(tc.code)
		if err != nil {
			t.Fatalf("code %d: %v", tc.code, err)
		}
		if got != tc.want {
			t.Fatalf("code %d: got %q, want %q", tc.code, got, tc.want)
		}
	}
}

func TestWindowsArchitectureFromProcessorRejectsUnsupported(t *testing.T) {
	if _, err := windowsArchitectureFromProcessor(12); err == nil {
		t.Fatal("expected ARM64/unsupported architecture to be rejected")
	}
}
