package api

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestTypedEngineSettingsAndTransferEvents(t *testing.T) {
	dir := t.TempDir()
	engine, err := New(dir, filepath.Join(dir, "GhostFTP.exe"))
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Close()

	settings, err := engine.Settings()
	if err != nil {
		t.Fatal(err)
	}
	if settings.Parallelism < 1 {
		t.Fatalf("invalid defaults: %+v", settings)
	}

	events, seq := engine.TransferEvents(0)
	if seq != 0 || len(events) != 0 {
		t.Fatalf("unexpected initial transfer state: seq=%d events=%d", seq, len(events))
	}
}

func TestNewRemovesLegacyLocalDiagnosticsWithoutFollowingSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "outside.log")
	if err := os.WriteFile(target, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "GhostFTP.log")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	engine, err := New(dir, filepath.Join(dir, "GhostFTP.exe"))
	if err != nil {
		t.Fatal(err)
	}
	engine.Close()
	if _, err := os.Lstat(link); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy log link still exists: %v", err)
	}
	b, err := os.ReadFile(target)
	if err != nil || string(b) != "outside" {
		t.Fatalf("symlink target changed: data=%q err=%v", string(b), err)
	}
}
