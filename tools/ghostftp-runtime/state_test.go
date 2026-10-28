package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func isolatedFallbackPath(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	t.Setenv("APPDATA", root)
	path, err := fallbackStatePath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

func fallbackTestState(locale string) persistedState {
	return persistedState{
		Locale: locale, Sites: json.RawMessage(`[{"name":"Production"}]`),
		Settings: json.RawMessage(`{"theme":"dark"}`),
	}
}

func TestFallbackStatePreservesPreviousVersionAndRecovers(t *testing.T) {
	path := isolatedFallbackPath(t)
	if err := writePersistedState(fallbackTestState("en")); err != nil {
		t.Fatal(err)
	}
	if err := writePersistedState(fallbackTestState("hr")); err != nil {
		t.Fatal(err)
	}
	beforeFailure, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatal(err)
	}
	var previous persistedState
	if err := json.Unmarshal(beforeFailure, &previous); err != nil {
		t.Fatal(err)
	}
	if previous.Locale != "en" {
		t.Fatalf("backup lost former state: %s", previous.Locale)
	}
	if err := os.WriteFile(path, []byte("truncated-write"), 0600); err != nil {
		t.Fatal(err)
	}
	recovered, err := readPersistedState()
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Locale != "en" {
		t.Fatalf("expected verified backup recovery, got %s", recovered.Locale)
	}
}

