//go:build darwin

package config

import "github.com/bren-wp/Ghost-FTP/internal/security"

const darwinProfileEnvelopePurpose = "profile-envelope-v1"

func protectProfileData(data []byte, _ string) (string, error) {
	return security.ProtectPersistentProfileBytes(data, darwinProfileEnvelopePurpose)
}

func unprotectProfileData(encoded, _ string) ([]byte, error) {
	return security.UnprotectPersistentProfileBytes(encoded, darwinProfileEnvelopePurpose)
}

func profileDataNeedsMigration(string) bool { return false }
