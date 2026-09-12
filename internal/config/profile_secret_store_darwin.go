//go:build darwin

package config

import "github.com/bren-wp/Ghost-FTP/internal/security"

const storedProfileSecretMarker = "profile-secret-v1\x00"

func protectStoredProfileSecret(value string) (string, error) {
	payload := append([]byte(storedProfileSecretMarker), []byte(value)...)
	defer security.WipeBytes(payload)
	return security.ProtectPersistentProfileBytes(payload)
}
