package remote

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestResolveClearsSFTPOnlyStateForFTPFamily(t *testing.T) {
	for _, protocol := range []string{"ftp", "ftps", "ftpsi"} {
		m := &Manager{}
		resolved, _, err := m.Resolve("", model.ConnectionConfig{
			Protocol:       protocol,
			Host:           "example.test",
			Port:           21,
			Username:       "alice",
			Password:       "password",
			PrivateKeyPath: `/tmp/id_ed25519`,
			Passphrase:     "stale-passphrase",
			Fingerprint:    "SHA256:stale-fingerprint",
		})
		if err != nil {
			t.Fatalf("Resolve(%s): %v", protocol, err)
		}
		if resolved.Config.PrivateKeyPath != "" || resolved.Config.Passphrase != "" || resolved.Config.Fingerprint != "" || resolved.PassphraseBlob != "" {
			t.Fatalf("%s retained SFTP-only state: %#v", protocol, resolved)
		}
	}
}

func TestConnectionIdentityIgnoresSFTPOnlyStateForFTPFamily(t *testing.T) {
	base := model.ConnectionConfig{
		Protocol: "ftps",
		Host:     "example.test",
		Port:     21,
		Username: "alice",
	}
	stale := base
	stale.PrivateKeyPath = `/tmp/id_ed25519`
	stale.Passphrase = "unused"
	stale.Fingerprint = "SHA256:unused"
	if got, want := connectionIdentity(stale), connectionIdentity(base); got != want {
		t.Fatalf("dead SFTP state changed FTPS connection identity: %q != %q", got, want)
	}
}

func TestResolvePreservesSFTPStateForSFTP(t *testing.T) {
	m := &Manager{}
	fingerprint := "SHA256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	resolved, _, err := m.Resolve("", model.ConnectionConfig{
		Protocol:       "sftp",
		Host:           "example.test",
		Port:           22,
		Username:       "alice",
		PrivateKeyPath: `/tmp/id_ed25519`,
		Passphrase:     "passphrase",
		Fingerprint:    fingerprint,
	})
	if err != nil {
		t.Fatalf("Resolve(sftp): %v", err)
	}
	if resolved.Config.PrivateKeyPath == "" || resolved.Config.Passphrase == "" || resolved.Config.Fingerprint != fingerprint {
		t.Fatalf("SFTP state was unexpectedly removed: %#v", resolved)
	}
}
