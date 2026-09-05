//go:build linux

package config

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinuxProfileStorageTightensOwnedLeafDirectory(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "ghostftp-state")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}

	plain := []byte(`[{"name":"Example","host":"ftp.example.test"}]`)
	encoded, err := protectProfileData(plain, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(encoded, linuxProfilePrefix) {
		t.Fatalf("secure envelope does not use Linux AES-GCM prefix: %q", encoded)
	}

	info, err := os.Lstat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0700 {
		t.Fatalf("profile storage directory permissions=%#o want 0700", got)
	}
	keyInfo, err := os.Lstat(filepath.Join(dir, linuxProfileKeyFile))
	if err != nil {
		t.Fatal(err)
	}
	if got := keyInfo.Mode().Perm(); got != 0600 {
		t.Fatalf("profile key permissions=%#o want 0600", got)
	}
	if keyInfo.Size() != linuxProfileKeySize {
		t.Fatalf("profile key size=%d want %d", keyInfo.Size(), linuxProfileKeySize)
	}

	decoded, err := unprotectProfileData(encoded, dir)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != string(plain) {
		t.Fatalf("round-trip mismatch: %q", decoded)
	}
}

func TestLinuxProfileStorageRejectsSymlinkDirectory(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "target")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "profile-link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := protectProfileData([]byte("profile"), link); err == nil {
		t.Fatal("symlink profile directory was accepted")
	}
}

func TestLinuxProfileStorageRejectsSymlinkKey(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "state")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(base, "outside.key")
	if err := os.WriteFile(target, make([]byte, linuxProfileKeySize), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, linuxProfileKeyFile)); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := protectProfileData([]byte("profile"), dir); err == nil {
		t.Fatal("symlink profile key was accepted")
	}
}

func TestLinuxProfileEnvelopeRejectsTampering(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	encoded, err := protectProfileData([]byte("authenticated profile metadata"), dir)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(encoded, linuxProfilePrefix))
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) < 2 {
		t.Fatal("unexpectedly short AES-GCM envelope")
	}
	raw[len(raw)-1] ^= 0x01
	tampered := linuxProfilePrefix + base64.StdEncoding.EncodeToString(raw)
	if _, err := unprotectProfileData(tampered, dir); err == nil {
		t.Fatal("tampered AES-GCM profile envelope was accepted")
	}
}

func TestLinuxLegacyProfileEnvelopeRequiresMigration(t *testing.T) {
	legacy := base64.StdEncoding.EncodeToString([]byte(`[]`))
	if !profileDataNeedsMigration(legacy) {
		t.Fatal("legacy Base64 profile envelope was not marked for migration")
	}
	if profileDataNeedsMigration(linuxProfilePrefix + "payload") {
		t.Fatal("current AES-GCM profile envelope was incorrectly marked for migration")
	}
}
