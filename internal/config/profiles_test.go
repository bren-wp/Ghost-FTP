package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bren-wp/by-ftp/internal/model"
)

var canonicalTestFingerprint = "SHA256:" + strings.Repeat("A", 43)

func TestSFTPProfileDefaultsToHomeDirectory(t *testing.T) {
	profiles := NewProfiles(New(t.TempDir()))
	saved, err := profiles.Save(model.ProfileInput{
		Name:     "SFTP test",
		Protocol: "sftp",
		Host:     "example.test",
		Port:     22,
		Username: "tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.RemotePath != "." {
		t.Fatalf("RemotePath=%q want .", saved.RemotePath)
	}
}

func TestFTPProfileDefaultsToRootDirectory(t *testing.T) {
	profiles := NewProfiles(New(t.TempDir()))
	saved, err := profiles.Save(model.ProfileInput{
		Name:     "FTP test",
		Protocol: "ftp",
		Host:     "example.test",
		Port:     21,
		Username: "tester",
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.RemotePath != "/" {
		t.Fatalf("RemotePath=%q want /", saved.RemotePath)
	}
}

func TestSavedProfileMetadataIsNotPlaintextOnDisk(t *testing.T) {
	dir := t.TempDir()
	profiles := NewProfiles(New(dir))
	_, err := profiles.Save(model.ProfileInput{
		Name: "Private profile", Protocol: "ftp", Host: "private.example.test", Port: 21,
		Username: "private-user", LocalPath: `C:\Users\Private\Site`, RemotePath: "/private",
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, secureProfilesFile))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, secret := range []string{"private.example.test", "private-user", "Private profile", "/private"} {
		if strings.Contains(text, secret) {
			t.Fatalf("profile metadata leaked in plaintext state file: %q", secret)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "profiles.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy plaintext profile file still exists: %v", err)
	}
}

func TestLegacyProfilesAreMigratedAndPlaintextRemoved(t *testing.T) {
	dir := t.TempDir()
	legacy := []model.Profile{{ID: "legacy", Name: "Legacy", Protocol: "ftp", Host: "legacy.example.test", Port: 21, Username: "legacy-user", RemotePath: "/"}}
	b, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "profiles.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	profiles := NewProfiles(New(dir))
	got, err := profiles.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Host != "legacy.example.test" {
		t.Fatalf("migration failed: %#v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "profiles.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("legacy plaintext file was not removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, secureProfilesFile)); err != nil {
		t.Fatalf("secure profile envelope missing: %v", err)
	}
}

func seedProfile(t *testing.T, profiles *Profiles, profile model.Profile) {
	t.Helper()
	if err := profiles.saveAll([]model.Profile{profile}); err != nil {
		t.Fatal(err)
	}
}

func TestRemovingSFTPPrivateKeyClearsStoredPassphrase(t *testing.T) {
	profiles := NewProfiles(New(t.TempDir()))
	seedProfile(t, profiles, model.Profile{
		ID: "p1", Name: "SFTP", Protocol: "sftp", Host: "example.test", Port: 22, Username: "tester",
		PrivateKeyPath: `C:\Keys\id_ed25519`, PassphraseBlob: "protected-passphrase", Fingerprint: canonicalTestFingerprint, RemotePath: ".",
	})

	saved, err := profiles.Save(model.ProfileInput{
		ID: "p1", Name: "SFTP", Protocol: "sftp", Host: "example.test", Port: 22, Username: "tester", PrivateKeyPath: "", RemotePath: ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.PrivateKeyPath != "" || saved.HasPassphrase {
		t.Fatalf("private key/passphrase not cleared: %#v", saved)
	}
	if saved.Fingerprint != canonicalTestFingerprint {
		t.Fatalf("same endpoint fingerprint was lost: %q", saved.Fingerprint)
	}
	stored, err := profiles.Get("p1")
	if err != nil {
		t.Fatal(err)
	}
	if stored.PassphraseBlob != "" {
		t.Fatal("dead passphrase blob remained after private key removal")
	}
}

func TestPassphraseWithoutPrivateKeyRejectedBeforeProtection(t *testing.T) {
	profiles := NewProfiles(New(t.TempDir()))
	_, err := profiles.Save(model.ProfileInput{
		Name: "SFTP", Protocol: "sftp", Host: "example.test", Port: 22, Username: "tester", Passphrase: "secret",
	})
	if err == nil || !strings.Contains(err.Error(), "zahtijeva odabran privatni ključ") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClearFlagsRemoveOnlyRequestedStoredSecrets(t *testing.T) {
	profiles := NewProfiles(New(t.TempDir()))
	seedProfile(t, profiles, model.Profile{
		ID: "p1", Name: "SFTP", Protocol: "sftp", Host: "example.test", Port: 22, Username: "tester",
		PasswordBlob: "protected-password", PrivateKeyPath: `C:\Keys\id_ed25519`, PassphraseBlob: "protected-passphrase", RemotePath: ".",
	})

	saved, err := profiles.Save(model.ProfileInput{
		ID: "p1", Name: "SFTP", Protocol: "sftp", Host: "example.test", Port: 22, Username: "tester",
		PrivateKeyPath: `C:\Keys\id_ed25519`, ClearPassword: true, RemotePath: ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.HasPassword || !saved.HasPassphrase {
		t.Fatalf("wrong secret preservation after password clear: %#v", saved)
	}

	saved, err = profiles.Save(model.ProfileInput{
		ID: "p1", Name: "SFTP", Protocol: "sftp", Host: "example.test", Port: 22, Username: "tester",
		PrivateKeyPath: `C:\Keys\id_ed25519`, ClearPassphrase: true, RemotePath: ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.HasPassword || saved.HasPassphrase {
		t.Fatalf("stored secret was not cleared: %#v", saved)
	}
}

func TestProfileSavePreservesFingerprintForSameEndpoint(t *testing.T) {
	profiles := NewProfiles(New(t.TempDir()))
	seedProfile(t, profiles, model.Profile{
		ID: "p1", Name: "Stari naziv", Protocol: "sftp", Host: "Example.TEST.", Port: 22, Username: "tester", Fingerprint: canonicalTestFingerprint, RemotePath: ".",
	})

	saved, err := profiles.Save(model.ProfileInput{
		ID: "p1", Name: "Novi naziv", Protocol: "sftp", Host: "example.test", Port: 22, Username: "drugi-korisnik", RemotePath: ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.Fingerprint != canonicalTestFingerprint {
		t.Fatalf("fingerprint=%q want preserved pin", saved.Fingerprint)
	}
}

func TestProfileSaveClearsFingerprintWhenEndpointChanges(t *testing.T) {
	for _, change := range []model.ProfileInput{
		{ID: "p1", Name: "SFTP", Protocol: "sftp", Host: "other.test", Port: 22, Username: "tester", RemotePath: "."},
		{ID: "p1", Name: "SFTP", Protocol: "sftp", Host: "example.test", Port: 2222, Username: "tester", RemotePath: "."},
		{ID: "p1", Name: "FTP", Protocol: "ftp", Host: "example.test", Port: 21, Username: "tester", RemotePath: "/"},
	} {
		profiles := NewProfiles(New(t.TempDir()))
		seedProfile(t, profiles, model.Profile{
			ID: "p1", Name: "SFTP", Protocol: "sftp", Host: "example.test", Port: 22, Username: "tester", Fingerprint: canonicalTestFingerprint, RemotePath: ".",
		})
		saved, err := profiles.Save(change)
		if err != nil {
			t.Fatal(err)
		}
		if saved.Fingerprint != "" {
			t.Fatalf("changed endpoint kept stale fingerprint %q for %#v", saved.Fingerprint, change)
		}
	}
}

func TestUpdateFingerprintRejectsNonSFTPProfile(t *testing.T) {
	profiles := NewProfiles(New(t.TempDir()))
	seedProfile(t, profiles, model.Profile{ID: "p1", Name: "FTP", Protocol: "ftp", Host: "example.test", Port: 21, Username: "tester", RemotePath: "/"})
	if err := profiles.UpdateFingerprint("p1", canonicalTestFingerprint); err == nil {
		t.Fatal("expected non-SFTP fingerprint update to fail")
	}
}
