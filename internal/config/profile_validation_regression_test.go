package config

import (
	"strings"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func validFTPProfileInput() model.ProfileInput {
	return model.ProfileInput{
		Name:     "Test",
		Protocol: "ftp",
		Host:     "example.test",
		Port:     21,
		Username: "alice",
	}
}

func TestProfileSaveRejectsInvalidUTF8Fields(t *testing.T) {
	invalid := string([]byte{0xff})
	for _, mutate := range []func(*model.ProfileInput){
		func(in *model.ProfileInput) { in.Name = invalid },
		func(in *model.ProfileInput) { in.LocalPath = invalid },
		func(in *model.ProfileInput) {
			in.Protocol = "sftp"
			in.Port = 22
			in.PrivateKeyPath = invalid
		},
	} {
		in := validFTPProfileInput()
		mutate(&in)
		profiles := NewProfiles(New(t.TempDir()))
		if _, err := profiles.Save(in); err == nil {
			t.Fatalf("profile accepted invalid UTF-8 input: %#v", in)
		}
	}
}

func TestProfileSaveRejectsMalformedSFTPFingerprint(t *testing.T) {
	in := validFTPProfileInput()
	in.Protocol = "sftp"
	in.Port = 22
	in.Fingerprint = "SHA256:short"
	profiles := NewProfiles(New(t.TempDir()))
	if _, err := profiles.Save(in); err == nil {
		t.Fatal("profile accepted malformed SFTP fingerprint")
	}
}

func TestProfileSaveAcceptsCanonicalSFTPFingerprint(t *testing.T) {
	in := validFTPProfileInput()
	in.Protocol = "sftp"
	in.Port = 22
	in.Fingerprint = "SHA256:" + strings.Repeat("A", 43)
	profiles := NewProfiles(New(t.TempDir()))
	saved, err := profiles.Save(in)
	if err != nil {
		t.Fatalf("canonical SFTP fingerprint rejected: %v", err)
	}
	if saved.Fingerprint != in.Fingerprint {
		t.Fatalf("fingerprint changed during save: %q != %q", saved.Fingerprint, in.Fingerprint)
	}
}
