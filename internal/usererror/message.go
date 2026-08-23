package usererror

import (
	"context"
	"errors"
	"strings"

	"github.com/bren-wp/by-ftp/internal/i18n"
)

// Message keeps source compatibility for callers that do not yet carry an
// explicit locale. English is the canonical/default user-facing language.
func Message(err error, fallback string) string {
	return MessageFor(i18n.DefaultLanguage, err, fallback)
}

// MessageFor converts low-level protocol/OS/tooling errors into concise,
// localized end-user messages without exposing command-line or library detail.
func MessageFor(language string, err error, fallback string) string {
	if err == nil {
		return ""
	}
	language = i18n.Normalize(language)
	s := strings.ToLower(strings.Join(strings.Fields(err.Error()), " "))

	// Specific connection lifecycle states must win over a joined generic
	// context deadline so the UI describes what is actually still happening.
	if containsAny(s, "prethodna veza se još sigurno zatvara", "previous connection is still closing safely") {
		return i18n.T(language, "error.session_closing")
	}
	if containsAny(s, "sigurno zatvaranje veze još traje", "connection is still closing safely") {
		return i18n.T(language, "error.disconnect_closing")
	}
	if errors.Is(err, context.Canceled) {
		return i18n.T(language, "error.cancelled")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return i18n.T(language, "error.timeout")
	}

	switch {
	case containsAny(s, "otisak sftp host ključa se promijenio", "fingerprint se promijenio", "host key verification failed", "sftp host-key fingerprint changed"):
		return i18n.T(language, "error.hostkey_changed")
	case containsAny(s, "sftp podrška nije dostupna u sustavu windows", "sftp komponenta nije pronađena", "openssh client nije instaliran", "nedostaje sftp.exe", "nedostaje ssh-keyscan.exe", "nedostaje ssh-keygen.exe", "sftp support is unavailable", "openssh client is not installed", "missing sftp.exe", "missing ssh-keyscan.exe", "missing ssh-keygen.exe"):
		return i18n.T(language, "error.sftp_unavailable")
	case containsAny(s, "nije moguće dohvatiti sftp host ključ", "poslužitelj nije vratio ssh host ključ", "could not retrieve sftp host key", "server did not return an ssh host key"):
		return i18n.T(language, "error.sftp_hostkey_missing")
	case containsAny(s, "authentication failed", "permission denied (publickey", "permission denied (password", "permission denied, please try again", "login incorrect", "access denied", "530 login", "530 user", "530 not logged", "authentication rejected"):
		return i18n.T(language, "error.auth")
	case containsAny(s, "421 too many connections", "421 service not available", "421 connection", "too many connections"):
		return i18n.T(language, "error.ftp_limit")
	case containsAny(s, "425 can't open data connection", "425 cannot open data connection", "425 failed to establish connection", "426 connection closed", "426 transfer aborted"):
		return i18n.T(language, "error.ftp_data")
	case containsAny(s, "could not resolve host", "name or service not known", "temporary failure in name resolution", "no such host", "host not found"):
		return i18n.T(language, "error.resolve")
	case containsAny(s, "connection refused", "actively refused"):
		return i18n.T(language, "error.refused")
	case containsAny(s, "timed out", "timeout", "operation timed out"):
		return i18n.T(language, "error.timeout")
	case containsAny(s, "connection reset", "connection closed", "broken pipe", "connection aborted", "network is unreachable", "no route to host"):
		return i18n.T(language, "error.connection_lost")
	case containsAny(s, "certificate", "ssl certificate", "tls", "schannel"):
		return i18n.T(language, "error.tls")
	case containsAny(s, "disk full", "no space left", "insufficient disk space", "quota exceeded", "552"):
		return i18n.T(language, "error.disk")
	case containsAny(s, "permission denied", "access is denied", "550 permission", "553 permission"):
		return i18n.T(language, "error.permission")
	case containsAny(s, "no such file", "not found", "550 file unavailable", "550 failed to open"):
		return i18n.T(language, "error.not_found")
	case containsAny(s, "already exists", "file exists", "ciljna stavka već postoji", "target item already exists"):
		return i18n.T(language, "error.exists")
	case containsAny(s, "nije uspostavljena veza", "not connected", "connection is not established"):
		return i18n.T(language, "error.not_connected")
	case containsAny(s, "red prijenosa je prevelik", "transfer queue is full", "transfer queue is too large"):
		return i18n.T(language, "error.queue_full")
	case containsAny(s, "previše stavki", "predubok", "too many items", "too deep"):
		return i18n.T(language, "error.structure_large")
	case containsAny(s, "neispravan port", "port mora biti", "invalid port", "port must be"):
		return i18n.T(language, "error.invalid_port")
	case containsAny(s, "neispravan poslužitelj", "invalid server", "invalid host"):
		return i18n.T(language, "error.invalid_host")
	case containsAny(s, "neispravno korisničko ime", "invalid username"):
		return i18n.T(language, "error.invalid_user")
	case containsAny(s, "neispravan naziv datoteke ili mape", "invalid file or folder name"):
		return i18n.T(language, "error.invalid_name")
	}

	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		fallback = i18n.T(language, "error.generic")
	}
	return fallback
}

func containsAny(s string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(s, value) {
			return true
		}
	}
	return false
}
