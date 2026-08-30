package profilebinding

import "testing"

func TestEndpointMatchesNormalizesHost(t *testing.T) {
	if !EndpointMatches("sftp", "Example.TEST.", 22, "SFTP", "example.test", 22) {
		t.Fatal("equivalent endpoint was not matched")
	}
	if !EndpointMatches("sftp", "[2001:db8::1]", 22, "sftp", "2001:db8::1", 22) {
		t.Fatal("equivalent IPv6 endpoint was not matched")
	}
	for _, tc := range []struct {
		protocol string
		host     string
		port     int
	}{
		{"ftp", "example.test", 22},
		{"sftp", "other.test", 22},
		{"sftp", "example.test", 2222},
	} {
		if EndpointMatches("sftp", "example.test", 22, tc.protocol, tc.host, tc.port) {
			t.Fatalf("different endpoint matched: %#v", tc)
		}
	}
}

func TestEndpointKeyUsesSameCanonicalizationAsEndpointMatches(t *testing.T) {
	pairs := [][2]string{
		{"Example.TEST.", "example.test"},
		{"[2001:db8::1]", "2001:db8::1"},
	}
	for _, pair := range pairs {
		a := EndpointKey(" SFTP ", pair[0], 22)
		b := EndpointKey("sftp", pair[1], 22)
		if a != b {
			t.Fatalf("equivalent endpoints produced different keys: %q != %q", a, b)
		}
	}
	if EndpointKey("sftp", "example.test", 22) == EndpointKey("sftp", "example.test", 2222) {
		t.Fatal("different ports produced the same endpoint key")
	}
}

func TestAccountMatchesRequiresExactUsername(t *testing.T) {
	if !AccountMatches("sftp", "example.test", 22, "alice", "sftp", "example.test", 22, "alice") {
		t.Fatal("same account was not matched")
	}
	if AccountMatches("sftp", "example.test", 22, "alice", "sftp", "example.test", 22, "Alice") {
		t.Fatal("username case change crossed account boundary")
	}
}

func TestPrivateKeyMatchesRequiresSameNonEmptyKey(t *testing.T) {
	const key = `/home/alice/.ssh/id_ed25519`
	if !PrivateKeyMatches("sftp", "example.test", 22, "alice", key, "sftp", "example.test", 22, "alice", key) {
		t.Fatal("same private-key path was not matched")
	}
	if PrivateKeyMatches("sftp", "example.test", 22, "alice", key, "sftp", "example.test", 22, "alice", "") {
		t.Fatal("empty key path matched stored key")
	}
	if PrivateKeyMatches("sftp", "example.test", 22, "alice", key, "sftp", "example.test", 22, "alice", `/home/alice/.ssh/other`) {
		t.Fatal("different key path matched")
	}
}
