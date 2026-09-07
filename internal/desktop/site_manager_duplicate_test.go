package desktop

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestDuplicateProfileDraftCopiesOnlyNonSecretConfiguration(t *testing.T) {
	source := model.PublicProfile{
		ID:             "profile-1",
		Name:           "Production",
		Protocol:       "sftp",
		Host:           "files.example.test",
		Port:           22,
		Username:       "deploy",
		HasPassword:    true,
		PrivateKeyPath: `C:\Keys\deploy_ed25519`,
		HasPassphrase:  true,
		Fingerprint:    "SHA256:trusted-original-host-key",
		RemotePath:     "/srv/www",
		LocalPath:      `C:\Sites\Production`,
	}

	got := duplicateProfileDraft(source)

	if got.ID != "" {
		t.Fatalf("duplicate retained source ID %q", got.ID)
	}
	if got.Password != "" || got.Passphrase != "" || got.Fingerprint != "" {
		t.Fatalf("duplicate copied secret or trust material: %#v", got)
	}
	if got.ClearPassword || got.ClearPassphrase {
		t.Fatalf("new duplicate draft must not carry destructive credential flags: %#v", got)
	}
	if got.Name != "Production 2" || got.Protocol != source.Protocol || got.Host != source.Host || got.Port != source.Port || got.Username != source.Username {
		t.Fatalf("duplicate lost connection configuration: %#v", got)
	}
	if got.PrivateKeyPath != source.PrivateKeyPath || got.RemotePath != source.RemotePath || got.LocalPath != source.LocalPath {
		t.Fatalf("duplicate lost non-secret path configuration: %#v", got)
	}
}
