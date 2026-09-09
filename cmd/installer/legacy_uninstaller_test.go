package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSameCanonicalLegacyCommand(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "Uninstall.exe")

	for _, raw := range []string{legacy, `"` + legacy + `"`, "  \"" + legacy + "\"  "} {
		if !sameCanonicalLegacyCommand(raw, legacy) {
			t.Fatalf("expected canonical legacy command to match: %q", raw)
		}
	}
	if sameCanonicalLegacyCommand(`"`+legacy+`" /S`, legacy) {
		t.Fatal("legacy command with extra arguments was accepted")
	}
	if sameCanonicalLegacyCommand(filepath.Join(dir, "Other.exe"), legacy) {
		t.Fatal("foreign executable path was accepted")
	}
	if sameCanonicalLegacyCommand("Uninstall.exe", legacy) {
		t.Fatal("relative legacy executable path was accepted as ownership evidence")
	}
}

func TestCaptureLegacyUninstallerProofRequiresRegistryOwnership(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "Uninstall.exe")
	if err := os.WriteFile(legacy, []byte("foreign"), 0600); err != nil {
		t.Fatal(err)
	}

	proof := captureLegacyUninstallerProof(dir, registrySnapshot{})
	if proof.owned {
		t.Fatal("unregistered legacy file was treated as owned")
	}
	if proof.warning == "" {
		t.Fatal("unregistered legacy file did not produce a preservation warning")
	}
	if got, err := os.ReadFile(legacy); err != nil || string(got) != "foreign" {
		t.Fatalf("unregistered legacy file was not preserved: %q %v", got, err)
	}
}

func TestCaptureLegacyUninstallerProofBindsDigest(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "Uninstall.exe")
	if err := os.WriteFile(legacy, []byte("legacy ghost ftp uninstaller"), 0600); err != nil {
		t.Fatal(err)
	}

	snapshot := registrySnapshot{strings: []registryStringSnapshot{{
		key: uninstallKey, name: "UninstallString", value: `"` + legacy + `"`, existed: true,
	}}}
	proof := captureLegacyUninstallerProof(dir, snapshot)
	if !proof.owned || proof.path != legacy || proof.digest == "" {
		t.Fatalf("registered legacy ownership was not captured: %+v", proof)
	}
}

func TestRegistrySnapshotStringValueRequiresExistingEntry(t *testing.T) {
	snapshot := registrySnapshot{strings: []registryStringSnapshot{
		{key: uninstallKey, name: "UninstallString", value: "old", existed: true},
		{key: uninstallKey, name: "DisplayName", value: "ignored", existed: false},
	}}
	if got, ok := snapshot.stringValue(uninstallKey, "UninstallString"); !ok || got != "old" {
		t.Fatalf("unexpected captured value: %q %v", got, ok)
	}
	if _, ok := snapshot.stringValue(uninstallKey, "DisplayName"); ok {
		t.Fatal("non-existing registry snapshot entry was exposed as evidence")
	}
}
