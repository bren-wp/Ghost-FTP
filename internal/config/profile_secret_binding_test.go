package config

import (
	"testing"

	"github.com/bren-wp/by-ftp/internal/model"
)

func TestProfileSavePreservesPasswordForSameAccount(t *testing.T) {
	profiles := NewProfiles(New(t.TempDir()))
	seedProfile(t, profiles, model.Profile{
		ID: "p1", Name: "SFTP", Protocol: "sftp", Host: "Example.TEST.", Port: 22, Username: "alice",
		PasswordBlob: "protected-password", RemotePath: ".",
	})

	saved, err := profiles.Save(model.ProfileInput{
		ID: "p1", Name: "Preimenovan", Protocol: "sftp", Host: "example.test", Port: 22, Username: "alice", RemotePath: ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !saved.HasPassword {
		t.Fatal("same account unexpectedly lost stored password")
	}
}

func TestProfileSaveClearsPasswordWhenAccountIdentityChanges(t *testing.T) {
	for _, change := range []model.ProfileInput{
		{ID: "p1", Name: "SFTP", Protocol: "sftp", Host: "other.test", Port: 22, Username: "alice", RemotePath: "."},
		{ID: "p1", Name: "SFTP", Protocol: "sftp", Host: "example.test", Port: 2222, Username: "alice", RemotePath: "."},
		{ID: "p1", Name: "SFTP", Protocol: "sftp", Host: "example.test", Port: 22, Username: "bob", RemotePath: "."},
		{ID: "p1", Name: "FTP", Protocol: "ftp", Host: "example.test", Port: 21, Username: "alice", RemotePath: "/"},
	} {
		profiles := NewProfiles(New(t.TempDir()))
		seedProfile(t, profiles, model.Profile{
			ID: "p1", Name: "SFTP", Protocol: "sftp", Host: "example.test", Port: 22, Username: "alice",
			PasswordBlob: "protected-password", RemotePath: ".",
		})
		saved, err := profiles.Save(change)
		if err != nil {
			t.Fatal(err)
		}
		if saved.HasPassword {
			t.Fatalf("stored password crossed profile account boundary: %#v", change)
		}
	}
}

func TestProfileSavePreservesPassphraseOnlyForSamePrivateKeyIdentity(t *testing.T) {
	const keyPath = `/home/alice/.ssh/id_ed25519`
	profiles := NewProfiles(New(t.TempDir()))
	seedProfile(t, profiles, model.Profile{
		ID: "p1", Name: "SFTP", Protocol: "sftp", Host: "example.test", Port: 22, Username: "alice",
		PrivateKeyPath: keyPath, PassphraseBlob: "protected-passphrase", RemotePath: ".",
	})

	saved, err := profiles.Save(model.ProfileInput{
		ID: "p1", Name: "Preimenovan", Protocol: "sftp", Host: "example.test", Port: 22, Username: "alice",
		PrivateKeyPath: keyPath, RemotePath: ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !saved.HasPassphrase {
		t.Fatal("same private-key identity unexpectedly lost stored passphrase")
	}

	saved, err = profiles.Save(model.ProfileInput{
		ID: "p1", Name: "Drugi ključ", Protocol: "sftp", Host: "example.test", Port: 22, Username: "alice",
		PrivateKeyPath: `/home/alice/.ssh/other`, RemotePath: ".",
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.HasPassphrase {
		t.Fatal("stored passphrase crossed private-key boundary")
	}
}
