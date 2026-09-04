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
	"strings"

	"github.com/bren-wp/by-ftp/internal/brand"
	"github.com/bren-wp/by-ftp/internal/platform"
	"github.com/bren-wp/by-ftp/internal/security"
)

//go:embed all:payload
var payload embed.FS

const (
	maxPayloadFileSize     = 128 << 20
	maxPayloadManifestSize = 64 << 10
	payloadSchema          = 2

	// These registry/application-path identifiers are retained for upgrade
	// compatibility with installations created before the Ghost FTP rebrand.
	legacyUninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\ByFTP`
	appPathsKey        = `Software\Microsoft\Windows\CurrentVersion\App Paths\ByFTP.exe`
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

func readPayloadFile() ([]byte, error) {
	data, err := payload.ReadFile("payload/payload.zip")
	if err != nil {
		return nil, errors.New("compressed installer payload is unavailable")
	}
	return parsePayload(data)
}

func parsePayload(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("installer payload is empty")
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, errors.New("installer payload is not a valid ZIP archive")
	}

	files := make(map[string][]byte, 1)
	var manifestData []byte

	for _, f := range zr.File {
		switch f.Name {
		case "ByFTP.exe":
			if _, exists := files[f.Name]; exists {
				return nil, errors.New("installer payload contains a duplicate file")
			}

			b, err := readZipEntry(f, maxPayloadFileSize)
			if err != nil {
				return nil, err
			}
			files[f.Name] = b

		case "manifest.json":
			if manifestData != nil {
				return nil, errors.New("installer payload contains a duplicate manifest")
			}

			manifestData, err = readZipEntry(f, maxPayloadManifestSize)
			if err != nil {
				return nil, err
			}

		default:
			return nil, errors.New("installer payload contains an unexpected file")
		}
	}

	app, appOK := files["ByFTP.exe"]
	if !appOK || manifestData == nil {
		return nil, errors.New("installer payload is missing required files")
	}

	if err := validatePayloadManifest(manifestData, files); err != nil {
		return nil, err
	}

	return app, nil
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
	if manifest.Schema != payloadSchema || len(manifest.Files) != 1 {
		return errors.New("installer manifest is unsupported")
	}

	seen := make(map[string]bool, 1)
	for _, item := range manifest.Files {
		content, ok := files[item.Name]
		if !ok || seen[item.Name] || item.Name != "ByFTP.exe" {
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

	if !seen["ByFTP.exe"] {
		return errors.New("installer manifest is incomplete")
	}

	return nil
}

func stageVerified(path string, data []byte) (string, error) {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".ghostftp-install-*.tmp")
	if err != nil {
		return "", err
	}

	tmp := f.Name()
	cleanup := func() {
		_ = f.Close()
		_ = os.Remove(tmp)
	}

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

func register(appPath string) error {
	if err := platform.SetRegistryString(appPathsKey, "", appPath); err != nil {
		return fmt.Errorf("Windows application-path registration failed: %w", err)
	}
	return nil
}

func cleanupLegacyUninstaller(dir string) string {
	var warnings []string
	legacyPath := filepath.Join(dir, "Uninstall.exe")
	info, err := os.Lstat(legacyPath)
	if err == nil {
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || security.IsReparsePoint(legacyPath) {
			warnings = append(warnings, "A legacy Uninstall.exe entry was not removed because it is not a safe regular file.")
		} else if err := os.Remove(legacyPath); err != nil {
			warnings = append(warnings, "The legacy Uninstall.exe file could not be removed. Close applications using it and delete it manually.")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		warnings = append(warnings, "The legacy Uninstall.exe path could not be checked safely.")
	}

	if err := platform.DeleteRegistryKey(legacyUninstallKey); err != nil {
		warnings = append(warnings, "The legacy Windows uninstall registry entry could not be removed.")
	}

	if len(warnings) == 0 {
		return ""
	}
	return "\n\n" + strings.Join(warnings, " ")
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
		brand.ProductName + " will be installed for your Windows user account and will be available from the Start menu."

	if !platform.ConfirmDialog(
		installerTitle(),
		"Install "+brand.ProductFull+" "+version+"?",
		content,
	) {
		return 0
	}

	app, err := readPayloadFile()
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
			"Remove any redirect from the Ghost FTP installation folder and try again.",
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

	// Keep the legacy executable filename for in-place upgrades and existing
	// shortcuts/App Paths registrations. All user-visible branding is Ghost FTP.
	appPath := filepath.Join(dir, "ByFTP.exe")
	appBackup, err := backupExisting(appPath)
	if err != nil {
		showInstallError(
			"Upgrade was not started",
			"The existing installation could not be prepared for upgrade. Close Ghost FTP and try again.",
		)
		return 1
	}

	registryBackup, err := captureRegistrySnapshot()
	if err != nil {
		appBackup.cleanup()
		showInstallError(
			"Upgrade was not started",
			"Existing application-path settings could not be prepared safely. Try again.",
		)
		return 1
	}

	freshInstall := !appBackup.existed()
	transactionCommitted := false
	rolledBack := false

	rollback := func() error {
		if rolledBack {
			return nil
		}
		rolledBack = true

		var errs []error
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
			if err := platform.DeleteRegistryKey(appPathsKey); err != nil {
				errs = append(errs, err)
			}
		}

		return errors.Join(errs...)
	}

	defer func() {
		if !transactionCommitted && !rolledBack {
			_ = rollback()
		}
		appBackup.cleanup()
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
			brand.ProductName+" could not be installed",
			"Close Ghost FTP if it is running and try again."+extra,
		)
		return 1
	}

	if err := register(appPath); err != nil {
		extra := rollbackMessage()
		showInstallError(
			"Setup did not finish",
			"Windows could not finish registering the application. Try again."+extra,
		)
		return 1
	}

	transactionCommitted = true
	legacyCleanupWarning := cleanupLegacyUninstaller(dir)

	languageWarning := ""
	if err := persistInstallerLanguage(installLanguage); err != nil {
		languageWarning = "\n\nThe selected language could not be saved. Ghost FTP will start in English; you can change the language in Settings."
	}

	shortcutWarning := ""
	if err := platform.CreateShortcuts(appPath); err != nil {
		shortcutWarning = "\n\nA shortcut could not be created. You can start Ghost FTP from its installation folder."
	}

	if platform.ConfirmDialog(
		installerTitle(),
		"Setup completed successfully",
		"Ghost FTP is ready to use."+legacyCleanupWarning+languageWarning+shortcutWarning+"\n\nLaunch Ghost FTP now?",
	) {
		cmd := exec.Command(appPath)
		if err := cmd.Start(); err != nil {
			platform.ErrorDialog(
				installerTitle(),
				"Ghost FTP is installed",
				"Ghost FTP could not be launched automatically. Start it from the Windows Start menu.",
			)
		}
	}

	return 0
}

func main() {
	platform.HardenProcessPrivacy()
	os.Exit(runInstaller())
}
