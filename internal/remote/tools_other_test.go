//go:build linux

package remote

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFakeTransportExecutable(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
}

func firstTrustedSystemTransport(name string) (string, bool) {
	for _, dir := range linuxTransportSystemDirs {
		if resolved, ok := trustedLinuxTransportExecutable(filepath.Join(dir, name)); ok {
			return resolved, true
		}
	}
	return "", false
}

func TestFindCurlRejectsPATHShadowing(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "curl")
	writeFakeTransportExecutable(t, fake)
	t.Setenv("PATH", dir)

	got, err := findCurl()
	if got == fake {
		t.Fatalf("findCurl accepted user-controlled PATH executable %q", got)
	}
	if err == nil {
		if resolved, ok := trustedLinuxTransportExecutable(got); !ok || resolved != got {
			t.Fatalf("findCurl returned untrusted executable %q", got)
		}
	}
}

func TestFindCurlFallsBackToTrustedSystemBinary(t *testing.T) {
	expected, ok := firstTrustedSystemTransport("curl")
	if !ok {
		t.Skip("no trusted system curl is installed")
	}
	dir := t.TempDir()
	writeFakeTransportExecutable(t, filepath.Join(dir, "curl"))
	t.Setenv("PATH", dir)

	got, err := findCurl()
	if err != nil {
		t.Fatal(err)
	}
	if got != expected {
		t.Fatalf("findCurl()=%q want trusted system binary %q", got, expected)
	}
}

func TestFindOpenSSHRejectsPATHShadowing(t *testing.T) {
	dir := t.TempDir()
	names := []string{"ssh", "sftp", "ssh-keyscan"}
	for _, name := range names {
		writeFakeTransportExecutable(t, filepath.Join(dir, name))
	}
	t.Setenv("PATH", dir)

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			fake := filepath.Join(dir, name)
			expected, haveSystem := firstTrustedSystemTransport(name)
			got, err := findOpenSSH(name + ".exe")
			if got == fake {
				t.Fatalf("findOpenSSH accepted user-controlled PATH executable %q", got)
			}
			if haveSystem {
				if err != nil {
					t.Fatal(err)
				}
				if got != expected {
					t.Fatalf("findOpenSSH()=%q want trusted system binary %q", got, expected)
				}
				return
			}
			if err == nil {
				t.Fatalf("findOpenSSH unexpectedly resolved %q without a trusted system candidate", got)
			}
		})
	}
}

func TestTrustedLinuxTransportRejectsUserControlledSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "unsafe-curl")
	writeFakeTransportExecutable(t, target)
	link := filepath.Join(dir, "curl")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if resolved, ok := trustedLinuxTransportExecutable(link); ok {
		t.Fatalf("user-controlled symlink unexpectedly trusted as %q", resolved)
	}

	if systemCurl, ok := firstTrustedSystemTransport("curl"); ok {
		systemLink := filepath.Join(dir, "system-curl")
		if err := os.Symlink(systemCurl, systemLink); err != nil {
			t.Fatal(err)
		}
		if resolved, ok := trustedLinuxTransportExecutable(systemLink); ok {
			t.Fatalf("user-controlled symlink to trusted target unexpectedly trusted as %q", resolved)
		}
	}
}

func TestTrustedLinuxTransportAcceptsSystemSymlinkChain(t *testing.T) {
	candidate := "/bin/sh"
	info, err := os.Lstat(candidate)
	if err != nil {
		t.Skip("/bin/sh is unavailable")
	}
	binInfo, binErr := os.Lstat("/bin")
	if info.Mode()&os.ModeSymlink == 0 && (binErr != nil || binInfo.Mode()&os.ModeSymlink == 0) {
		t.Skip("system does not expose a symlink chain through /bin/sh")
	}
	resolved, ok := trustedLinuxTransportExecutable(candidate)
	if !ok {
		t.Fatalf("legitimate system symlink chain %q was rejected", candidate)
	}
	if !filepath.IsAbs(resolved) {
		t.Fatalf("resolved system executable is not absolute: %q", resolved)
	}
}

func TestTrustedLinuxDirectoryPermissionBoundary(t *testing.T) {
	if trustedLinuxMetadata(0, os.ModeDir|0775, true) {
		t.Fatal("group-writable root-owned directory was trusted")
	}
	if trustedLinuxMetadata(0, os.ModeDir|0757, true) {
		t.Fatal("other-writable root-owned directory was trusted")
	}
	if !trustedLinuxMetadata(0, os.ModeDir|0755, true) {
		t.Fatal("root-owned non-writable directory was rejected")
	}
	if trustedLinuxDirectoryChain("/tmp", 0) {
		t.Fatal("world-writable /tmp unexpectedly passed trusted directory-chain validation")
	}
}

func TestTrustedLinuxTransportRejectsNonRegularFile(t *testing.T) {
	if trustedLinuxMetadata(0, os.ModeNamedPipe|0755, false) {
		t.Fatal("non-regular executable metadata was trusted")
	}
	if trustedLinuxMetadata(0, os.ModeDir|0755, false) {
		t.Fatal("directory executable metadata was trusted")
	}
}

func TestTrustedLinuxTransportRequiresExecutablePermission(t *testing.T) {
	if trustedLinuxMetadata(0, 0644, false) {
		t.Fatal("non-executable regular file was trusted")
	}
	if !trustedLinuxMetadata(0, 0755, false) {
		t.Fatal("root-owned executable regular file was rejected")
	}
}

func TestTrustedLinuxTransportRequiresRootOwnership(t *testing.T) {
	if trustedLinuxMetadata(1000, 0755, false) {
		t.Fatal("non-root-owned executable was trusted")
	}
}
