package config

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestProfileSavePreservesPathFieldsVerbatim(t *testing.T) {
	in := model.ProfileInput{
		Name:           "Path preservation",
		Protocol:       "sftp",
		Host:           "example.test",
		Port:           22,
		Username:       "deploy",
		PrivateKeyPath: " /keys/id_ed25519 ",
		RemotePath:     "/srv/site ",
		LocalPath:      " /workspace/site ",
	}

	saved, err := NewProfiles(New(t.TempDir())).Save(in)
	if err != nil {
		t.Fatalf("valid path-bearing profile unexpectedly rejected: %v", err)
	}
	if saved.PrivateKeyPath != in.PrivateKeyPath {
		t.Fatalf("private key path was silently normalized: got %q want %q", saved.PrivateKeyPath, in.PrivateKeyPath)
	}
	if saved.RemotePath != in.RemotePath {
		t.Fatalf("remote path was silently normalized: got %q want %q", saved.RemotePath, in.RemotePath)
	}
	if saved.LocalPath != in.LocalPath {
		t.Fatalf("local path was silently normalized: got %q want %q", saved.LocalPath, in.LocalPath)
	}
}

func TestProfileSaveRejectsPathControlCharactersWithoutNormalization(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*model.ProfileInput)
	}{
		{name: "private key newline", mutate: func(in *model.ProfileInput) { in.PrivateKeyPath = "/keys/id\nother" }},
		{name: "remote path carriage return", mutate: func(in *model.ProfileInput) { in.RemotePath = "/srv/site\rhidden" }},
		{name: "local path nul", mutate: func(in *model.ProfileInput) { in.LocalPath = "/workspace/site\x00hidden" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := rawProfileInput()
			tt.mutate(&in)
			if _, err := NewProfiles(New(t.TempDir())).Save(in); err == nil {
				t.Fatal("profile path containing a control character unexpectedly accepted")
			}
		})
	}
}
