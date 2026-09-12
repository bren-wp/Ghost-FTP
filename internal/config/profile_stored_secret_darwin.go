//go:build darwin

package config

import "github.com/bren-wp/Ghost-FTP/internal/security"

const storedProfileSecretPurpose = "profile-secret-v1"

func protectStoredProfileSecret(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	plain := []byte(value)
	defer security.WipeBytes(plain)
	return security.ProtectPersistentProfileBytes(plain, storedProfileSecretPurpose)
}
