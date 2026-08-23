package remote

import (
	"testing"

	"github.com/bren-wp/by-ftp/internal/model"
)

func TestMergeConnectionAllowsClearingPrivateKeyAndFingerprint(t *testing.T) {
	base := model.ConnectionConfig{
		Protocol: "sftp", Host: "example.test", Port: 22, Username: "tester",
		PrivateKeyPath: `C:\Keys\id_ed25519`, Fingerprint: "SHA256:old",
	}
	override := model.ConnectionConfig{
		Protocol: "sftp", Host: "example.test", Port: 22, Username: "tester",
		PrivateKeyPath: "", Fingerprint: "",
	}
	got := mergeConnection(base, override)
	if got.PrivateKeyPath != "" {
		t.Fatalf("cleared private key was inherited again: %q", got.PrivateKeyPath)
	}
	if got.Fingerprint != "" {
		t.Fatalf("profile fingerprint leaked into live config merge: %q", got.Fingerprint)
	}
}

func TestProfileEndpointMatchesNormalizesHost(t *testing.T) {
	profile := model.Profile{ID: "p1", Protocol: "sftp", Host: "Example.TEST.", Port: 22}
	cfg := model.ConnectionConfig{Protocol: "SFTP", Host: "example.test", Port: 22}
	if !profileEndpointMatches(profile, cfg) {
		t.Fatal("equivalent SFTP endpoint was not recognized")
	}

	for _, changed := range []model.ConnectionConfig{
		{Protocol: "sftp", Host: "other.test", Port: 22},
		{Protocol: "sftp", Host: "example.test", Port: 2222},
		{Protocol: "ftp", Host: "example.test", Port: 22},
	} {
		if profileEndpointMatches(profile, changed) {
			t.Fatalf("changed endpoint was treated as same profile endpoint: %#v", changed)
		}
	}
}

func TestProfilePasswordBindingIncludesUsername(t *testing.T) {
	profile := model.Profile{ID: "p1", Protocol: "sftp", Host: "example.test", Port: 22, Username: "alice"}
	cfg := model.ConnectionConfig{Protocol: "sftp", Host: "example.test", Port: 22, Username: "alice"}
	if !profileAccountMatches(profile, cfg) {
		t.Fatal("same account was not recognized")
	}
	cfg.Username = "bob"
	if profileAccountMatches(profile, cfg) {
		t.Fatal("stored password would cross username boundary")
	}
}

func TestProfilePassphraseBindingIncludesPrivateKey(t *testing.T) {
	profile := model.Profile{
		ID: "p1", Protocol: "sftp", Host: "example.test", Port: 22, Username: "alice",
		PrivateKeyPath: `C:\Keys\id_ed25519`,
	}
	cfg := model.ConnectionConfig{
		Protocol: "sftp", Host: "example.test", Port: 22, Username: "alice",
		PrivateKeyPath: `c:\keys\ID_ED25519`,
	}
	if !profilePrivateKeyMatches(profile, cfg) {
		t.Fatal("same Windows private-key path was not recognized")
	}

	cfg.PrivateKeyPath = ""
	if profilePrivateKeyMatches(profile, cfg) {
		t.Fatal("stored passphrase would survive private-key removal")
	}
	cfg.PrivateKeyPath = `C:\Keys\other_key`
	if profilePrivateKeyMatches(profile, cfg) {
		t.Fatal("stored passphrase would cross private-key boundary")
	}
}
