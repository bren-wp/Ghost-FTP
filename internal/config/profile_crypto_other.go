//go:build !windows

package config

import "encoding/base64"

// The desktop product is Windows-only. This compatibility codec keeps unit
// tests and source inspection usable on non-Windows builders; Windows release
// builds use DPAPI in profile_crypto_windows.go.
func protectProfileData(data []byte) (string, error) {
	return base64.StdEncoding.EncodeToString(data), nil
}

func unprotectProfileData(encoded string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(encoded)
}
