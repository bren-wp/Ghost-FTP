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
		return nil, nil, errors.New("compressed installer payload is unavailable")
	}
	return parsePayload(data)
}

func parsePayload(data []byte) ([]byte, []byte, error) {
	if len(data) == 0 {
		return nil, nil, errors.New("installer payload is empty")
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, nil, errors.New("installer payload is not a valid ZIP archive")
	}

	files := make(map[string][]byte, 2)
	var manifestData []byte

	for _, f := range zr.File {
		switch f.Name {
		case "ByFTP.exe", "Uninstall.exe":
			if _, exists := files[f.Name]; exists {
				return nil, nil, errors.New("installer payload contains a duplicate file")
			}

			b, err := readZipEntry(f, maxPayloadFileSize)
			if err != nil {
				return nil, nil, err
			}
			files[f.Name] = b

		case "manifest.json":
			if manifestData != nil {
				return nil, nil, errors.New("installer payload contains a duplicate manifest")
			}

			manifestData, err = readZipEntry(f, maxPayloadManifestSize)
			if err != nil {
				return nil, nil, err
			}

		default:
			return nil, nil, errors.New("installer payload contains an unexpected file")
		}
	}

	app, appOK := files["ByFTP.exe"]
	uninstaller, uninstallOK := files["Uninstall.exe"]
	if !appOK || !uninstallOK || manifestData == nil {
		return nil, nil, errors.New("installer payload is missing required files")
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
		return nil, errors.New("installer payload has an invalid size")
	}

	r, err := f.Open()
	if err != nil {
		return nil, errors.New("installer file could not be opened")
	}

	data, readErr := io.ReadAll(io.LimitReader(r, int64(maxSize)+1))
	closeErr := r.Close()

	if readErr != nil {
		return nil, errors.New("installer payload is damaged or incomplete")
	}
	if closeErr != nil {
		return nil, closeErr
	}
	if uint64(len(data)) != f.UncompressedSize64 || uint64(len(data)) > maxSize {
		return nil, errors.New("installer payload has an invalid size")
	}

	return data, nil
}

func validatePayloadManifest(data []byte, files map[string][]byte) error {
	var manifest payloadManifest

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	if err := dec.Decode(&manifest); err != nil {
		return errors.New("installer manifest is invalid")
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("installer manifest contains trailing data")
	}
	if manifest.Schema != 1 || len(manifest.Files) != 2 {
		return errors.New("installer manifest is unsupported")
	}

	seen := make(map[string]bool, 2)
	for _, item := range manifest.Files {
		content, ok := files[item.Name]
		if !ok || seen[item.Name] || (item.Name != "ByFTP.exe" && item.Name != "Uninstall.exe") {
			return errors.New("installer manifest does not match the package")
		}
		seen[item.Name] = true

		expectedDigest, err := hex.DecodeString(item.SHA256)
		if err != nil || len(expectedDigest) != sha256.Size {
			return errors.New("installer manifest contains an invalid SHA-256 digest")
		}

		actualDigest := sha256.Sum256(content)
		if item.Size != len(content) || !bytes.Equal(expectedDigest, actualDigest[:]) {
			return errors.New("installer package integrity verification failed")
		}
	}

	if !seen["ByFTP.exe"] || !seen["Uninstall.exe"] {
		return errors.New("installer manifest is incomplete")
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
		return errors.New("installer file size verification failed")
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
		return errors.New("installer file size verification failed")
	}
	if !bytes.Equal(h.Sum(nil), expectedDigest[:]) {
		return errors.New("installer file integrity verification failed")
	}

	return nil
}

