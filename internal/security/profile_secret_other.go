//go:build !darwin

package security

// PersistentProfileSecretToRuntime is a no-op ownership adapter on platforms
// whose existing protected profile blob is already consumable by the runtime.
// macOS overrides this with a Keychain-to-runtime capability conversion.
func PersistentProfileSecretToRuntime(encoded string) (string, bool, error) {
	return encoded, false, nil
}

func ProtectRuntimeBytes(data []byte) (string, error) { return ProtectBytes(data) }
