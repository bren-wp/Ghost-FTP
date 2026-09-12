//go:build darwin

package security

// macOS runtime secrets use the existing peer-credential checked Unix-socket
// broker because SFTP AskPass runs out of process. This stays deliberately
// ephemeral and separate from durable Keychain-backed profile storage.
func ProtectRuntimeBytes(value []byte) (string, error) {
	return ProtectBytes(value)
}

func ProtectRuntimeString(value string) (string, error) {
	return ProtectString(value)
}

func UnprotectRuntimeBytes(encoded string) ([]byte, error) {
	return UnprotectBytes(encoded)
}

func ForgetRuntimeSecret(encoded string) {
	ForgetProtectedSecret(encoded)
}
