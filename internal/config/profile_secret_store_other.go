//go:build !darwin

package config

import "github.com/bren-wp/Ghost-FTP/internal/security"

func protectStoredProfileSecret(value string) (string, error) {
	return security.ProtectString(value)
}
