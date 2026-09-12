//go:build darwin

package security

// macOS external transport helpers and in-process curl operations share the
// same peer-credential-checked Unix-socket runtime broker. Durable Keychain
// profile material is converted into this ephemeral capability only after
// shared profile binding succeeds.
func ProtectRuntimeString(value string) (string, error) {
	return ProtectString(value)
}

func UnprotectRuntimeBytes(encoded string) ([]byte, error) {
	return UnprotectBytes(encoded)
}

func ForgetRuntimeSecret(encoded string) {
	ForgetProtectedSecret(encoded)
}
