package desktop

import "testing"

func TestParseLaunchTargetAcceptsSanitizedConnectionMetadata(t *testing.T) {
	target, err := ParseLaunchTarget("ghostftp://open?protocol=sftp&host=example.com&port=2222&username=alice%40example.com&path=%2Fhome%2Falice")
	if err != nil {
		t.Fatalf("ParseLaunchTarget returned error: %v", err)
	}
	if target.Protocol != "sftp" || target.Host != "example.com" || target.Port != "2222" {
		t.Fatalf("unexpected connection target: %#v", target)
	}
	if target.Username != "alice@example.com" || target.Path != "/home/alice" {
		t.Fatalf("unexpected decoded metadata: %#v", target)
	}
}

func TestParseLaunchTargetAppliesSafeDefaults(t *testing.T) {
	target, err := ParseLaunchTarget("ghostftp://open?protocol=ftps&host=files.example.com")
	if err != nil {
		t.Fatalf("ParseLaunchTarget returned error: %v", err)
	}
	if target.Port != "21" || target.Path != "/" || target.Username != "" {
		t.Fatalf("unexpected defaults: %#v", target)
	}
}

func TestParseLaunchTargetRejectsCredentialChannels(t *testing.T) {
	cases := []string{
		"ghostftp://open?protocol=sftp&host=example.com&password=secret",
		"ghostftp://open?protocol=sftp&host=example.com&passphrase=secret",
		"ghostftp://user:secret@open?protocol=sftp&host=example.com",
		"ghostftp://open?protocol=sftp&host=example.com#secret",
	}
	for _, raw := range cases {
		if _, err := ParseLaunchTarget(raw); err == nil {
			t.Fatalf("expected credential-bearing target to be rejected: %q", raw)
		}
	}
}

func TestParseLaunchTargetRejectsMalformedOrAmbiguousInput(t *testing.T) {
	cases := []string{
		"https://open?protocol=sftp&host=example.com",
		"ghostftp://other?protocol=sftp&host=example.com",
		"ghostftp://open?protocol=ssh&host=example.com",
		"ghostftp://open?protocol=sftp&protocol=ftp&host=example.com",
		"ghostftp://open?protocol=sftp&host=example.com&host=evil.example",
		"ghostftp://open?protocol=sftp&host=example.com&port=0",
		"ghostftp://open?protocol=sftp&host=example.com&port=65536",
		"ghostftp://open?protocol=sftp&host=example.com&port=022",
		"ghostftp://open?protocol=sftp&host=example.com&path=relative",
		"ghostftp://open?protocol=sftp&host=example.com&path=%2Fsafe%0Aevil",
		"ghostftp://open/?protocol=sftp&host=example.com&extra=value",
	}
	for _, raw := range cases {
		if _, err := ParseLaunchTarget(raw); err == nil {
			t.Fatalf("expected invalid target to be rejected: %q", raw)
		}
	}
}

func TestConfigureInitialLaunchTargetIsConsumedOnce(t *testing.T) {
	if err := ConfigureInitialLaunchTarget("ghostftp://open?protocol=ftp&host=example.com"); err != nil {
		t.Fatalf("ConfigureInitialLaunchTarget returned error: %v", err)
	}
	first, ok := takeInitialLaunchTarget()
	if !ok || first.Host != "example.com" {
		t.Fatalf("expected configured target, got %#v, %v", first, ok)
	}
	if _, ok := takeInitialLaunchTarget(); ok {
		t.Fatal("launch target must be consumed exactly once")
	}
}
