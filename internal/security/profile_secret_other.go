//go:build !darwin

package security

// Non-Darwin platforms keep their existing protected-profile representation.
// Windows DPAPI blobs and the existing Linux compatibility path are already in
// the runtime format expected by their transport implementation.
func PersistentProfileSecretToRuntime(encoded string) (string, error) {
	return encoded, nil
}
