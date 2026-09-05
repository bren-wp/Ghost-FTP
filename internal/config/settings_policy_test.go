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
	if got.ConflictPolicy != model.ConflictPolicyReplaceBackup || !got.BackupBeforeOverwrite || got.SkipExisting || !got.ConfirmDelete {
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

func TestLegacyConflictPolicyMigrationPreservesBehavior(t *testing.T) {
	tests := []struct {
		name   string
		legacy model.Settings
		policy string
		skip   bool
		backup bool
	}{
		{
			name: "skip wins even if old backup flag was also set",
			legacy: model.Settings{SkipExisting: true, BackupBeforeOverwrite: true},
			policy: model.ConflictPolicySkip, skip: true, backup: false,
		},
		{
			name: "replace with recovery backup",
			legacy: model.Settings{BackupBeforeOverwrite: true},
			policy: model.ConflictPolicyReplaceBackup, skip: false, backup: true,
		},
		{
			name: "replace without retained recovery backup",
			legacy: model.Settings{},
			policy: model.ConflictPolicyReplace, skip: false, backup: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := migrateConflictPolicy(test.legacy, true)
			if got.ConflictPolicy != test.policy || got.SkipExisting != test.skip || got.BackupBeforeOverwrite != test.backup {
				t.Fatalf("migration = %#v; want policy=%q skip=%v backup=%v", got, test.policy, test.skip, test.backup)
			}
		})
	}
}

func TestCanonicalConflictPolicyWinsOverLegacyFlagsOnSave(t *testing.T) {
	store := New(t.TempDir())
	settings := NewSettings(store)
	value := DefaultSettings()
	value.ConflictPolicy = model.ConflictPolicySkip
	value.SkipExisting = false
	value.BackupBeforeOverwrite = true
	got, err := settings.Set(value)
	if err != nil {
		t.Fatal(err)
	}
	if got.ConflictPolicy != model.ConflictPolicySkip || !got.SkipExisting || got.BackupBeforeOverwrite {
		t.Fatalf("canonical policy did not synchronize legacy flags: %#v", got)
	}
}

func TestInvalidNewConflictPolicyIsRejected(t *testing.T) {
	value := DefaultSettings()
	value.ConflictPolicy = "overwrite_everything"
	if _, err := NewSettings(New(t.TempDir())).Set(value); err == nil {
		t.Fatal("expected unsupported conflict policy to be rejected")
	}
}

func TestUnknownPersistedConflictPolicyFallsBackToRecovery(t *testing.T) {
	store := New(t.TempDir())
	value := DefaultSettings()
	value.ConflictPolicy = "future_or_corrupt_value"
	value.SkipExisting = false
	value.BackupBeforeOverwrite = false
	if err := store.Write("settings.json", value); err != nil {
		t.Fatal(err)
	}
	got, err := NewSettings(store).Get()
	if err != nil {
		t.Fatal(err)
	}
	if got.ConflictPolicy != model.ConflictPolicyReplaceBackup || got.SkipExisting || !got.BackupBeforeOverwrite {
		t.Fatalf("unknown persisted policy did not fail closed: %#v", got)
	}
}
