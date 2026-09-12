//go:build darwin

package config

import (
	"bytes"
	"errors"

	"github.com/bren-wp/Ghost-FTP/internal/security"
)

const darwinProfileEnvelopeMarker = "profile-envelope-v1\x00"

func protectProfileData(data []byte, _ string) (string, error) {
	payload := make([]byte, 0, len(darwinProfileEnvelopeMarker)+len(data))
	payload = append(payload, darwinProfileEnvelopeMarker...)
	payload = append(payload, data...)
	defer security.WipeBytes(payload)
	return security.ProtectPersistentProfileBytes(payload)
}

func unprotectProfileData(encoded, _ string) ([]byte, error) {
	plain, err := security.UnprotectPersistentProfileBytes(encoded)
	if err != nil {
		return nil, err
	}
	marker := []byte(darwinProfileEnvelopeMarker)
	if !bytes.HasPrefix(plain, marker) {
		security.WipeBytes(plain)
		return nil, errors.New("macOS saved profile envelope is malformed")
	}
	out := append([]byte(nil), plain[len(marker):]...)
	security.WipeBytes(plain)
	return out, nil
}

func profileDataNeedsMigration(string) bool { return false }
