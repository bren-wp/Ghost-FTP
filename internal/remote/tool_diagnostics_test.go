package remote

import "testing"

func TestCurlToolErrorUserErrorKind(t *testing.T) {
	tests := []struct {
		name       string
		code       int
		diagnostic string
		want       string
	}{
		{name: "resolve code", code: 6, diagnostic: "opaque curl failure", want: "resolve"},
		{name: "timeout code", code: 28, diagnostic: "opaque curl failure", want: "timeout"},
		{name: "tls verification code", code: 60, diagnostic: "opaque curl failure", want: "tls"},
		{name: "login code", code: 67, diagnostic: "opaque curl failure", want: "auth"},
		{name: "remote missing code", code: 78, diagnostic: "opaque curl failure", want: "not_found"},
		{name: "partial transfer code", code: 18, diagnostic: "opaque curl failure", want: "connection_lost"},
		{name: "ftp limit reply wins", code: 7, diagnostic: "421 Too many connections from this IP", want: "ftp_limit"},
		{name: "ftp data reply wins", code: 7, diagnostic: "425 Can't open data connection", want: "ftp_data"},
		{name: "ftp login reply wins", code: 7, diagnostic: "530 Login incorrect", want: "auth"},
		{name: "refused message", code: 7, diagnostic: "Failed to connect: Connection refused", want: "refused"},
		{name: "unknown code stays unclassified", code: 99, diagnostic: "opaque curl failure", want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyToolDiagnostic("curl", tc.code, tc.diagnostic); got != tc.want {
				t.Fatalf("classifyToolDiagnostic()=%q want %q", got, tc.want)
			}
		})
	}
}

func TestSFTPToolErrorUserErrorKind(t *testing.T) {
	tests := []struct {
		name       string
		diagnostic string
		want       string
	}{
		{name: "host key changed", diagnostic: "WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!", want: "hostkey_changed"},
		{name: "host key verification", diagnostic: "Host key verification failed.", want: "hostkey_changed"},
		{name: "resolve", diagnostic: "ssh: Could not resolve hostname server.invalid: Name or service not known", want: "resolve"},
		{name: "refused", diagnostic: "ssh: connect to host example port 22: Connection refused", want: "refused"},
		{name: "timeout", diagnostic: "ssh: connect to host example port 22: Connection timed out", want: "timeout"},
		{name: "connection lost", diagnostic: "Connection reset by peer", want: "connection_lost"},
		{name: "missing remote file", diagnostic: "stat remote: No such file or directory", want: "not_found"},
		{name: "public key auth", diagnostic: "Permission denied (publickey,password).", want: "auth"},
		{name: "keyboard interactive auth", diagnostic: "Permission denied (keyboard-interactive).", want: "auth"},
		{name: "bad passphrase", diagnostic: "Load key C:/secret/key: incorrect passphrase supplied to decrypt private key", want: "sftp_settings"},
		{name: "invalid key format", diagnostic: "Load key C:/secret/key: invalid format", want: "sftp_settings"},
		{name: "libcrypto key failure", diagnostic: "Load key /home/user/key: error in libcrypto", want: "sftp_settings"},
		{name: "private key permissions", diagnostic: "WARNING: UNPROTECTED PRIVATE KEY FILE! Bad permissions: ignore key", want: "sftp_settings"},
		{name: "generic permission", diagnostic: "remote open: Permission denied", want: "permission"},
		{name: "unknown output stays unclassified", diagnostic: "opaque sftp failure", want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyToolDiagnostic("sftp", 255, tc.diagnostic); got != tc.want {
				t.Fatalf("classifyToolDiagnostic()=%q want %q", got, tc.want)
			}
		})
	}
}

func TestToolErrorUnknownToolHasNoUserClassification(t *testing.T) {
	if got := classifyToolDiagnostic("other", 6, "Could not resolve host"); got != "" {
		t.Fatalf("classifyToolDiagnostic()=%q", got)
	}
}

func TestToolDiagnosticRetryClassification(t *testing.T) {
	tests := []struct {
		name       string
		tool       string
		code       int
		diagnostic string
		want       bool
	}{
		{name: "curl connect code", tool: "curl", code: 7, diagnostic: "opaque", want: true},
		{name: "curl timeout code", tool: "curl", code: 28, diagnostic: "opaque", want: true},
		{name: "curl auth code", tool: "curl", code: 67, diagnostic: "connection reset", want: false},
		{name: "sftp reset", tool: "sftp", code: 255, diagnostic: "Connection reset by peer", want: true},
		{name: "sftp resolve", tool: "sftp", code: 255, diagnostic: "Could not resolve hostname secret-host.example", want: true},
		{name: "sftp auth", tool: "sftp", code: 255, diagnostic: "Permission denied (publickey,password)", want: false},
		{name: "sftp missing", tool: "sftp", code: 255, diagnostic: "No such file", want: false},
		{name: "unknown transient", tool: "other", code: 1, diagnostic: "server returned nothing", want: true},
		{name: "unknown opaque", tool: "other", code: 1, diagnostic: "opaque failure", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyToolRetryable(tc.tool, tc.code, tc.diagnostic); got != tc.want {
				t.Fatalf("classifyToolRetryable()=%v want %v", got, tc.want)
			}
		})
	}
}
