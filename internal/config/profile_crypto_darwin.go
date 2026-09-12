//go:build darwin

package config

import "errors"

var errDarwinPersistentProfilesUnavailable = errors.New("saved profiles are unavailable until macOS Keychain protection is enabled")

// protectProfileData deliberately fails closed on macOS for now. The native
// Quick Connect path does not persist connection secrets, and the in-memory
// AskPass capability broker must never be reused as durable profile storage.
// A later Site Manager parity change will replace this gate with Keychain-backed
// profile protection before saved macOS profiles are enabled.
func protectProfileData([]byte, string) (string, error) {
	return "", errDarwinPersistentProfilesUnavailable
}

// Existing protected profile envelopes are not interpreted with a weaker or
// platform-incompatible codec. Until Keychain-backed storage lands, macOS
// refuses to unlock durable profile data rather than silently downgrading it.
func unprotectProfileData(string, string) ([]byte, error) {
	return nil, errDarwinPersistentProfilesUnavailable
}

func profileDataNeedsMigration(string) bool { return false }
