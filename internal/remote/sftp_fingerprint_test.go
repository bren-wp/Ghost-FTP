package remote

import "testing"

func TestFingerprintScannedKeyMatchesOpenSSHFormat(t *testing.T) {
	line := "example.test ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIAABAgMEBQYHCAkKCwwNDg8QERITFBUWFxgZGhscHR4f"
	got, err := fingerprintScannedKey(line)
	if err != nil {
		t.Fatal(err)
	}
	const want = "SHA256:ZkAslGjFiUHdGf/WUL8rQvkib4PTvQatUV0OUQSncCA"
	if got != want {
		t.Fatalf("fingerprint=%q want %q", got, want)
	}
}

func TestFingerprintScannedKeyRejectsAlgorithmMismatch(t *testing.T) {
	line := "example.test ssh-rsa AAAAC3NzaC1lZDI1NTE5AAAAIAABAgMEBQYHCAkKCwwNDg8QERITFBUWFxgZGhscHR4f"
	if _, err := fingerprintScannedKey(line); err == nil {
		t.Fatal("expected declared algorithm mismatch to fail")
	}
}

func TestFingerprintScannedKeyRejectsMalformedBlob(t *testing.T) {
	if _, err := fingerprintScannedKey("example.test ssh-ed25519 not-base64"); err == nil {
		t.Fatal("expected malformed key blob to fail")
	}
}
