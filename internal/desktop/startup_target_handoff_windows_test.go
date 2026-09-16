//go:build windows

package desktop

import "testing"

func TestStartupTargetHandoffRoundTrip(t *testing.T) {
	// Keep this test isolated from any startup target left by another test.
	_, _ = takeStartupTarget()

	stop := StartStartupTargetReceiver()
	defer stop()

	raw := "ghostftp://connect?protocol=sftp&host=example.com&port=2222&username=alice&path=%2Fhome%2Falice"
	if !ForwardStartupTarget(raw) {
		t.Fatal("expected local Windows startup target handoff to succeed")
	}

	target, ok := takeStartupTarget()
	if !ok {
		t.Fatal("expected forwarded startup target")
	}
	if target.Protocol != "sftp" || target.Host != "example.com" || target.Port != 2222 || target.Username != "alice" || target.Path != "/home/alice" {
		t.Fatalf("unexpected forwarded target: %#v", target)
	}
}

func TestStartupTargetHandoffRejectsSensitivePayload(t *testing.T) {
	if ForwardStartupTarget("ghostftp://connect?protocol=sftp&host=example.com&password=secret") {
		t.Fatal("sensitive startup target must be rejected before Windows handoff")
	}
}