func TestFallbackStateRejectsNonRegularDestinationWithoutDeletingBackup(t *testing.T) {
	path := isolatedFallbackPath(t)
	backup := []byte(`{"locale":"en","sites":[],"settings":{}}`)
	if err := os.WriteFile(path+".bak", backup, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := writePersistedState(fallbackTestState("hr")); err == nil {
		t.Fatal("directory masquerading as state should be rejected")
	}
	after, err := os.ReadFile(path + ".bak")
	if err != nil || string(after) != string(backup) {
		t.Fatalf("failed write changed good backup: %v", err)
	}
}

func TestFallbackStateNeverBacksUpCorruptCurrentOverGoodBackup(t *testing.T) {
	path := isolatedFallbackPath(t)
	backup := []byte(`{"locale":"en","sites":[],"settings":{}}`)
	if err := os.WriteFile(path+".bak", backup, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := writePersistedState(fallbackTestState("hr")); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(path + ".bak")
	if err != nil || string(after) != string(backup) {
		t.Fatalf("corrupt original displaced known-good backup: %v", err)
	}
	current, err := readPersistedState()
	if err != nil || current.Locale != "hr" {
		t.Fatalf("valid new state not persisted: %v", err)
	}
}

func TestFallbackStateUnrelatedPredictableTmpNeverTruncated(t *testing.T) {
	path := isolatedFallbackPath(t)
	sentinel := filepath.Join(t.TempDir(), "sentinel")
	if err := os.WriteFile(sentinel, []byte("keep me"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sentinel, path+".tmp"); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("Windows symlink privilege unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if err := writePersistedState(fallbackTestState("hr")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(sentinel)
	if err != nil || string(data) != "keep me" {
		t.Fatalf("fixed temporary path could corrupt unrelated data: %v", err)
	}
}

func TestFallbackStateRejectsSymlinkDestination(t *testing.T) {
	path := isolatedFallbackPath(t)
	sentinel := filepath.Join(t.TempDir(), "sentinel")
	if err := os.WriteFile(sentinel, []byte("keep me"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sentinel, path); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("Windows symlink privilege unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if err := writePersistedState(fallbackTestState("hr")); err == nil {
		t.Fatal("a symlink destination must be refused")
	}
	data, err := os.ReadFile(sentinel)
	if err != nil || string(data) != "keep me" {
		t.Fatalf("state write followed unsafe symlink: %v", err)
	}
}

// A corrupt backup on an otherwise missing primary must not masquerade as a
// first launch. A user may have had valid sites before an interrupted update.
func TestMissingFallbackPrimaryWithCorruptBackupFailsClosed(t *testing.T) {
	path := isolatedFallbackPath(t)
	broken := []byte("{truncated-backup")
	if err := os.WriteFile(path+".bak", broken, 0600); err != nil {
		t.Fatal(err)
	}
	if state, err := readPersistedState(); err == nil {
		t.Fatalf("silent empty-state reset with corrupt backup: %+v", state)
	}
	preserved, err := os.ReadFile(path + ".bak")
	if err != nil || string(preserved) != string(broken) {
		t.Fatalf("recovery failed to preserve original bytes: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("failed recovery unexpectedly created a primary state: %v", err)
	}
}

func TestFreshFallbackInstallInitializesOnlyWhenBothFilesAbsent(t *testing.T) {
	path := isolatedFallbackPath(t)
	state, err := readPersistedState()
	if err != nil {
		t.Fatal(err)
	}
	if state.Locale != "en" || string(state.Sites) != "[]" || string(state.Settings) != "{}" {
		t.Fatalf("unexpected first-launch defaults: %+v", state)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("reading defaults should not create persisted data: %v", err)
	}
}

func TestInvalidFallbackStateShapeRecoversBackupRatherThanClearingSites(t *testing.T) {
	path := isolatedFallbackPath(t)
	backup := []byte(`{"locale":"hr","sites":[{"name":"Existing Site"}],"settings":{"theme":"dark"}}`)
	if err := os.WriteFile(path+".bak", backup, 0600); err != nil {
		t.Fatal(err)
	}
	for _, corrupted := range []string{
		`{"locale":"en","sites":null,"settings":{}}`,
		`{"locale":"en","sites":[],"settings":null}`,
		`null`,
	} {
		if err := os.WriteFile(path, []byte(corrupted), 0600); err != nil {
			t.Fatal(err)
		}
		recovered, err := readPersistedState()
		if err != nil {
			t.Fatal(err)
		}
		if recovered.Locale != "hr" || !strings.Contains(string(recovered.Sites), "Existing Site") {
			t.Fatalf("invalid state silently discarded saved sites: %+v", recovered)
		}
	}
}

func TestCorruptFallbackPrimaryNeverOverwritesPriorForensicCopy(t *testing.T) {
	path := isolatedFallbackPath(t)
	if err := os.WriteFile(path+".bak", []byte(`{"locale":"en","sites":[],"settings":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("broken-JSON"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readPersistedState(); err != nil {
		t.Fatal(err)
	}
	if _, err := readPersistedState(); err != nil {
		t.Fatal(err)
	}
	copies, err := filepath.Glob(filepath.Join(filepath.Dir(path), "fallback-state.json.corrupt.*"))
	if err != nil || len(copies) != 1 {
		t.Fatalf("repeat recovery should retain just one identical snapshot, got %d: %v", len(copies), err)
	}
	if err := os.WriteFile(path, []byte("different-broken-JSON"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readPersistedState(); err != nil {
		t.Fatal(err)
	}
	copies, err = filepath.Glob(filepath.Join(filepath.Dir(path), "fallback-state.json.corrupt.*"))
	if err != nil || len(copies) != 2 {
		t.Fatalf("distinct corruption should retain separate snapshots, got %d: %v", len(copies), err)
	}
	got := make(map[string]bool)
	for _, copy := range copies {
		data, err := os.ReadFile(copy)
		if err != nil {
			t.Fatal(err)
		}
		got[string(data)] = true
	}
	if !got["broken-JSON"] || !got["different-broken-JSON"] {
		t.Fatalf("forensic snapshots changed original bytes: %+v", got)
	}
}

func TestFallbackRecoveryRefusesLinkedBackup(t *testing.T) {
	path := isolatedFallbackPath(t)
	sentinel := filepath.Join(t.TempDir(), "external-config")
	original := []byte(`{"locale":"hr","sites":[{"name":"Other data"}],"settings":{}}`)
	if err := os.WriteFile(sentinel, original, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sentinel, path+".bak"); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("Windows symlink privilege unavailable: %v", err)
		}
		t.Fatal(err)
	}
	if recovered, err := readPersistedState(); err == nil {
		t.Fatalf("following an external backup link bypassed recovery restrictions: %+v", recovered)
	}
	after, err := os.ReadFile(sentinel)
	if err != nil || string(after) != string(original) {
		t.Fatalf("external state changed after rejected recovery: %v", err)
	}
}
