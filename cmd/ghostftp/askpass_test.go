package main

import (
	"bytes"
	"testing"
)

const (
	testExecutable = `C:\Program Files\GhostFTP\GhostFTP.exe`
	validToken     = "0123456789abcdef0123456789abcdef"
)

func TestValidAskpassInvocation(t *testing.T) {
	t.Run("accepts controlled invocation", func(t *testing.T) {
		if !validAskpassInvocation(
			testExecutable,
			testExecutable,
			"force",
			validToken,
		) {
			t.Fatal("expected controlled AskPass invocation to be valid")
		}
	})

	tests := []struct {
		name       string
		askpassExe string
		require    string
		token      string
	}{
		{
			name:       "rejects wrong executable",
			askpassExe: `C:\Other\GhostFTP.exe`,
			require:    "force",
			token:      validToken,
		},
		{
			name:       "rejects wrong require mode",
			askpassExe: testExecutable,
			require:    "prefer",
			token:      validToken,
		},
		{
			name:       "rejects short token",
			askpassExe: testExecutable,
			require:    "force",
			token:      "abc",
		},
		{
			name:       "rejects non-hex token",
			askpassExe: testExecutable,
			require:    "force",
			token:      "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
		},
		{
			name:       "rejects empty executable",
			askpassExe: "",
			require:    "force",
			token:      validToken,
		},
		{
			name:       "rejects empty require mode",
			askpassExe: testExecutable,
			require:    "",
			token:      validToken,
		},
		{
			name:       "rejects empty token",
			askpassExe: testExecutable,
			require:    "force",
			token:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if validAskpassInvocation(
				testExecutable,
				tt.askpassExe,
				tt.require,
				tt.token,
			) {
				t.Fatal("invalid AskPass invocation unexpectedly validated")
			}
		})
	}
}

func TestSelectAskpassSecret(t *testing.T) {
	password := []byte("server-password")
	passphrase := []byte("key-passphrase")

	tests := []struct {
		name       string
		prompt     string
		password   []byte
		passphrase []byte
		expected   []byte
		wantOK     bool
	}{
		{
			name:       "selects password for password prompt",
			prompt:     "user@example.test's password:",
			password:   password,
			passphrase: passphrase,
			expected:   password,
			wantOK:     true,
		},
		{
			name:       "selects passphrase for key prompt",
			prompt:     "Enter passphrase for key 'id_ed25519':",
			password:   password,
			passphrase: passphrase,
			expected:   passphrase,
			wantOK:     true,
		},
		{
			name:       "rejects verification code prompt",
			prompt:     "Verification code:",
			password:   password,
			passphrase: passphrase,
		},
		{
			name:       "rejects one-time password prompt",
			prompt:     "One-time password token:",
			password:   password,
			passphrase: passphrase,
		},
		{
			name:       "rejects security key prompt",
			prompt:     "Touch your security key",
			password:   password,
			passphrase: passphrase,
		},
		{
			name:       "rejects empty prompt",
			prompt:     "",
			password:   password,
			passphrase: passphrase,
		},
		{
			name:       "password does not fall back to passphrase",
			prompt:     "Password:",
			password:   nil,
			passphrase: passphrase,
		},
		{
			name:       "passphrase does not fall back to password",
			prompt:     "Passphrase:",
			password:   password,
			passphrase: nil,
		},
		{
			name:       "password prompt rejects empty password",
			prompt:     "Password:",
			password:   []byte{},
			passphrase: passphrase,
		},
		{
			name:       "passphrase prompt rejects empty passphrase",
			prompt:     "Passphrase:",
			password:   password,
			passphrase: []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret, ok := selectAskpassSecret(
				tt.prompt,
				tt.password,
				tt.passphrase,
			)

			if ok != tt.wantOK {
				t.Fatalf(
					"selectAskpassSecret() ok = %v, want %v",
					ok,
					tt.wantOK,
				)
			}

			if !tt.wantOK {
				if secret != nil {
					t.Fatal("expected nil secret for rejected prompt")
				}
				return
			}

			if !bytes.Equal(secret, tt.expected) {
				t.Fatal("selectAskpassSecret() returned unexpected secret")
			}
		})
	}
}
