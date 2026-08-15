//go:build windows

package config

import "brendigo.com/byftp/internal/security"

func protectProfileData(data []byte) (string, error) {
	return security.ProtectBytes(data)
}

func unprotectProfileData(encoded string) ([]byte, error) {
	return security.UnprotectBytes(encoded)
}
