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

var version = "2.12.0"

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
	prompt := strings.ToLower(strings.Join(os.Args[1:], " "))
	secret := password
	if strings.Contains(prompt, "passphrase") && len(passphrase) != 0 {
		secret = passphrase
	}
	if len(secret) == 0 {
		return true, errors.New("vjerodajnica nije dostupna")
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
			platform.MessageBox(brand.ProductFull, "ByFTP je neočekivano zatvoren. Ponovno pokrenite aplikaciju.", 0x10)
		}
	}()
	release, ok := platform.AcquireSingleInstance(brand.Company + "." + brand.ProductName + ".Client")
	if !ok {
		platform.MessageBox(brand.ProductFull, brand.ProductName+" je već pokrenut.", 0x40)
		return
	}
	defer release()

	exe, err := os.Executable()
	if err != nil {
		platform.MessageBox("ByFTP", "ByFTP se ne može pokrenuti. Ponovno pokrenite računalo i pokušajte ponovno.", 0x10)
		return
	}
	dataDir, err := api.DataDir()
	if err != nil {
		platform.MessageBox("ByFTP", usererror.Message(err, "ByFTP se ne može pokrenuti. Provjerite dopuštenja korisničke mape i pokušajte ponovno."), 0x10)
		return
	}
	localAppData, err := platform.LocalAppData()
	if err != nil || security.EnsureNoRedirectDirectory(localAppData, dataDir) != nil {
		platform.MessageBox("ByFTP", "ByFTP podatkovna mapa nije sigurna. Uklonite preusmjeravanje te mape i pokušajte ponovno.", 0x10)
		return
	}
	engine, err := api.New(dataDir, exe)
	if err != nil {
		platform.MessageBox("ByFTP", usererror.Message(err, "ByFTP se ne može pokrenuti. Pokušajte ponovno."), 0x10)
		return
	}
	defer engine.Close()

	if err := desktop.Run(engine, version); err != nil {
		platform.MessageBox("ByFTP", usererror.Message(err, "ByFTP prozor nije moguće otvoriti. Pokušajte ponovno."), 0x10)
	}
}
