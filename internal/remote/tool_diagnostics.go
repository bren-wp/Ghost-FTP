package remote

import "strings"

// UserErrorKind exposes only a stable, non-secret semantic category to the UI
// error mapper. Raw child-process output remains inside the remote package and
// must never be rendered directly to users.
func (e *toolError) UserErrorKind() string {
	if e == nil {
		return ""
	}
	s := strings.ToLower(strings.Join(strings.Fields(e.message), " "))

	switch strings.ToLower(strings.TrimSpace(e.tool)) {
	case "curl":
		// Protocol replies are more specific than curl's process exit code and
		// should win when both are available.
		switch {
		case containsDiagnosticMarker(s, "421 too many connections", "421 service not available", "421 connection", "too many connections"):
			return "ftp_limit"
		case containsDiagnosticMarker(s, "425 can't open data connection", "425 cannot open data connection", "425 failed to establish connection", "426 connection closed", "426 transfer aborted"):
			return "ftp_data"
		case containsDiagnosticMarker(s, "530 login", "530 user", "530 not logged", "login denied", "login incorrect", "authentication rejected"):
			return "auth"
		case containsDiagnosticMarker(s, "552 quota", "552 disk", "quota exceeded", "no space left", "disk full"):
			return "disk"
		case containsDiagnosticMarker(s, "connection refused", "actively refused"):
			return "refused"
		case containsDiagnosticMarker(s, "could not resolve host", "name or service not known", "temporary failure in name resolution", "no such host", "host not found"):
			return "resolve"
		case containsDiagnosticMarker(s, "timed out", "timeout", "operation timed out"):
			return "timeout"
		case containsDiagnosticMarker(s, "certificate", "ssl certificate", "tls", "schannel"):
			return "tls"
		}

		switch e.code {
		case 6:
			return "resolve"
		case 9:
			return "permission"
		case 18, 52, 55, 56:
			return "connection_lost"
		case 28:
			return "timeout"
		case 35, 51, 58, 59, 60, 64, 66, 77, 80, 82, 83, 90, 91:
			return "tls"
		case 67:
			return "auth"
		case 78:
			return "not_found"
		}

	case "sftp":
		switch {
		case containsDiagnosticMarker(s,
			"remote host identification has changed",
			"host key verification failed",
			"sftp host-key fingerprint changed",
			"fingerprint se promijenio",
			"otisak sftp host ključa se promijenio",
		):
			return "hostkey_changed"
		case containsDiagnosticMarker(s, "could not resolve hostname", "name or service not known", "temporary failure in name resolution", "no such host", "host not found"):
			return "resolve"
		case containsDiagnosticMarker(s, "connection refused", "actively refused"):
			return "refused"
		case containsDiagnosticMarker(s, "connection timed out", "operation timed out", "timed out"):
			return "timeout"
		case containsDiagnosticMarker(s, "connection reset", "connection closed", "broken pipe", "connection aborted", "network is unreachable", "no route to host"):
			return "connection_lost"
		case containsDiagnosticMarker(s, "no such file", "not found", "couldn't stat remote file", "could not stat remote file"):
			return "not_found"
		case containsDiagnosticMarker(s,
			"incorrect passphrase",
			"bad passphrase",
			"error in libcrypto",
			"unprotected private key file",
			"bad permissions: ignore key",
		) || (strings.Contains(s, "load key") && strings.Contains(s, "invalid format")):
			// Existing localized SFTP copy is intentionally used here rather than
			// exposing the OpenSSH key path, parser details or passphrase prompt.
			return "sftp_settings"
		case containsDiagnosticMarker(s,
			"authentication failed",
			"authentication rejected",
			"permission denied (publickey",
			"permission denied (password",
			"permission denied (keyboard-interactive",
			"permission denied, please try again",
			"too many authentication failures",
		):
			return "auth"
		case containsDiagnosticMarker(s, "permission denied", "access denied"):
			return "permission"
		}
	}

	return ""
}

func containsDiagnosticMarker(s string, markers ...string) bool {
	for _, marker := range markers {
		if strings.Contains(s, marker) {
			return true
		}
	}
	return false
}
