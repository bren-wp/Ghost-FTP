package main

import (
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bren-wp/by-ftp/internal/api"
	"github.com/bren-wp/by-ftp/internal/brand"
	"github.com/bren-wp/by-ftp/internal/desktop"
	"github.com/bren-wp/by-ftp/internal/platform"
	"github.com/bren-wp/by-ftp/internal/security"
	"github.com/bren-wp/by-ftp/internal/usererror"
)

var version = "dev"

const (
	messageBoxError       = 0x10
	messageBoxInformation = 0x40
	askpassTokenLength    = 32
)

var askpassEnvironmentKeys = [...]string{
	"BYFTP_ASKPASS_TOKEN",
	"BYFTP_PASSWORD_BLOB",
	"BYFTP_PASSPHRASE_BLOB",
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
// challenges. ByFTP must never automatically provide a stored credential
// to an unknown prompt.
//
// Only clearly recognized password and private-key passphrase prompts are
// accepted.
func selectAskpassSecret(
	prompt string,
	password []byte,
	passphrase []byte,
) ([]byte, bool) {
	normalized := normalizeAskpassPrompt(prompt)
	if normalized == "" {
		return nil, false
	}

	if isBlockedAskpassPrompt(normalized) {
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
	return strings.ToLower(
		strings.Join(strings.Fields(prompt), " "),
	)
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
		return errors.New("vjerodajnica nije dostupna")
	}

	if err := writeAll(os.Stdout, secret); err != nil {
		return err
	}

	return writeAll(os.Stdout, []byte{'\n'})
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

	// Remove inherited AskPass and credential material from this helper's
	// environment as early as possible.
	clearAskpassEnvironment()

	exe, err := os.Executable()
	if err != nil {
		return true, err
	}

	if !validAskpassInvocation(exe, askpassExe, require, token) {
		return true, errors.New("neispravan zahtjev za prijavu")
	}

	if !platform.TrustedAskPassParent() {
		return true, errors.New("nepouzdan nadređeni proces")
	}

	if passwordBlob == "" && passphraseBlob == "" {
		return true, errors.New("vjerodajnica nije dostupna")
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

	secret, ok := selectAskpassSecret(
		prompt,
		password,
		passphrase,
	)
	if !ok {
		return true, errors.New(
			"nepoznat ili nepodržan zahtjev za vjerodajnicu",
		)
	}

	if err := writeAskpassSecret(secret); err != nil {
		return true, err
	}

	return true, nil
}

func showError(message string) {
	platform.MessageBox(
		brand.ProductFull,
		message,
		messageBoxError,
	)
}

func runApplication() (exitCode int) {
	defer func() {
		if recover() != nil {
			showError(
				"ByFTP je neočekivano zatvoren. " +
					"Ponovno pokrenite aplikaciju.",
			)

			exitCode = 1
		}
	}()

	release, ok := platform.AcquireSingleInstance(
		brand.Company + "." + brand.ProductName + ".Client",
	)
	if !ok {
		platform.MessageBox(
			brand.ProductFull,
			brand.ProductName+" je već pokrenut.",
			messageBoxInformation,
		)

		return 0
	}
	defer release()

	exe, err := os.Executable()
	if err != nil {
		showError(
			"ByFTP se ne može pokrenuti. " +
				"Ponovno pokrenite računalo i pokušajte ponovno.",
		)

		return 1
	}

	dataDir, err := api.DataDir()
	if err != nil {
		showError(
			usererror.Message(
				err,
				"ByFTP se ne može pokrenuti. "+
					"Provjerite dopuštenja korisničke mape "+
					"i pokušajte ponovno.",
			),
		)

		return 1
	}

	localAppData, err := platform.LocalAppData()
	if err != nil {
		showError(
			"ByFTP ne može pristupiti lokalnoj podatkovnoj mapi.",
		)

		return 1
	}

	if err := security.EnsureNoRedirectDirectory(
		localAppData,
		dataDir,
	); err != nil {
		showError(
			"ByFTP podatkovna mapa nije sigurna. " +
				"Uklonite preusmjeravanje te mape " +
				"i pokušajte ponovno.",
		)

		return 1
	}

	engine, err := api.New(dataDir, exe)
	if err != nil {
		showError(
			usererror.Message(
				err,
				"ByFTP se ne može pokrenuti. "+
					"Pokušajte ponovno.",
			),
		)

		return 1
	}
	defer engine.Close()

	if err := desktop.Run(engine, version); err != nil {
		showError(
			usererror.Message(
				err,
				"ByFTP prozor nije moguće otvoriti. "+
					"Pokušajte ponovno.",
			),
		)

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
