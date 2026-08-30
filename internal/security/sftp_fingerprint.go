package security

import (
	"encoding/base64"
	"errors"
	"strings"
	"unicode/utf8"
)

// ValidateSFTPFingerprint accepts the canonical OpenSSH SHA256 fingerprint
// representation only. The payload must decode to exactly one SHA-256 digest.
func ValidateSFTPFingerprint(fingerprint string) error {
	if !utf8.ValidString(fingerprint) || !strings.HasPrefix(fingerprint, "SHA256:") || strings.ContainsAny(fingerprint, "\x00\r\n\t ") {
		return errors.New("SFTP fingerprint je neispravan")
	}
	encoded := strings.TrimPrefix(fingerprint, "SHA256:")
	if encoded == "" {
		return errors.New("SFTP fingerprint je neispravan")
	}
	digest, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil || len(digest) != 32 {
		return errors.New("SFTP fingerprint je neispravan")
	}
	return nil
}
