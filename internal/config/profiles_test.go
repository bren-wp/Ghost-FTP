package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"brendigo.com/byftp/internal/model"
)

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
