//go:build windows

package config

import "github.com/bren-wp/Ghost-FTP/internal/security"

func protectProfileData(data []byte, _ string) (string, error) {
	return security.ProtectBytes(data)
}

func unprotectProfileData(encoded, _ string) ([]byte, error) {
	return security.UnprotectBytes(encoded)
}

func profileDataNeedsMigration(string) bool { return false }
