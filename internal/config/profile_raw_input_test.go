package config

import (
	"strings"
	"testing"

	"github.com/bren-wp/by-ftp/internal/model"
)

func rawProfileInput() model.ProfileInput {
	return model.ProfileInput{
		Name:       "Raw input test",
		Protocol:   "sftp",
		Host:       "example.test",
		Port:       22,
		Username:   "deploy",
		RemotePath: ".",
	}
}

func TestProfileSaveRejectsNonCanonicalRawProtocol(t *testing.T) {
	for _, protocol := range []string{"SFTP", " sftp", "sftp ", "sftp\t"} {
		in := rawProfileInput()
		in.Protocol = protocol
		if _, err := NewProfiles(New(t.TempDir())).Save(in); err == nil {
			t.Fatalf("non-canonical raw protocol %q unexpectedly accepted", protocol)
		}
	}
}

func TestProfileSaveRejectsRawHostBeforeNormalization(t *testing.T) {
	for _, host := range []string{" example.test", "example.test ", "example.test\r\n", "example.test\t"} {
		in := rawProfileInput()
		in.Host = host
		if _, err := NewProfiles(New(t.TempDir())).Save(in); err == nil {
			t.Fatalf("non-canonical raw host %q unexpectedly accepted", host)
		}
	}
}

func TestProfileSaveRejectsUsernameControlsBeforeNormalization(t *testing.T) {
	for _, username := range []string{"deploy\r", "deploy\n", "deploy\x00root"} {
		in := rawProfileInput()
		in.Username = username
		if _, err := NewProfiles(New(t.TempDir())).Save(in); err == nil {
			t.Fatalf("raw username control %q unexpectedly accepted", username)
		}
	}
}

func TestProfileSavePreservesUsernameVerbatim(t *testing.T) {
	in := rawProfileInput()
	in.Username = " deploy account "
	saved, err := NewProfiles(New(t.TempDir())).Save(in)
	if err != nil {
		t.Fatalf("backend-compatible username unexpectedly rejected: %v", err)
	}
	if saved.Username != in.Username {
		t.Fatalf("username was silently normalized: got %q want %q", saved.Username, in.Username)
	}
}

func TestProfileSaveRejectsNonCanonicalFingerprintBeforeNormalization(t *testing.T) {
	canonical := "SHA256:" + strings.Repeat("A", 43)
	for _, fingerprint := range []string{" " + canonical, canonical + " ", canonical + "\r\n", canonical + "\t"} {
		in := rawProfileInput()
		in.Fingerprint = fingerprint
		if _, err := NewProfiles(New(t.TempDir())).Save(in); err == nil {
			t.Fatalf("non-canonical raw fingerprint %q unexpectedly accepted", fingerprint)
		}
	}
}

func TestProfileUpdateFingerprintRejectsNonCanonicalRawInput(t *testing.T) {
	canonical := "SHA256:" + strings.Repeat("A", 43)
	for _, fingerprint := range []string{" " + canonical, canonical + " ", canonical + "\r\n", canonical + "\t"} {
		profiles := NewProfiles(New(t.TempDir()))
		saved, err := profiles.Save(rawProfileInput())
		if err != nil {
			t.Fatal(err)
		}
		if err := profiles.UpdateFingerprint(saved.ID, fingerprint); err == nil {
			t.Fatalf("non-canonical direct fingerprint %q unexpectedly accepted", fingerprint)
		}
	}
}

func TestProfileSaveAcceptsCanonicalConnectionAndFingerprint(t *testing.T) {
	in := rawProfileInput()
	in.Fingerprint = "SHA256:" + strings.Repeat("A", 43)
	saved, err := NewProfiles(New(t.TempDir())).Save(in)
	if err != nil {
		t.Fatalf("canonical profile unexpectedly rejected: %v", err)
	}
	if saved.Protocol != in.Protocol || saved.Host != in.Host || saved.Username != in.Username || saved.Fingerprint != in.Fingerprint {
		t.Fatalf("canonical profile fields changed: %#v", saved)
	}
}
