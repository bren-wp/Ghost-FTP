//go:build !windows

package security

import (
	"fmt"
	"testing"
)

func isolateRuntimeSecrets(t *testing.T) {
	t.Helper()
	runtimeValues.Lock()
	previous := runtimeValues.values
	runtimeValues.values = make(map[string][]byte)
	runtimeValues.Unlock()

	t.Cleanup(func() {
		runtimeValues.Lock()
		current := runtimeValues.values
		runtimeValues.values = previous
		runtimeValues.Unlock()
		for _, value := range current {
			WipeBytes(value)
		}
	})
}

func TestRuntimeSecretRoundTripReturnsCopy(t *testing.T) {
	isolateRuntimeSecrets(t)

	token, err := ProtectRuntimeString("credential")
	if err != nil {
		t.Fatalf("ProtectRuntimeString: %v", err)
	}
	defer ForgetRuntimeSecret(token)

	first, err := UnprotectRuntimeBytes(token)
	if err != nil {
		t.Fatalf("UnprotectRuntimeBytes: %v", err)
	}
	first[0] = 'X'

	second, err := UnprotectRuntimeBytes(token)
	if err != nil {
		t.Fatalf("UnprotectRuntimeBytes second read: %v", err)
	}
	if got := string(second); got != "credential" {
		t.Fatalf("stored runtime secret was mutated through returned buffer: %q", got)
	}
	WipeBytes(first)
	WipeBytes(second)
}

func TestRuntimeSecretCapacityFailsClosed(t *testing.T) {
	isolateRuntimeSecrets(t)

	tokens := make([]string, 0, runtimeSecretLimit)
	for i := 0; i < runtimeSecretLimit; i++ {
		token, err := ProtectRuntimeString(fmt.Sprintf("credential-%03d", i))
		if err != nil {
			t.Fatalf("ProtectRuntimeString(%d): %v", i, err)
		}
		tokens = append(tokens, token)
	}

	if token, err := ProtectRuntimeString("overflow"); err == nil || token != "" {
		t.Fatalf("capacity overflow must fail closed, token=%q err=%v", token, err)
	}

	runtimeValues.Lock()
	count := len(runtimeValues.values)
	runtimeValues.Unlock()
	if count != runtimeSecretLimit {
		t.Fatalf("runtime secret store size changed after rejected overflow: got %d want %d", count, runtimeSecretLimit)
	}

	for _, token := range tokens {
		ForgetRuntimeSecret(token)
	}

	token, err := ProtectRuntimeString("after-cleanup")
	if err != nil {
		t.Fatalf("capacity must be reusable after cleanup: %v", err)
	}
	ForgetRuntimeSecret(token)
}
