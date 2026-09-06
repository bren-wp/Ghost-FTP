package remote

import (
	"path/filepath"
	"testing"
)

func TestWindowsOpenSSHCandidatesUseNativeSystemDirectory(t *testing.T) {
	systemDir := filepath.Join("C:", "Windows", "System32")
	got := windowsOpenSSHCandidates(systemDir, "amd64", "sftp.exe")
	if len(got) != 1 || got[0] != filepath.Join(systemDir, "OpenSSH", "sftp.exe") {
		t.Fatalf("unexpected candidates: %#v", got)
	}
}

func TestWindowsOpenSSHCandidatesAddSysnativeForWOW64X86(t *testing.T) {
	windowsDir := filepath.Join("C:", "Windows")
	systemDir := filepath.Join(windowsDir, "SysWOW64")
	got := windowsOpenSSHCandidates(systemDir, "386", "ssh.exe")
	if len(got) != 2 {
		t.Fatalf("candidate count=%d want 2: %#v", len(got), got)
	}
	want := filepath.Join(windowsDir, "Sysnative", "OpenSSH", "ssh.exe")
	if got[1] != want {
		t.Fatalf("Sysnative candidate=%q want %q", got[1], want)
	}
}
