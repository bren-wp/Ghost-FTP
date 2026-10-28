package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
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
