package main

import (
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/api"
	"github.com/bren-wp/Ghost-FTP/internal/brand"
	"github.com/bren-wp/Ghost-FTP/internal/desktop"
	"github.com/bren-wp/Ghost-FTP/internal/platform"
	"github.com/bren-wp/Ghost-FTP/internal/security"
	"github.com/bren-wp/Ghost-FTP/internal/usererror"
)

var version = "dev"

const (
	messageBoxError       = 0x10
	messageBoxInformation = 0x40
	askpassTokenLength    = 32
)

var askpassEnvironmentKeys = [...]string{
	"GhostFTP_ASKPASS_TOKEN",
	"GhostFTP_PASSWORD_BLOB",
	"GhostFTP_PASSPHRASE_BLOB",
	"SSH_ASKPASS",
	"SSH_ASKPASS_REQUIRE",
}

func validAskpassInvocation(exePath, askpassExe, require, token string) bool {
	if exePath == "" || askpassExe == "" {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(require), "force") {
		return false
	}
	if !validAskpassToken(token) {
		return false
	}

	exeAbs, err := filepath.Abs(exePath)
	if err != nil {
		return false
	}
	askpassAbs, err := filepath.Abs(askpassExe)
	if err != nil {
		return false
	}

	exeAbs = filepath.Clean(exeAbs)
	askpassAbs = filepath.Clean(askpassAbs)
	return strings.EqualFold(exeAbs, askpassAbs)
}

func validAskpassToken(token string) bool {
	if len(token) != askpassTokenLength {
		return false
	}
	_, err := hex.DecodeString(token)
	return err == nil
}

// selectAskpassSecret is intentionally fail-closed.
//
// OpenSSH AskPass can also be invoked for keyboard-interactive and MFA
// challenges. Ghost FTP must never automatically provide a stored credential
// to an unknown prompt.
//
// Only clearly recognized password and private-key passphrase prompts are
// accepted.
func selectAskpassSecret(prompt string, password, passphrase []byte) ([]byte, bool) {
	normalized := normalizeAskpassPrompt(prompt)
	if normalized == "" || isBlockedAskpassPrompt(normalized) {
		return nil, false
	}

	// Check passphrase first so that private-key prompts can never
	// accidentally fall through to password handling.
	if strings.Contains(normalized, "passphrase") {
		if len(passphrase) == 0 {
			return nil, false
		}
		return passphrase, true
	}
	if strings.Contains(normalized, "password") {
		if len(password) == 0 {
			return nil, false
		}
		return password, true
	}
	return nil, false
}

func normalizeAskpassPrompt(prompt string) string {
	return strings.ToLower(strings.Join(strings.Fields(prompt), " "))
}

func isBlockedAskpassPrompt(prompt string) bool {
	blockedPrompts := [...]string{
		"verification code",
		"one-time",
		"one time",
		"otp",
		"security key",
		"touch your",
		"challenge",
		"response code",
		"authentication code",
		"token",
	}
	for _, blocked := range blockedPrompts {
		if strings.Contains(prompt, blocked) {
			return true
		}
	}
	return false
}

func clearAskpassEnvironment() {
	for _, key := range askpassEnvironmentKeys {
		_ = os.Unsetenv(key)
	}
}

func writeAll(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if err != nil {
			return err
		}
		if n <= 0 || n > len(data) {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}

func writeAskpassSecret(secret []byte) error {
	if len(secret) == 0 {
		return errors.New("credential is not available")
	}
	if err := writeAll(os.Stdout, secret); err != nil {
		return err
	}
	return writeAll(os.Stdout, []byte{'\n'})
}

func askpassMode() (bool, error) {
	token := os.Getenv("GhostFTP_ASKPASS_TOKEN")
	if token == "" {
		return false, nil
	}

	passwordBlob := os.Getenv("GhostFTP_PASSWORD_BLOB")
	passphraseBlob := os.Getenv("GhostFTP_PASSPHRASE_BLOB")
	askpassExe := os.Getenv("SSH_ASKPASS")
	require := os.Getenv("SSH_ASKPASS_REQUIRE")

	// Remove inherited AskPass and credential material from this helper's
	// environment as early as possible.
	clearAskpassEnvironment()

	exe, err := os.Executable()
	if err != nil {
		return true, err
	}
	if !validAskpassInvocation(exe, askpassExe, require, token) {
		return true, errors.New("invalid authentication request")
	}
	if !platform.TrustedAskPassParent() {
		return true, errors.New("untrusted parent process")
	}
	if passwordBlob == "" && passphraseBlob == "" {
		return true, errors.New("credential is not available")
	}

	var (
		password   []byte
		passphrase []byte
	)
	if passwordBlob != "" {
		password, err = security.UnprotectBytes(passwordBlob)
		if err != nil {
			return true, err
		}
		defer security.WipeBytes(password)
	}
	if passphraseBlob != "" {
		passphrase, err = security.UnprotectBytes(passphraseBlob)
		if err != nil {
			return true, err
		}
		defer security.WipeBytes(passphrase)
	}

	prompt := strings.Join(os.Args[1:], " ")
	secret, ok := selectAskpassSecret(prompt, password, passphrase)
	if !ok {
		return true, errors.New("unknown or unsupported credential request")
	}
	if err := writeAskpassSecret(secret); err != nil {
		return true, err
	}
	return true, nil
}

func showError(message string) {
	platform.MessageBox(brand.ProductFull, message, messageBoxError)
}

func runApplication() (exitCode int) {
	defer func() {
		if recover() != nil {
			showError("Ghost FTP closed unexpectedly. Restart the application and try again.")
			exitCode = 1
		}
	}()

	release, ok := platform.AcquireSingleInstance(brand.Company + "." + brand.ProductName + ".Client")
	if !ok {
		platform.MessageBox(brand.ProductFull, brand.ProductName+" is already running.", messageBoxInformation)
		return 0
	}
	defer release()

	exe, err := os.Executable()
	if err != nil {
		showError("Ghost FTP could not start. Restart the computer and try again.")
		return 1
	}

	dataDir, err := api.DataDir()
	if err != nil {
		showError(usererror.Message(err, "Ghost FTP could not start. Check permissions for the user data folder and try again."))
		return 1
	}
	localAppData, err := platform.LocalAppData()
	if err != nil {
		showError("Ghost FTP could not access the local application-data folder.")
		return 1
	}
	if err := security.EnsureNoRedirectDirectory(localAppData, dataDir); err != nil {
		showError("The Ghost FTP data folder is not safe to use. Remove filesystem redirection for that folder and try again.")
		return 1
	}

	engine, err := api.New(dataDir, exe)
	if err != nil {
		showError(usererror.Message(err, "Ghost FTP could not start. Please try again."))
		return 1
	}
	defer engine.Close()

	if err := desktop.Run(engine, version); err != nil {
		showError(usererror.Message(err, "The Ghost FTP window could not be opened. Please try again."))
		return 1
	}
	return 0
}

func main() {
	platform.HardenProcessPrivacy()

	handled, err := askpassMode()
	if handled {
		if err != nil {
			os.Exit(1)
		}
		return
	}

	exitCode := runApplication()
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
