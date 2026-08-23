package config

import (
	"encoding/json"
	"github.com/bren-wp/by-ftp/internal/model"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

type testState struct {
	Value int `json:"value"`
}

func TestStoreOverwriteAndRecovery(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := s.Write("state.json", testState{Value: 1}); err != nil {
		t.Fatal(err)
	}
	if err := s.Write("state.json", testState{Value: 2}); err != nil {
		t.Fatal(err)
	}
	var got testState
	gen, err := s.Read("state.json", testState{}, &got)
	if err != nil || gen != "current" || got.Value != 2 {
		t.Fatalf("current read: gen=%s value=%d err=%v", gen, got.Value, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	gen, err = s.Read("state.json", testState{}, &got)
	if gen != "previous" || got.Value != 1 {
		t.Fatalf("recovery read: gen=%s value=%d err=%v", gen, got.Value, err)
	}
}

func TestStoreSerializesConcurrentWrites(t *testing.T) {
	s := New(t.TempDir())
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.Write("state.json", testState{Value: i}); err != nil {
				t.Errorf("write: %v", err)
			}
		}()
	}
	wg.Wait()
	var got testState
	if _, err := s.Read("state.json", testState{}, &got); err != nil {
		t.Fatal(err)
	}
	if got.Value < 0 || got.Value > 31 {
		t.Fatalf("invalid final value: %d", got.Value)
	}
}

func TestStoreFallsBackWhenBothGenerationsAreCorrupt(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "state.json.previous"), []byte("also broken"), 0600); err != nil {
		t.Fatal(err)
	}
	var got testState
	gen, err := s.Read("state.json", testState{Value: 42}, &got)
	if err != nil || gen != "fallback" || got.Value != 42 {
		t.Fatalf("fallback read: gen=%s value=%d err=%v", gen, got.Value, err)
	}
}

func TestStoreDoesNotOverwriteGoodPreviousWithCorruptCurrent(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := s.Write("state.json", testState{Value: 7}); err != nil {
		t.Fatal(err)
	}
	if err := s.Write("state.json", testState{Value: 8}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := s.Write("state.json", testState{Value: 9}); err != nil {
		t.Fatal(err)
	}
	var previous testState
	data, err := os.ReadFile(filepath.Join(dir, "state.json.previous"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &previous); err != nil {
		t.Fatal(err)
	}
	if previous.Value != 7 {
		t.Fatalf("previous generation unexpectedly replaced: got %d want 7", previous.Value)
	}
}

func TestSettingsReadsLegacyDarkOnlyFieldsWithoutFailing(t *testing.T) {
	dir := t.TempDir()
	legacy := []byte(`{"parallelism":3,"theme":"dark","backupBeforeOverwrite":true,"confirmDelete":false,"showActivityLog":true}`)
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), legacy, 0600); err != nil {
		t.Fatal(err)
	}
	settings, err := NewSettings(New(dir)).Get()
	if err != nil {
		t.Fatal(err)
	}
	if settings.Parallelism != 3 || !settings.BackupBeforeOverwrite || settings.ConfirmDelete {
		t.Fatalf("unexpected migrated settings: %+v", settings)
	}
}

func TestStoreRejectsTraversalNames(t *testing.T) {
	s := New(t.TempDir())
	for _, name := range []string{"", "../outside.json", `..\outside.json`, "a/b.json", `a\b.json`, ".", ".."} {
		if err := s.Write(name, testState{Value: 1}); err == nil {
			t.Fatalf("Write(%q) unexpectedly succeeded", name)
		}
	}
}

func TestStoreDoesNotReadSymlinkedState(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(outside, []byte(`{"value":99}`), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "state.json")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink not available: %v", err)
	}
	s := New(dir)
	var got testState
	gen, err := s.Read("state.json", testState{Value: 7}, &got)
	if err != nil {
		t.Fatal(err)
	}
	if gen != "fallback" || got.Value != 7 {
		t.Fatalf("symlinked state was followed: gen=%q value=%d", gen, got.Value)
	}
}

func TestSettingsStoreCachesAfterFirstRead(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"parallelism":4,"backupBeforeOverwrite":true,"confirmDelete":true,"retryDelaySeconds":3}`), 0600); err != nil {
		t.Fatal(err)
	}
	settings := NewSettings(New(dir))
	first, err := settings.Get()
	if err != nil {
		t.Fatal(err)
	}
	if first.Parallelism != 4 {
		t.Fatalf("first parallelism=%d want 4", first.Parallelism)
	}
	// External edits during a running ByFTP process are intentionally ignored;
	// the settings store is the single writer and hot-path reads stay in memory.
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"parallelism":1,"backupBeforeOverwrite":false,"confirmDelete":false,"retryDelaySeconds":3}`), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := settings.Get()
	if err != nil {
		t.Fatal(err)
	}
	if second != first {
		t.Fatalf("cached settings changed after external edit: first=%+v second=%+v", first, second)
	}
}

func TestSettingsStoreDoesNotCacheFailedInitialLoad(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "state")
	if err := os.WriteFile(dir, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	settings := NewSettings(New(dir))
	if _, err := settings.Get(); err == nil {
		t.Fatal("expected first load to fail")
	}
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"parallelism":7,"backupBeforeOverwrite":true,"confirmDelete":true,"retryDelaySeconds":3}`), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := settings.Get()
	if err != nil {
		t.Fatal(err)
	}
	if got.Parallelism != 7 {
		t.Fatalf("parallelism=%d want 7; failed load was incorrectly cached", got.Parallelism)
	}
}

func TestSettingsConnectionTimeoutAndLegacyRetryDelayNormalization(t *testing.T) {
	settings := NewSettings(New(t.TempDir()))
	got, err := settings.Set(model.Settings{
		Parallelism:              2,
		BackupBeforeOverwrite:    true,
		ConfirmDelete:            true,
		ConnectionTimeoutSeconds: 27,
		RetryDelaySeconds:        0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.ConnectionTimeoutSeconds != 27 {
		t.Fatalf("connection timeout=%d want 27", got.ConnectionTimeoutSeconds)
	}
	if got.RetryDelaySeconds != 3 {
		t.Fatalf("legacy retry delay=%d want normalized 3", got.RetryDelaySeconds)
	}

	for _, invalid := range []int{4, 61} {
		candidate := got
		candidate.ConnectionTimeoutSeconds = invalid
		if _, err := settings.Set(candidate); err == nil {
			t.Fatalf("invalid connection timeout %d unexpectedly accepted", invalid)
		}
	}
}
