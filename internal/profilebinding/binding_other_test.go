//go:build !windows

package profilebinding

import "testing"

func TestPrivateKeyMatchesDoesNotCrossCaseSensitivePathBoundary(t *testing.T) {
	if PrivateKeyMatches(
		"sftp", "example.test", 22, "alice", `/home/alice/Keys/id_ed25519`,
		"sftp", "example.test", 22, "alice", `/home/alice/keys/id_ed25519`,
	) {
		t.Fatal("non-Windows private-key path case change crossed key identity boundary")
	}
}