func installFile(path string, data []byte, backup *fileBackup) error {
	if backup == nil || backup.target != path {
		return errors.New("installer transaction does not match the target file")
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

func register(appPath, uninstallPath, dir, language string) error {
	entries := [...]struct {
		key   string
		name  string
		value string
	}{
		{uninstallKey, "DisplayName", brand.ProductFull},
		{uninstallKey, "DisplayVersion", version},
		{uninstallKey, "InstallLanguage", language},
		{uninstallKey, "Publisher", brand.Company},
		{uninstallKey, "InstallLocation", dir},
		{uninstallKey, "DisplayIcon", appPath},
		{uninstallKey, "UninstallString", `"` + uninstallPath + `"`},
		{appPathsKey, "", appPath},
	}

	for _, entry := range entries {
		if err := platform.SetRegistryString(entry.key, entry.name, entry.value); err != nil {
			return fmt.Errorf("Windows registration failed (%s): %w", entry.name, err)
		}
	}

	if err := platform.SetRegistryString(uninstallKey, "InstallDate", time.Now().Format("20060102")); err != nil {
		return fmt.Errorf("Windows registration failed (InstallDate): %w", err)
	}
	if err := platform.SetRegistryDWORD(uninstallKey, "NoModify", 1); err != nil {
		return fmt.Errorf("Windows registration failed (NoModify): %w", err)
	}
	if err := platform.SetRegistryDWORD(uninstallKey, "NoRepair", 1); err != nil {
		return fmt.Errorf("Windows registration failed (NoRepair): %w", err)
	}

	return nil
}

func installerTitle() string {
	return brand.ProductName + " Setup"
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
				"Setup did not finish",
				"Setup did not finish. Restart Windows and try again.",
			)
			exitCode = 1
		}
	}()

	installLanguage, ok := selectInstallerLanguage()
	if !ok {
		return 0
	}

	content := brand.Website + "\n" + brand.Support + "\n\n" +
		"ByFTP will be installed for your Windows user account and will be available from the Start menu."

	if !platform.ConfirmDialog(
		installerTitle(),
		"Install "+brand.ProductFull+" "+version+"?",
		content,
	) {
		return 0
	}

	// Validate the embedded package before changing the filesystem or registry.
	app, uninstaller, err := readPayloadFiles()
	if err != nil {
		showInstallError(
			"Installer package is invalid",
			"Download a fresh copy of the installer and try again.",
		)
		return 1
	}

	dir, err := platform.InstallDir()
	if err != nil {
		showInstallError(
			"Setup cannot continue",
			"The Windows user folder is unavailable.",
		)
		return 1
	}

	localAppData, err := platform.LocalAppData()
	if err != nil {
		showInstallError(
			"Setup cannot continue",
			"The Windows local application-data folder is unavailable.",
		)
		return 1
	}
	if err := security.EnsureNoRedirectDirectory(localAppData, dir); err != nil {
		showInstallError(
			"The installation folder is not safe",
			"Remove any redirect from the ByFTP installation folder and try again.",
		)
		return 1
	}

	if err := ensureInstallDir(dir); err != nil {
		showInstallError(
			"Setup was not started",
			"The installation folder could not be prepared. Check free space and permissions, then try again.",
		)
		return 1
	}

	appPath := filepath.Join(dir, "ByFTP.exe")
	uninstallPath := filepath.Join(dir, "Uninstall.exe")

	appBackup, err := backupExisting(appPath)
	if err != nil {
		showInstallError(
			"Upgrade was not started",
			"The existing installation could not be prepared for upgrade. Close ByFTP and try again.",
		)
		return 1
	}

	uninstallBackup, err := backupExisting(uninstallPath)
	if err != nil {
		appBackup.cleanup()
		showInstallError(
			"Upgrade was not started",
			"The existing installation could not be prepared for upgrade. Close ByFTP and try again.",
		)
		return 1
	}

	registryBackup, err := captureRegistrySnapshot()
	if err != nil {
		appBackup.cleanup()
		uninstallBackup.cleanup()
		showInstallError(
			"Upgrade was not started",
			"Existing installation settings could not be prepared safely. Try again.",
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
			return " The previous installation could not be fully restored; restart Windows before trying again."
		}
		return ""
	}

	if err := installFile(appPath, app, &appBackup); err != nil {
		extra := rollbackMessage()
		showInstallError(
			"ByFTP could not be installed",
			"Close ByFTP if it is running and try again."+extra,
		)
		return 1
	}

	if err := installFile(uninstallPath, uninstaller, &uninstallBackup); err != nil {
		extra := rollbackMessage()
		showInstallError(
			"Setup did not finish",
			"Required files could not be saved. Try again."+extra,
		)
		return 1
	}

	if err := register(appPath, uninstallPath, dir, installLanguage); err != nil {
		extra := rollbackMessage()
		showInstallError(
			"Setup did not finish",
			"Windows could not finish registering the application. Try again."+extra,
		)
		return 1
	}

	transactionCommitted = true

	languageWarning := ""
	if err := persistInstallerLanguage(installLanguage); err != nil {
		languageWarning = "\n\nThe selected language could not be saved. ByFTP will start in English; you can change the language in Settings."
	}

	shortcutWarning := ""
	if err := platform.CreateShortcuts(appPath); err != nil {
		shortcutWarning = "\n\nA shortcut could not be created. You can start ByFTP from its installation folder."
	}

	if platform.ConfirmDialog(
		installerTitle(),
		"Setup completed successfully",
		"ByFTP is ready to use."+languageWarning+shortcutWarning+"\n\nLaunch ByFTP now?",
	) {
		cmd := exec.Command(appPath)
		if err := cmd.Start(); err != nil {
			platform.ErrorDialog(
				installerTitle(),
				"ByFTP is installed",
				"ByFTP could not be launched automatically. Start it from the Windows Start menu.",
			)
		}
	}

	return 0
}

func main() {
	platform.HardenProcessPrivacy()
	os.Exit(runInstaller())
}
