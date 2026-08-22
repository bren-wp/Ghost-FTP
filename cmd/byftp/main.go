package main

import (
	"brendigo.com/byftp/internal/api"
	"brendigo.com/byftp/internal/brand"
	"brendigo.com/byftp/internal/desktop"
	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/security"
	"brendigo.com/byftp/internal/usererror"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var version = "dev"

func validAskpassInvocation(exePath, askpassExe, require, token string) bool {
	if askpassExe == "" || !strings.EqualFold(strings.TrimSpace(require), "force") || len(token) != 32 {
		return false
	}
	if _, err := hex.DecodeString(token); err != nil {
		return false
	}
	exeAbs, err := filepath.Abs(exePath)
	if err != nil {
		return false
	}
	askAbs, err := filepath.Abs(askpassExe)
	return err == nil && strings.EqualFold(filepath.Clean(exeAbs), filepath.Clean(askAbs))
}

// selectAskpassSecret is intentionally fail-closed. OpenSSH AskPass may also
// be invoked for keyboard-interactive or MFA challenges. ByFTP must never send
// a stored secret to an unknown prompt; only explicit password/passphrase
// prompts are eligible.
func selectAskpassSecret(prompt string, password, passphrase []byte) ([]byte, bool) {
	normalized := strings.ToLower(strings.Join(strings.Fields(prompt), " "))
	for _, blocked := range []string{
		"verification code", "one-time", "one time", "otp", "security key",
		"touch your", "challenge", "response code", "authentication code", "token",
	} {
		if strings.Contains(normalized, blocked) {
			return nil, false
		}
	}
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

func askpassMode() (bool, error) {
	token := os.Getenv("BYFTP_ASKPASS_TOKEN")
	if token == "" {
		return false, nil
	}
	passwordBlob := os.Getenv("BYFTP_PASSWORD_BLOB")
	passphraseBlob := os.Getenv("BYFTP_PASSPHRASE_BLOB")
	askpassExe := os.Getenv("SSH_ASKPASS")
	require := os.Getenv("SSH_ASKPASS_REQUIRE")
	// Remove inherited protected credential material from this helper's own
	// environment immediately. The values remain local to this short-lived
	// process and are never persisted or sent anywhere else.
	for _, key := range []string{"BYFTP_ASKPASS_TOKEN", "BYFTP_PASSWORD_BLOB", "BYFTP_PASSPHRASE_BLOB"} {
		_ = os.Unsetenv(key)
	}
	exe, err := os.Executable()
	if err != nil {
		return true, err
	}
	if !validAskpassInvocation(exe, askpassExe, require, token) || !platform.TrustedAskPassParent() {
		return true, errors.New("invalid sign-in request")
	}
	if passwordBlob == "" && passphraseBlob == "" {
		return true, errors.New("credential is unavailable")
	}
	var password, passphrase []byte
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
	if _, err = os.Stdout.Write(secret); err != nil {
		return true, err
	}
	_, err = os.Stdout.Write([]byte{'\n'})
	return true, err
}

func main() {
	platform.HardenProcessPrivacy()
	if handled, err := askpassMode(); handled {
		if err != nil {
			os.Exit(1)
		}
		return
	}
	defer func() {
		if recover() != nil {
			platform.MessageBox(brand.ProductFull, "ByFTP closed unexpectedly. Restart the application and try again.", 0x10)
		}
	}()
	release, ok := platform.AcquireSingleInstance(brand.Company + "." + brand.ProductName + ".Client")
	if !ok {
		platform.MessageBox(brand.ProductFull, brand.ProductName+" is already running.", 0x40)
		return
	}
	defer release()

	exe, err := os.Executable()
	if err != nil {
		platform.MessageBox("ByFTP", "ByFTP could not start. Restart the computer and try again.", 0x10)
		return
	}
	dataDir, err := api.DataDir()
	if err != nil {
		platform.MessageBox("ByFTP", usererror.Message(err, "ByFTP could not start. Check the user-folder permissions and try again."), 0x10)
		return
	}
	localAppData, err := platform.LocalAppData()
	if err != nil || security.EnsureNoRedirectDirectory(localAppData, dataDir) != nil {
		platform.MessageBox("ByFTP", "The ByFTP data folder is not safe to use. Remove the folder redirection and try again.", 0x10)
		return
	}
	engine, err := api.New(dataDir, exe)
	if err != nil {
		platform.MessageBox("ByFTP", usererror.Message(err, "ByFTP could not start. Try again."), 0x10)
		return
	}
	defer engine.Close()

	if err := desktop.Run(engine, version); err != nil {
		platform.MessageBox("ByFTP", usererror.Message(err, "The ByFTP window could not be opened. Try again."), 0x10)
	}
}
