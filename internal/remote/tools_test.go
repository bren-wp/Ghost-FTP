package remote

import (
	"path/filepath"
	"testing"
)

func TestWindowsCurlCandidatesIncludeSystemCurl(t *testing.T) {
	systemDir := filepath.Join("C:", "Windows", "System32")
	got := windowsCurlCandidates(systemDir, "amd64")
	if len(got) != 1 {
		t.Fatalf("candidate count=%d want 1: %#v", len(got), got)
	}
	want := filepath.Join(systemDir, "curl.exe")
	if got[0] != want {
		t.Fatalf("first candidate=%q want %q", got[0], want)
	}
}

func TestWindowsCurlCandidatesAddSysnativeForWOW64X86(t *testing.T) {
	windowsDir := filepath.Join("C:", "Windows")
	systemDir := filepath.Join(windowsDir, "SysWOW64")
	got := windowsCurlCandidates(systemDir, "386")
	if len(got) != 2 {
		t.Fatalf("candidate count=%d want 2: %#v", len(got), got)
	}
	wantNative := filepath.Join(windowsDir, "Sysnative", "curl.exe")
	if got[1] != wantNative {
		t.Fatalf("Sysnative candidate=%q want %q", got[1], wantNative)
	}
}

func TestWindowsCurlCandidatesDoNotUseSysnativeForNativeBuilds(t *testing.T) {
	systemDir := filepath.Join("C:", "Windows", "SysWOW64")
	for _, arch := range []string{"amd64", "arm64"} {
		got := windowsCurlCandidates(systemDir, arch)
		if len(got) != 1 {
			t.Fatalf("arch=%s candidate count=%d want 1: %#v", arch, len(got), got)
		}
	}
}
