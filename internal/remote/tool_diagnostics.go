package remote

import "strings"

// UserErrorKind exposes only the stable, non-secret semantic category captured
// when the child-process failure was created. Raw diagnostic text is classified
// transiently and is not retained in toolError.
func (e *toolError) UserErrorKind() string {
	if e == nil {
		return ""
	}
	return e.kind
}

func normalizeToolName(tool string) string {
	switch strings.ToLower(strings.TrimSpace(tool)) {
	case "curl":
		return "curl"
	case "ssh":
		return "ssh"
	case "sftp":
		return "sftp"
	default:
		return ""
	}
}

func normalizeDiagnostic(diagnostic string) string {
	return strings.ToLower(strings.Join(strings.Fields(diagnostic), " "))
}

func classifyToolDiagnostic(tool string, code int, diagnostic string) string {
	s := normalizeDiagnostic(diagnostic)

	switch normalizeToolName(tool) {
	case "curl":
		// Protocol replies are more specific than curl's process exit code and
		// should win when both are available.
		switch {
		case containsDiagnosticMarker(s, "500 ", "500-", "502 ", "502-", "504 ", "504-", "unknown command", "command not understood", "not implemented", "unsupported command"):
			return "ftp_unsupported"
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

		switch code {
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

// classifyToolRetryable captures the exact retry decision while the bounded
// child-process diagnostic is still in scope. The raw text does not need to
// survive inside the returned error object.
func classifyToolRetryable(tool string, code int, diagnostic string) bool {
	if normalizeToolName(tool) == "curl" {
		// Preserve the established curl contract: exit code is authoritative for
		// automatic retry, independent of the diagnostic wording.
		switch code {
		case 6, 7, 18, 28, 52, 55, 56:
			return true
		default:
			return false
		}
	}

	msg := strings.ToLower(diagnostic)
	for _, marker := range []string{
		"permission denied", "access denied", "authentication failed", "login denied", "login incorrect",
		"host key verification failed", "fingerprint", "no such file", "not found", "not a directory",
		"is a directory", "file exists", "already exists", "invalid argument", "unsupported", "not implemented",
	} {
		if strings.Contains(msg, marker) {
			return false
		}
	}
	for _, marker := range []string{
		"connection reset", "connection timed out", "operation timed out", "connection timeout",
		"failed to connect", "couldn't connect", "could not connect", "connection refused", "connection closed",
		"broken pipe", "network is unreachable", "no route to host", "temporary failure in name resolution",
		"could not resolve", "recv failure", "send failure", "partial file", "server returned nothing",
	} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}

func containsDiagnosticMarker(s string, markers ...string) bool {
	for _, marker := range markers {
		if strings.Contains(s, marker) {
			return true
		}
	}
	return false
}
