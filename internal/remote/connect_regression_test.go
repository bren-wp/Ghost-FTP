package remote

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSSHSessionConfigNormalizesBracketedIPv6Host(t *testing.T) {
	dir := t.TempDir()
	known := filepath.Join(dir, "known_hosts")
	if err := os.WriteFile(known, []byte("host ssh-ed25519 key\n"), 0600); err != nil {
		t.Fatal(err)
	}
	config, _, err := createSSHSessionConfig(dir, "[2001:db8::25]", 22, "user", "", known, "ssh-ed25519", 15)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(config)
	data, err := os.ReadFile(config)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "HostName 2001:db8::25") {
		t.Fatalf("bracketed IPv6 was not normalized for OpenSSH: %q", text)
	}
	if strings.Contains(text, "HostName [2001:db8::25]") {
		t.Fatalf("OpenSSH config retained URL-style IPv6 brackets: %q", text)
	}
}

func TestFindOpenSSHUsesNativeExecutableNameOutsideWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non-Windows executable-name regression")
	}

	withWindowsSuffix, err := findOpenSSH("sftp.exe")
	if err != nil {
		t.Skipf("trusted native sftp is unavailable: %v", err)
	}
	native, err := findOpenSSH("sftp")
	if err != nil {
		t.Fatal(err)
	}
	if withWindowsSuffix != native {
		t.Fatalf("findOpenSSH(sftp.exe)=%q want same trusted native executable as findOpenSSH(sftp)=%q", withWindowsSuffix, native)
	}
}
