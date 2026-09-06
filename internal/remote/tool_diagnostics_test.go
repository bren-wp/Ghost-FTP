package remote

import "testing"

func TestCurlToolErrorUserErrorKind(t *testing.T) {
	tests := []struct {
		name    string
		code    int
		message string
		want    string
	}{
		{name: "resolve code", code: 6, message: "opaque curl failure", want: "resolve"},
		{name: "timeout code", code: 28, message: "opaque curl failure", want: "timeout"},
		{name: "tls verification code", code: 60, message: "opaque curl failure", want: "tls"},
		{name: "login code", code: 67, message: "opaque curl failure", want: "auth"},
		{name: "remote missing code", code: 78, message: "opaque curl failure", want: "not_found"},
		{name: "partial transfer code", code: 18, message: "opaque curl failure", want: "connection_lost"},
		{name: "ftp limit reply wins", code: 7, message: "421 Too many connections from this IP", want: "ftp_limit"},
		{name: "ftp data reply wins", code: 7, message: "425 Can't open data connection", want: "ftp_data"},
		{name: "ftp login reply wins", code: 7, message: "530 Login incorrect", want: "auth"},
		{name: "refused message", code: 7, message: "Failed to connect: Connection refused", want: "refused"},
		{name: "unknown code stays unclassified", code: 99, message: "opaque curl failure", want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := &toolError{tool: "curl", code: tc.code, message: tc.message}
			if got := err.UserErrorKind(); got != tc.want {
				t.Fatalf("UserErrorKind()=%q want %q", got, tc.want)
			}
		})
	}
}

func TestSFTPToolErrorUserErrorKind(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
	}{
		{name: "host key changed", message: "WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!", want: "hostkey_changed"},
		{name: "host key verification", message: "Host key verification failed.", want: "hostkey_changed"},
		{name: "resolve", message: "ssh: Could not resolve hostname server.invalid: Name or service not known", want: "resolve"},
		{name: "refused", message: "ssh: connect to host example port 22: Connection refused", want: "refused"},
		{name: "timeout", message: "ssh: connect to host example port 22: Connection timed out", want: "timeout"},
		{name: "connection lost", message: "Connection reset by peer", want: "connection_lost"},
		{name: "missing remote file", message: "stat remote: No such file or directory", want: "not_found"},
		{name: "public key auth", message: "Permission denied (publickey,password).", want: "auth"},
		{name: "keyboard interactive auth", message: "Permission denied (keyboard-interactive).", want: "auth"},
		{name: "bad passphrase", message: "Load key C:/secret/key: incorrect passphrase supplied to decrypt private key", want: "sftp_settings"},
		{name: "invalid key format", message: "Load key C:/secret/key: invalid format", want: "sftp_settings"},
		{name: "libcrypto key failure", message: "Load key /home/user/key: error in libcrypto", want: "sftp_settings"},
		{name: "private key permissions", message: "WARNING: UNPROTECTED PRIVATE KEY FILE! Bad permissions: ignore key", want: "sftp_settings"},
		{name: "generic permission", message: "remote open: Permission denied", want: "permission"},
		{name: "unknown output stays unclassified", message: "opaque sftp failure", want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := &toolError{tool: "sftp", code: 255, message: tc.message}
			if got := err.UserErrorKind(); got != tc.want {
				t.Fatalf("UserErrorKind()=%q want %q", got, tc.want)
			}
		})
	}
}

func TestToolErrorUnknownToolHasNoUserClassification(t *testing.T) {
	err := &toolError{tool: "other", code: 6, message: "Could not resolve host"}
	if got := err.UserErrorKind(); got != "" {
		t.Fatalf("UserErrorKind()=%q", got)
	}
}
