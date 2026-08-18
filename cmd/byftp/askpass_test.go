package main

import (
	"bytes"
	"testing"
)

func TestValidAskpassInvocation(t *testing.T) {
	exe := `C:\\Program Files\\ByFTP\\ByFTP.exe`
	token := "0123456789abcdef0123456789abcdef"
	if !validAskpassInvocation(exe, exe, "force", token) {
		t.Fatal("expected controlled AskPass invocation to validate")
	}
	for _, tc := range []struct {
		name, askExe, require, token string
	}{
		{"wrong exe", `C:\\Other\\ByFTP.exe`, "force", token},
		{"wrong require", exe, "prefer", token},
		{"short token", exe, "force", "abc"},
		{"nonhex token", exe, "force", "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"},
	} {
		if validAskpassInvocation(exe, tc.askExe, tc.require, tc.token) {
			t.Fatalf("%s unexpectedly validated", tc.name)
		}
	}
}

func TestSelectAskpassSecretOnlyUsesRecognizedPrompts(t *testing.T) {
	password := []byte("server-password")
	passphrase := []byte("key-passphrase")

	secret, ok := selectAskpassSecret("user@example.test's password:", password, passphrase)
	if !ok || !bytes.Equal(secret, password) {
		t.Fatalf("password prompt did not select password: ok=%v secret=%q", ok, secret)
	}
	secret, ok = selectAskpassSecret("Enter passphrase for key 'id_ed25519':", password, passphrase)
	if !ok || !bytes.Equal(secret, passphrase) {
		t.Fatalf("passphrase prompt did not select passphrase: ok=%v secret=%q", ok, secret)
	}
	for _, prompt := range []string{
		"Verification code:",
		"One-time password token:",
		"Touch your security key",
		"",
	} {
		if secret, ok := selectAskpassSecret(prompt, password, passphrase); ok || secret != nil {
			t.Fatalf("unknown prompt %q unexpectedly received a secret", prompt)
		}
	}
}

func TestSelectAskpassSecretRequiresMatchingCredential(t *testing.T) {
	if _, ok := selectAskpassSecret("Password:", nil, []byte("passphrase")); ok {
		t.Fatal("password prompt must not fall back to passphrase")
	}
	if _, ok := selectAskpassSecret("Passphrase:", []byte("password"), nil); ok {
		t.Fatal("passphrase prompt must not fall back to password")
	}
}
