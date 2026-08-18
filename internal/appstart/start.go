package appstart

import (
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"brendigo.com/byftp/internal/api"
	"brendigo.com/byftp/internal/clientmode"
	"brendigo.com/byftp/internal/desktop"
	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/security"
	"brendigo.com/byftp/internal/usererror"
)

func ValidAskpassInvocation(exePath, askpassExe, require, token string) bool {
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

// SelectAskpassSecret je fail-closed. OpenSSH AskPass može se pozvati i za
// keyboard-interactive/MFA upite. ByFTP tajnu daje samo jasno prepoznatom
// password/passphrase promptu.
func SelectAskpassSecret(prompt string, password, passphrase []byte) ([]byte, bool) {
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
	for _, key := range []string{"BYFTP_ASKPASS_TOKEN", "BYFTP_PASSWORD_BLOB", "BYFTP_PASSPHRASE_BLOB"} {
		_ = os.Unsetenv(key)
	}
	exe, err := os.Executable()
	if err != nil {
		return true, err
	}
	if !ValidAskpassInvocation(exe, askpassExe, require, token) || !platform.TrustedAskPassParent() {
		return true, errors.New("neispravan zahtjev za prijavu")
	}
	if passwordBlob == "" && passphraseBlob == "" {
		return true, errors.New("vjerodajnica nije dostupna")
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
	secret, ok := SelectAskpassSecret(prompt, password, passphrase)
	if !ok {
		return true, errors.New("nepoznat ili nepodržan zahtjev za vjerodajnicu")
	}
	if _, err = os.Stdout.Write(secret); err != nil {
		return true, err
	}
	_, err = os.Stdout.Write([]byte{'\n'})
	return true, err
}

func modeDataDir(mode clientmode.Mode) (string, error) {
	base, err := api.DataDir()
	if err != nil {
		return "", err
	}
	if mode == clientmode.Suite {
		return base, nil
	}
	return filepath.Join(base, "clients", mode.Slug()), nil
}

func RunFileClient(mode clientmode.Mode, version string) {
	platform.HardenProcessPrivacy()
	if handled, err := askpassMode(); handled {
		if err != nil {
			os.Exit(1)
		}
		return
	}
	product := mode.ProductName()
	defer func() {
		if recover() != nil {
			platform.MessageBox(product, product+" je neočekivano zatvoren. Ponovno pokrenite aplikaciju.", 0x10)
		}
	}()
	release, ok := platform.AcquireSingleInstance(mode.InstanceKey())
	if !ok {
		platform.MessageBox(product, product+" je već pokrenut.", 0x40)
		return
	}
	defer release()

	exe, err := os.Executable()
	if err != nil {
		platform.MessageBox(product, product+" se ne može pokrenuti. Ponovno pokrenite računalo i pokušajte ponovno.", 0x10)
		return
	}
	dataDir, err := modeDataDir(mode)
	if err != nil {
		platform.MessageBox(product, usererror.Message(err, product+" se ne može pokrenuti. Provjerite dopuštenja korisničke mape."), 0x10)
		return
	}
	localAppData, err := platform.LocalAppData()
	if err != nil || security.EnsureNoRedirectDirectory(localAppData, dataDir) != nil {
		platform.MessageBox(product, product+" podatkovna mapa nije sigurna. Uklonite preusmjeravanje te mape i pokušajte ponovno.", 0x10)
		return
	}
	engine, err := api.New(dataDir, exe)
	if err != nil {
		platform.MessageBox(product, usererror.Message(err, product+" se ne može pokrenuti. Pokušajte ponovno."), 0x10)
		return
	}
	defer engine.Close()

	desktop.SetClientMode(mode)
	if err := desktop.Run(engine, version); err != nil {
		platform.MessageBox(product, usererror.Message(err, product+" prozor nije moguće otvoriti. Pokušajte ponovno."), 0x10)
	}
}
