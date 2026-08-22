package userdata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateLegacyMovesExistingDataWithoutOverwrite(t *testing.T) {
	base := t.TempDir()
	legacy := LegacyDir(base, "ByFTP")
	if err := os.MkdirAll(legacy, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "settings.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := MigrateLegacy(base, "ByFTP"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(base, "ByFTP", "settings.json")); err != nil {
		t.Fatalf("migrated settings missing: %v", err)
	}
}

func TestMigrateLegacyKeepsExistingDestinationAuthoritative(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "ByFTP")
	if err := os.MkdirAll(target, 0700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(target, "current")
	if err := os.WriteFile(marker, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	legacy := LegacyDir(base, "ByFTP")
	if err := os.MkdirAll(legacy, 0700); err != nil {
		t.Fatal(err)
	}
	if err := MigrateLegacy(base, "ByFTP"); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(marker); err != nil || string(got) != "keep" {
		t.Fatalf("existing destination changed: %q, %v", got, err)
	}
}
