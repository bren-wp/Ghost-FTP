//go:build !windows

package remote

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewManagerDoesNotDeleteOtherUnixSFTPSessionArtifacts(t *testing.T) {
	dataDir := t.TempDir()
	knownHostsDir := filepath.Join(dataDir, "known_hosts")
	if err := os.MkdirAll(knownHostsDir, 0700); err != nil {
		t.Fatal(err)
	}
	active := filepath.Join(knownHostsDir, ".GhostFTP-sftp-active.conf")
	if err := os.WriteFile(active, []byte("active-session"), 0600); err != nil {
		t.Fatal(err)
	}
	_ = NewManager(nil, nil, dataDir, "/tmp/GhostFTP")
	got, err := os.ReadFile(active)
	if err != nil {
		t.Fatalf("new Unix manager removed another session artifact: %v", err)
	}
	if string(got) != "active-session" {
		t.Fatalf("session artifact changed: %q", got)
	}
}
