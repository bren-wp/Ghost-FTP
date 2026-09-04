package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestDefaultSettingsAreValid(t *testing.T) {
	got := DefaultSettings()
	if err := validateSettings(got); err != nil {
		t.Fatalf("DefaultSettings is invalid: %v", err)
	}
	if got.Parallelism != DefaultParallelism ||
		got.AutoRetryCount != DefaultAutoRetryCount ||
		got.RetryDelaySeconds != DefaultRetryDelaySeconds ||
		got.ConnectionTimeoutSeconds != DefaultConnectionTimeoutSeconds {
		t.Fatalf("DefaultSettings drifted from canonical constants: %#v", got)
	}
	if !got.BackupBeforeOverwrite || !got.ConfirmDelete {
		t.Fatalf("safe overwrite/delete defaults were weakened: %#v", got)
	}
}

func TestEffectiveSettingsHandlesNilStore(t *testing.T) {
	var nilSettings *SettingsStore
	if got, want := nilSettings.Effective(), DefaultSettings(); got != want {
		t.Fatalf("nil settings fallback = %#v, want %#v", got, want)
	}

	settings := &SettingsStore{}
	if got, want := settings.Effective(), DefaultSettings(); got != want {
		t.Fatalf("nil store fallback = %#v, want %#v", got, want)
	}
}

func TestEffectiveSettingsFallsBackWhenStateDirectoryIsUnavailable(t *testing.T) {
	root := t.TempDir()
	blocked := filepath.Join(root, "state")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	settings := NewSettings(New(blocked))
	if got, want := settings.Effective(), DefaultSettings(); got != want {
		t.Fatalf("unavailable state fallback = %#v, want %#v", got, want)
	}
}

func TestGetNormalizesLegacyOutOfRangeSettings(t *testing.T) {
	store := New(t.TempDir())
	legacy := model.Settings{
		Language:                 "",
		Parallelism:              99,
		BackupBeforeOverwrite:    true,
		ConfirmDelete:            true,
		AutoRetryCount:           99,
		RetryDelaySeconds:        0,
		ConnectionTimeoutSeconds: 0,
	}
	if err := store.Write("settings.json", legacy); err != nil {
		t.Fatal(err)
	}
	got, err := NewSettings(store).Get()
	if err != nil {
		t.Fatal(err)
	}
	want := DefaultSettings()
	if got != want {
		t.Fatalf("legacy settings normalized to %#v, want %#v", got, want)
	}
}
