package remote

import (
	"slices"
	"testing"
)

func TestCurlBandwidthConfigLine(t *testing.T) {
	if got := curlBandwidthConfigLine(0); got != "" {
		t.Fatalf("unlimited curl transfer must not emit limit-rate, got %q", got)
	}
	if got, want := curlBandwidthConfigLine(1048576), `limit-rate = "1048576"`; got != want {
		t.Fatalf("curl limit line = %q, want %q", got, want)
	}
}

func TestSFTPLimitConversionNeverRoundsAboveBudget(t *testing.T) {
	if got := sftpLimitKbitPerSecond(0); got != 0 {
		t.Fatalf("unlimited SFTP transfer must not emit -l, got %d", got)
	}
	if got, want := sftpLimitKbitPerSecond(125000), int64(1000); got != want {
		t.Fatalf("SFTP conversion = %d Kbit/s, want %d", got, want)
	}
	if got := sftpLimitKbitPerSecond(128); got != 1 {
		t.Fatalf("small positive SFTP budget must remain throttled, got %d", got)
	}
}

func TestSFTPBandwidthArgsPreserveStrictSecurityControls(t *testing.T) {
	s := &SFTP{sshConfig: "strict.conf", sessionHost: "GhostFTP-test"}
	args := s.commandArgsWithBandwidth(125000)
	for _, required := range []string{
		"-oStrictHostKeyChecking=yes",
		"-oGlobalKnownHostsFile=none",
		"-oVerifyHostKeyDNS=no",
		"-oUpdateHostKeys=no",
		"-oProxyCommand=none",
		"-oProxyJump=none",
		"-oClearAllForwardings=yes",
	} {
		if !slices.Contains(args, required) {
			t.Fatalf("bandwidth args dropped security control %q: %v", required, args)
		}
	}
	if len(args) < 3 || args[len(args)-3] != "-l" || args[len(args)-2] != "1000" || args[len(args)-1] != "GhostFTP-test" {
		t.Fatalf("SFTP bandwidth limit must be inserted directly before host: %v", args)
	}
}

func TestSFTPUnlimitedArgsDoNotAddLimiter(t *testing.T) {
	s := &SFTP{sshConfig: "strict.conf", sessionHost: "GhostFTP-test"}
	args := s.commandArgsWithBandwidth(0)
	if slices.Contains(args, "-l") {
		t.Fatalf("unlimited SFTP transfer unexpectedly contains -l: %v", args)
	}
}
