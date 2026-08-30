//go:build windows

package profilebinding

import "testing"

func TestPrivateKeyMatchesUsesWindowsCaseInsensitivePathIdentity(t *testing.T) {
	if !PrivateKeyMatches(
		"sftp", "example.test", 22, "alice", `C:\Keys\id_ed25519`,
		"sftp", "example.test", 22, "alice", `c:\keys\ID_ED25519`,
	) {
		t.Fatal("Windows private-key path case change should preserve key identity")
	}
}
