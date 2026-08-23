package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/bren-wp/by-ftp/internal/brand"
	"github.com/bren-wp/by-ftp/internal/platform"
	"github.com/bren-wp/by-ftp/internal/security"
)

//go:embed all:payload
var payload embed.FS

const (
	maxPayloadFileSize     = 128 << 20
	maxPayloadManifestSize = 64 << 10

	uninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\ByFTP`
	appPathsKey  = `Software\Microsoft\Windows\CurrentVersion\App Paths\ByFTP.exe`
)

var version = "dev"

type payloadManifestFile struct {
	Name   string `json:"name"`
	Size   int    `json:"size"`
	SHA256 string `json:"sha256"`
}

type payloadManifest struct {
	Schema int                   `json:"schema"`
	Files  []payloadManifestFile `json:"files"`
}

func readPayloadFiles() ([]byte, []byte, error) {
	data, err := payload.ReadFile("payload/payload.zip")
	if err != nil {
		return nil, nil, errors.New("komprimirani instalacijski payload nije dostupan")
	}
	return parsePayload(data)
}

func parsePayload(data []byte) ([]byte, []byte, error) {
	if len(data) == 0 {
		return nil, nil, errors.New("instalacijski payload je prazan")
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, nil, errors.New("instalacijski payload nije ispravan ZIP")
	}

	files := make(map[string][]byte, 2)
	var manifestData []byte

	for _, f := range zr.File {
		switch f.Name {
		case "ByFTP.exe", "Uninstall.exe":
			if _, exists := files[f.Name]; exists {
				return nil, nil, errors.New("instalacijski payload sadrži dupliciranu datoteku")
			}

			b, err := readZipEntry(f, maxPayloadFileSize)
			if err != nil {
				return nil, nil, err
			}
			files[f.Name] = b

		case "manifest.json":
			if manifestData != nil {
				return nil, nil, errors.New("instalacijski payload sadrži duplicirani manifest")
			}

			manifestData, err = readZipEntry(f, maxPayloadManifestSize)
			if err != nil {
				return nil, nil, err
			}

		default:
			return nil, nil, errors.New("instalacijski payload sadrži neočekivanu datoteku")
		}
	}

	app, appOK := files["ByFTP.exe"]
	uninstaller, uninstallOK := files["Uninstall.exe"]
	if !appOK || !uninstallOK || manifestData == nil {
		return nil, nil, errors.New("instalacijski payload ne sadrži sve obavezne datoteke")
	}

	if err := validatePayloadManifest(manifestData, files); err != nil {
		return nil, nil, err
	}

	return app, uninstaller, nil
}

// readZipEntry always reads through EOF. Besides enforcing the size limit,
// this is important because archive/zip verifies the entry checksum while
// reading and may report checksum corruption only at the end of the stream.
func readZipEntry(f *zip.File, maxSize uint64) ([]byte, error) {
	if f == nil || f.UncompressedSize64 == 0 || f.UncompressedSize64 > maxSize {
		return nil, errors.New("instalacijski payload ima neispravnu veličinu")
	}

	r, err := f.Open()
	if err != nil {
		return nil, errors.New("instalacijsku datoteku nije moguće otvoriti")
	}

	data, readErr := io.ReadAll(io.LimitReader(r, int64(maxSize)+1))
	closeErr := r.Close()

	if readErr != nil {
		return nil, errors.New("instalacijski payload je oštećen ili nepotpun")
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if uint64(len(data)) != f.UncompressedSize64 || uint64(len(data)) > maxSize {
		return nil, errors.New("instalacijski payload ima neispravnu veličinu")
	}

	return data, nil
}

func validatePayloadManifest(data []byte, files map[string][]byte) error {
	var manifest payloadManifest

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	if err := dec.Decode(&manifest); err != nil {
		return errors.New("instalacijski manifest nije ispravan")
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("instalacijski manifest sadrži višak podataka")
	}
	if manifest.Schema != 1 || len(manifest.Files) != 2 {
		return errors.New("instalacijski manifest nije podržan")
	}

	seen := make(map[string]bool, 2)
	for _, item := range manifest.Files {
		content, ok := files[item.Name]
		if !ok || seen[item.Name] || (item.Name != "ByFTP.exe" && item.Name != "Uninstall.exe") {
			return errors.New("instalacijski manifest ne odgovara paketu")
		}
		seen[item.Name] = true

		expectedDigest, err := hex.DecodeString(item.SHA256)
		if err != nil || len(expectedDigest) != sha256.Size {
			return errors.New("instalacijski manifest sadrži neispravan SHA-256")
		}

		actualDigest := sha256.Sum256(content)
		if item.Size != len(content) || !bytes.Equal(expectedDigest, actualDigest[:]) {
			return errors.New("provjera integriteta instalacijskog paketa nije uspjela")
		}
	}

	if !seen["ByFTP.exe"] || !seen["Uninstall.exe"] {
		return errors.New("instalacijski manifest nije potpun")
	}

	return nil
}

func stageVerified(path string, data []byte) (string, error) {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".byftp-install-*.tmp")
	if err != nil {
		return "", err
	}

	tmp := f.Name()
	cleanup := func() {
		_ = f.Close()
		_ = os.Remove(tmp)
	}

	// Restrict the temporary file to the current user where the platform
	// honors POSIX-style mode bits. On Windows the effective protection is
	// provided by the directory ACL and the platform security checks.
	if err := f.Chmod(0700); err != nil {
		cleanup()
		return "", err
	}

	n, err := io.Copy(f, bytes.NewReader(data))
	if err != nil || n != int64(len(data)) {
		cleanup()
		if err != nil {
			return "", err
		}
		return "", io.ErrShortWrite
	}

	if err := f.Sync(); err != nil {
		cleanup()
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}

	expectedDigest := sha256.Sum256(data)
	if err := verifyStagedFile(tmp, int64(len(data)), expectedDigest); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}

	return tmp, nil
}

func verifyStagedFile(path string, expectedSize int64, expectedDigest [sha256.Size]byte) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	info, statErr := f.Stat()
	if statErr != nil {
		_ = f.Close()
		return statErr
	}
	if info.Size() != expectedSize {
		_ = f.Close()
		return errors.New("provjera veličine instalacijske datoteke nije uspjela")
	}

	h := sha256.New()
	n, copyErr := io.Copy(h, f)
	closeErr := f.Close()

	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if n != expectedSize {
		return errors.New("provjera veličine instalacijske datoteke nije uspjela")
	}
	if !bytes.Equal(h.Sum(nil), expectedDigest[:]) {
		return errors.New("provjera integriteta instalacijske datoteke nije uspjela")
	}

	return nil
}

func installFile(path string, data []byte, backup *fileBackup) error {
	if backup == nil || backup.target != path {
		return errors.New("instalacijska transakcija ne odgovara ciljnoj datoteci")
	}

	tmp, err := stageVerified(path, data)
	if err != nil {
		return err
	}

	if err := backup.verifyBeforeInstall(); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	// A fresh install must never overwrite a file that appeared after the
	// transaction snapshot. Upgrades use ReplaceFile only after the previous
	// object was backed up and revalidated immediately above.
	if backup.existed() {
		err = platform.ReplaceFile(tmp, path)
	} else {
		err = platform.RenameNoReplace(tmp, path)
	}
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}

	return backup.recordActivated(sha256.Sum256(data))
}

func register(appPath, uninstallPath, dir string) error {
	entries := [...]struct {
		key   string
		name  string
		value string
	}{
		{uninstallKey, "DisplayName", brand.ProductFull},
		{uninstallKey, "DisplayVersion", version},
		{uninstallKey, "Publisher", brand.Company},
		{uninstallKey, "InstallLocation", dir},
		{uninstallKey, "DisplayIcon", appPath},
		{uninstallKey, "UninstallString", `"` + uninstallPath + `"`},
		{appPathsKey, "", appPath},
	}

	for _, entry := range entries {
		if err := platform.SetRegistryString(entry.key, entry.name, entry.value); err != nil {
			return fmt.Errorf("Windows registracija nije uspjela (%s): %w", entry.name, err)
		}
	}

	if err := platform.SetRegistryString(uninstallKey, "InstallDate", time.Now().Format("20060102")); err != nil {
		return fmt.Errorf("Windows registracija nije uspjela (InstallDate): %w", err)
	}
	if err := platform.SetRegistryDWORD(uninstallKey, "NoModify", 1); err != nil {
		return fmt.Errorf("Windows registracija nije uspjela (NoModify): %w", err)
	}
	if err := platform.SetRegistryDWORD(uninstallKey, "NoRepair", 1); err != nil {
		return fmt.Errorf("Windows registracija nije uspjela (NoRepair): %w", err)
	}

	return nil
}

func installerTitle() string {
	return brand.ProductName + " — Instalacija"
}

func showInstallError(title, message string) {
	platform.ErrorDialog(installerTitle(), title, message)
}

func runInstaller() (exitCode int) {
	exitCode = 1

	defer func() {
		if recover() != nil {
			platform.ErrorDialog(
				brand.ProductFull+" — "+brand.Company,
				"Radnja nije dovršena",
				"Instalacija nije dovršena. Ponovno pokrenite računalo i pokušajte ponovno.",
			)
			exitCode = 1
		}
	}()

	content := brand.Company + "\n" + brand.Website + "\n" + brand.Support + "\n\n" +
		"ByFTP će se instalirati za vaš Windows korisnički račun i bit će dostupan iz izbornika Start."

	if !platform.ConfirmDialog(
		installerTitle(),
		"Instalirati "+brand.ProductFull+" "+version+"?",
		content,
	) {
		return 0
	}

	// Validate the embedded package before changing the filesystem or registry.
	app, uninstaller, err := readPayloadFiles()
	if err != nil {
		showInstallError(
			"Instalacijski paket nije ispravan",
			"Preuzmite novu kopiju instalacijskog paketa i pokušajte ponovno.",
		)
		return 1
	}

	dir, err := platform.InstallDir()
	if err != nil {
		showInstallError(
			"Instalacija nije moguća",
			"Korisnička mapa sustava Windows nije dostupna.",
		)
		return 1
	}

	localAppData, err := platform.LocalAppData()
	if err != nil {
		showInstallError(
			"Instalacija nije moguća",
			"Lokalna korisnička mapa sustava Windows nije dostupna.",
		)
		return 1
	}
	if err := security.EnsureNoRedirectDirectory(localAppData, dir); err != nil {
		showInstallError(
			"Instalacijska mapa nije sigurna",
			"Uklonite preusmjeravanje ByFTP instalacijske mape i pokušajte ponovno.",
		)
		return 1
	}

	if err := ensureInstallDir(dir); err != nil {
		showInstallError(
			"Instalacija nije pokrenuta",
			"Instalacijsku mapu nije moguće pripremiti. Provjerite slobodan prostor i dopuštenja pa pokušajte ponovno.",
		)
		return 1
	}

	appPath := filepath.Join(dir, "ByFTP.exe")
	uninstallPath := filepath.Join(dir, "Uninstall.exe")

	appBackup, err := backupExisting(appPath)
	if err != nil {
		showInstallError(
			"Nadogradnja nije pokrenuta",
			"Postojeću instalaciju nije moguće pripremiti za nadogradnju. Zatvorite ByFTP i pokušajte ponovno.",
		)
		return 1
	}

	uninstallBackup, err := backupExisting(uninstallPath)
	if err != nil {
		appBackup.cleanup()
		showInstallError(
			"Nadogradnja nije pokrenuta",
			"Postojeću instalaciju nije moguće pripremiti za nadogradnju. Zatvorite ByFTP i pokušajte ponovno.",
		)
		return 1
	}

	registryBackup, err := captureRegistrySnapshot()
	if err != nil {
		appBackup.cleanup()
		uninstallBackup.cleanup()
		showInstallError(
			"Nadogradnja nije pokrenuta",
			"Postojeće postavke instalacije nije moguće sigurno pripremiti. Pokušajte ponovno.",
		)
		return 1
	}

	freshInstall := !appBackup.existed() && !uninstallBackup.existed()
	transactionCommitted := false
	rolledBack := false

	rollback := func() error {
		if rolledBack {
			return nil
		}
		rolledBack = true

		var errs []error
		if err := uninstallBackup.rollback(); err != nil {
			errs = append(errs, err)
		}
		if err := appBackup.rollback(); err != nil {
			errs = append(errs, err)
		}
		if err := registryBackup.restore(); err != nil {
			errs = append(errs, err)
		}
		if freshInstall {
			if err := platform.RemoveShortcuts(); err != nil {
				errs = append(errs, err)
			}
			if err := platform.DeleteRegistryKey(uninstallKey); err != nil {
				errs = append(errs, err)
			}
			if err := platform.DeleteRegistryKey(appPathsKey); err != nil {
				errs = append(errs, err)
			}
		}

		return errors.Join(errs...)
	}

	// From this point forward, an unexpected panic must also trigger rollback.
	defer func() {
		if !transactionCommitted && !rolledBack {
			_ = rollback()
		}
		appBackup.cleanup()
		uninstallBackup.cleanup()
	}()

	rollbackMessage := func() string {
		if err := rollback(); err != nil {
			return " Prethodnu instalaciju nije bilo moguće potpuno vratiti; ponovno pokrenite Windows prije novog pokušaja."
		}
		return ""
	}

	if err := installFile(appPath, app, &appBackup); err != nil {
		extra := rollbackMessage()
		showInstallError(
			"ByFTP nije moguće instalirati",
			"Zatvorite pokrenuti ByFTP i pokušajte ponovno."+extra,
		)
		return 1
	}

	if err := installFile(uninstallPath, uninstaller, &uninstallBackup); err != nil {
		extra := rollbackMessage()
		showInstallError(
			"Instalacija nije dovršena",
			"Potrebne datoteke nije moguće spremiti. Pokušajte ponovno."+extra,
		)
		return 1
	}

	if err := register(appPath, uninstallPath, dir); err != nil {
		extra := rollbackMessage()
		showInstallError(
			"Instalacija nije dovršena",
			"Windows nije uspio dovršiti postavljanje aplikacije. Pokušajte ponovno."+extra,
		)
		return 1
	}

	transactionCommitted = true

	shortcutWarning := ""
	if err := platform.CreateShortcuts(appPath); err != nil {
		shortcutWarning = "\n\nPrečac nije moguće izraditi. ByFTP možete pokrenuti iz instalacijske mape."
	}

	if platform.ConfirmDialog(
		installerTitle(),
		"Instalacija je uspješno dovršena",
		"ByFTP je spreman za korištenje."+shortcutWarning+"\n\nPokrenuti ByFTP sada?",
	) {
		cmd := exec.Command(appPath)
		if err := cmd.Start(); err != nil {
			platform.ErrorDialog(
				installerTitle(),
				"ByFTP je instaliran",
				"Aplikaciju nije moguće automatski pokrenuti. Pokrenite ByFTP iz izbornika Start.",
			)
		}
	}

	return 0
}

func main() {
	platform.HardenProcessPrivacy()
	os.Exit(runInstaller())
}
