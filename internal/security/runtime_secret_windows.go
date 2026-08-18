//go:build windows

package security

// Windows runtime tajne koriste isti DPAPI model kao i spremljene vjerodajnice.
// Time aktivna sesija nikada ne mora držati plaintext lozinku dulje od jednog
// poziva prema vanjskom mrežnom alatu.
func ProtectRuntimeString(value string) (string, error) {
	return ProtectString(value)
}

func UnprotectRuntimeBytes(encoded string) ([]byte, error) {
	return UnprotectBytes(encoded)
}

func ForgetRuntimeSecret(string) {}
