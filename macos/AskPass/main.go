package main

import (
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/platform"
	"github.com/bren-wp/Ghost-FTP/internal/security"
)

const askpassTokenLength = 32

var askpassEnvironmentKeys = [...]string{
	"GhostFTP_ASKPASS_TOKEN",
	"GhostFTP_PASSWORD_BLOB",
	"GhostFTP_PASSPHRASE_BLOB",
	"SSH_ASKPASS",
	"SSH_ASKPASS_REQUIRE",
}

func validToken(token string) bool {
	if len(token) != askpassTokenLength {
		return false
	}
	_, err := hex.DecodeString(token)
	return err == nil
}

func sameExecutable(a, b string) bool {
	a = filepath.Clean(strings.TrimSpace(a))
	b = filepath.Clean(strings.TrimSpace(b))
	if a == "" || b == "" || !filepath.IsAbs(a) || !filepath.IsAbs(b) {
		return false
	}
	ai, err := os.Stat(a)
	if err != nil || !ai.Mode().IsRegular() {
		return false
	}
	bi, err := os.Stat(b)
	return err == nil && bi.Mode().IsRegular() && os.SameFile(ai, bi)
}

func normalizePrompt(prompt string) string {
	return strings.ToLower(strings.Join(strings.Fields(prompt), " "))
}

func selectSecret(prompt string, password, passphrase []byte) ([]byte, bool) {
	prompt = normalizePrompt(prompt)
	if prompt == "" {
		return nil, false
	}
	for _, blocked := range []string{"verification code", "one-time", "one time", "otp", "security key", "touch your", "challenge", "response code", "authentication code", "token"} {
		if strings.Contains(prompt, blocked) {
			return nil, false
		}
	}
	if strings.Contains(prompt, "passphrase") && len(passphrase) > 0 {
		return passphrase, true
	}
	if strings.Contains(prompt, "password") && len(password) > 0 {
		return password, true
	}
	return nil, false
}

func clearEnvironment() {
	for _, key := range askpassEnvironmentKeys {
		_ = os.Unsetenv(key)
	}
}

func writeSecret(secret []byte) error {
	if len(secret) == 0 {
		return errors.New("credential is not available")
	}
	if _, err := os.Stdout.Write(secret); err != nil {
		return err
	}
	_, err := io.WriteString(os.Stdout, "\n")
	return err
}

func run() error {
	platform.HardenProcessPrivacy()
	token := os.Getenv("GhostFTP_ASKPASS_TOKEN")
	passwordBlob := os.Getenv("GhostFTP_PASSWORD_BLOB")
	passphraseBlob := os.Getenv("GhostFTP_PASSPHRASE_BLOB")
	askpassExe := os.Getenv("SSH_ASKPASS")
	require := os.Getenv("SSH_ASKPASS_REQUIRE")
	clearEnvironment()
	if !validToken(token) || !strings.EqualFold(strings.TrimSpace(require), "force") {
		return errors.New("invalid authentication request")
	}
	exe, err := os.Executable()
	if err != nil || !sameExecutable(exe, askpassExe) {
		return errors.New("invalid authentication helper identity")
	}
	if !platform.TrustedAskPassParent() {
		return errors.New("untrusted parent process")
	}
	if passwordBlob == "" && passphraseBlob == "" {
		return errors.New("credential is not available")
	}
	var password, passphrase []byte
	if passwordBlob != "" {
		password, err = security.UnprotectBytes(passwordBlob)
		if err != nil {
			return err
		}
		defer security.WipeBytes(password)
	}
	if passphraseBlob != "" {
		passphrase, err = security.UnprotectBytes(passphraseBlob)
		if err != nil {
			return err
		}
		defer security.WipeBytes(passphrase)
	}
	secret, ok := selectSecret(strings.Join(os.Args[1:], " "), password, passphrase)
	if !ok {
		return errors.New("unknown or unsupported credential request")
	}
	return writeSecret(secret)
}

func main() {
	if run() != nil {
		os.Exit(1)
	}
}
