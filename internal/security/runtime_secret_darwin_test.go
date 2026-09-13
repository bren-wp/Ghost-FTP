//go:build darwin

package security

import "testing"

func TestDarwinRuntimeSecretRoundTripAndForget(t *testing.T) {
	token, err := ProtectRuntimeString("credential")
	if err != nil {
		t.Fatalf("ProtectRuntimeString: %v", err)
	}
	if token == "" {
		t.Fatal("ProtectRuntimeString returned an empty capability")
	}

	first, err := UnprotectRuntimeBytes(token)
	if err != nil {
		t.Fatalf("UnprotectRuntimeBytes: %v", err)
	}
	if got := string(first); got != "credential" {
		WipeBytes(first)
		t.Fatalf("round trip mismatch: %q", got)
	}
	first[0] = 'X'
	WipeBytes(first)

	second, err := UnprotectRuntimeBytes(token)
	if err != nil {
		t.Fatalf("second UnprotectRuntimeBytes: %v", err)
	}
	if got := string(second); got != "credential" {
		WipeBytes(second)
		t.Fatalf("broker value was mutated through returned buffer: %q", got)
	}
	WipeBytes(second)

	ForgetRuntimeSecret(token)
	if secret, err := UnprotectRuntimeBytes(token); err == nil {
		WipeBytes(secret)
		t.Fatal("forgotten Darwin runtime capability remained readable")
	}
}
