package clientmode

import "testing"

func TestParseAndProtocolIsolation(t *testing.T) {
	cases := []struct {
		input string
		mode  Mode
	}{
		{"suite", Suite}, {"FTP", FTP}, {"sftp", SFTP}, {"ssh", SSH}, {"s3", S3}, {"unknown", Suite},
	}
	for _, tc := range cases {
		if got := Parse(tc.input); got != tc.mode {
			t.Fatalf("Parse(%q)=%q, očekivano %q", tc.input, got, tc.mode)
		}
	}
	if !FTP.AllowsProtocol("ftp") || !FTP.AllowsProtocol("ftps") || !FTP.AllowsProtocol("ftpsi") || FTP.AllowsProtocol("sftp") {
		t.Fatal("FTP client ne izolira FTP/FTPS protokole")
	}
	if !SFTP.AllowsProtocol("sftp") || SFTP.AllowsProtocol("ftp") {
		t.Fatal("SFTP client ne izolira SFTP protokol")
	}
	if SSH.AllowsProtocol("sftp") || S3.AllowsProtocol("ftp") {
		t.Fatal("SSH/S3 ne smiju koristiti file-transfer protocol selector")
	}
}

func TestModeIdentityIsSeparated(t *testing.T) {
	seen := map[string]bool{}
	for _, mode := range []Mode{Suite, FTP, SFTP, SSH, S3} {
		if mode.ProductName() == "" || mode.InstanceKey() == "" || mode.Slug() == "" {
			t.Fatalf("mode %q nema potpuni identitet", mode)
		}
		if seen[mode.InstanceKey()] {
			t.Fatalf("duplicirani instance key za %q", mode)
		}
		seen[mode.InstanceKey()] = true
	}
}
