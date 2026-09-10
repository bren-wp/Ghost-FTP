package config

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestProfileRemoteStartDoesNotSilentlyCrossAccountIdentity(t *testing.T) {
	profiles := NewProfiles(New(t.TempDir()))
	first, err := profiles.Save(model.ProfileInput{
		Name: "Site", Protocol: "ftps", Host: "old.example.com", Port: 21,
		Username: "alice", RemotePath: "/alice-root", LocalPath: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("initial Save: %v", err)
	}

	changed, err := profiles.Save(model.ProfileInput{
		ID: first.ID, Name: "Site", Protocol: "ftps", Host: "new.example.com", Port: 21,
		Username: "alice", RemotePath: first.RemotePath, LocalPath: first.LocalPath,
	})
	if err != nil {
		t.Fatalf("identity-changing Save: %v", err)
	}
	if changed.RemotePath != "/" {
		t.Fatalf("inherited stale remote start = %q, want protocol default /", changed.RemotePath)
	}
}

func TestProfileExplicitNewRemoteStartSurvivesIdentityChange(t *testing.T) {
	profiles := NewProfiles(New(t.TempDir()))
	first, err := profiles.Save(model.ProfileInput{
		Name: "Site", Protocol: "sftp", Host: "host.example.com", Port: 22,
		Username: "alice", RemotePath: "/home/alice", LocalPath: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("initial Save: %v", err)
	}

	changed, err := profiles.Save(model.ProfileInput{
		ID: first.ID, Name: "Site", Protocol: "sftp", Host: "host.example.com", Port: 22,
		Username: "bob", RemotePath: "/home/bob", LocalPath: first.LocalPath,
	})
	if err != nil {
		t.Fatalf("explicit new start Save: %v", err)
	}
	if changed.RemotePath != "/home/bob" {
		t.Fatalf("explicit new remote start = %q, want /home/bob", changed.RemotePath)
	}
}

func TestProfileInheritedSFTPRemoteStartResetsToDot(t *testing.T) {
	profiles := NewProfiles(New(t.TempDir()))
	first, err := profiles.Save(model.ProfileInput{
		Name: "Site", Protocol: "sftp", Host: "one.example.com", Port: 22,
		Username: "alice", RemotePath: "/srv/site", LocalPath: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("initial Save: %v", err)
	}

	changed, err := profiles.Save(model.ProfileInput{
		ID: first.ID, Name: "Site", Protocol: "sftp", Host: "two.example.com", Port: 22,
		Username: "alice", RemotePath: first.RemotePath, LocalPath: first.LocalPath,
	})
	if err != nil {
		t.Fatalf("identity-changing Save: %v", err)
	}
	if changed.RemotePath != "." {
		t.Fatalf("inherited SFTP remote start = %q, want protocol default .", changed.RemotePath)
	}
}
