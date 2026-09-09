//go:build windows

package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/bren-wp/Ghost-FTP/internal/brand"
)

const (
	ghostFTPUninstallKey              = `Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTP`
	ghostFTPAppPathsKey               = `Software\Microsoft\Windows\CurrentVersion\App Paths\GhostFTP.exe`
	ghostFTPInstalledDigestValue      = "InstalledExecutableSHA256"
	integratedUninstallFinalizeArg    = "--uninstall-finalize"
	integratedUninstallDigestEnv      = "GHOSTFTP_UNINSTALL_SHA256"
	integratedUninstallHelperPrefix   = ".ghostftp-uninstall-"
	processSynchronizeAccess  uintptr = 0x00100000
	waitObject0               uintptr = 0x00000000
	waitInfinite              uintptr = 0xffffffff
)

var kernel32IntegratedUninstall = syscall.NewLazyDLL("kernel32.dll")
var openProcessIntegratedUninstall = kernel32IntegratedUninstall.NewProc("OpenProcess")
var waitForSingleObjectIntegratedUninstall = kernel32IntegratedUninstall.NewProc("WaitForSingleObject")

func canonicalWindowsPath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("path is unavailable")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func sameCanonicalWindowsPath(a, b string) bool {
	left, err := canonicalWindowsPath(a)
	if err != nil {
		return false
	}
	right, err := canonicalWindowsPath(b)
	if err != nil {
		return false
	}
	return strings.EqualFold(left, right)
}

func validSHA256Digest(value string) bool {
	decoded, err := hex.DecodeString(strings.TrimSpace(value))
	return err == nil && len(decoded) == sha256.Size
}

func installedGhostFTPExecutablePath() (string, error) {
	dir, err := InstallDir()
	if err != nil {
		return "", err
	}
	return canonicalWindowsPath(filepath.Join(dir, "GhostFTP.exe"))
}

func verifyIntegratedUninstallExecutable(exe string) (string, string, error) {
	expected, err := installedGhostFTPExecutablePath()
	if err != nil {
		return "", "", err
	}
	actual, err := canonicalWindowsPath(exe)
	if err != nil {
		return "", "", err
	}
	if !SameExecutableIdentity(actual, expected) {
		return "", "", errors.New("running executable does not match the installed Ghost FTP file object")
	}

	registeredPath, ok, err := GetRegistryString(ghostFTPAppPathsKey, "")
	if err != nil || !ok || !sameCanonicalWindowsPath(registeredPath, expected) {
		return "", "", errors.New("Windows App Paths registration does not prove executable ownership")
	}

	registeredLocation, ok, err := GetRegistryString(ghostFTPUninstallKey, "InstallLocation")
	if err != nil || !ok || !sameCanonicalWindowsPath(registeredLocation, filepath.Dir(expected)) {
		return "", "", errors.New("Windows Installed Apps location does not prove executable ownership")
	}

	registeredCommand, ok, err := GetRegistryString(ghostFTPUninstallKey, "UninstallString")
	expectedCommand := fmt.Sprintf("\"%s\" --uninstall", expected)
	if err != nil || !ok || !strings.EqualFold(strings.TrimSpace(registeredCommand), expectedCommand) {
		return "", "", errors.New("Windows Installed Apps command does not prove executable ownership")
	}

	digest, ok, err := GetRegistryString(ghostFTPUninstallKey, ghostFTPInstalledDigestValue)
	if err != nil || !ok || !validSHA256Digest(digest) {
		return "", "", errors.New("installed executable ownership digest is unavailable")
	}
	digest = strings.ToLower(strings.TrimSpace(digest))

	actualDigest, err := VerifiedRegularFileSHA256(actual)
	if err != nil || !strings.EqualFold(actualDigest, digest) {
		return "", "", errors.New("installed executable no longer matches its ownership digest")
	}
	return expected, digest, nil
}

func isInstalledGhostFTPExecutable(exe string) bool {
	_, _, err := verifyIntegratedUninstallExecutable(exe)
	return err == nil
}

func helperEnvironment(expectedDigest string) []string {
	prefix := strings.ToUpper(integratedUninstallDigestEnv + "=")
	env := make([]string, 0, len(os.Environ())+1)
	for _, item := range os.Environ() {
		if strings.HasPrefix(strings.ToUpper(item), prefix) {
			continue
		}
		env = append(env, item)
	}
	return append(env, integratedUninstallDigestEnv+"="+expectedDigest)
}

func prepareIntegratedUninstallHelper(exe, expectedDigest string) (string, *os.File, error) {
	source, err := openVerifiedRegularFile(exe, false)
	if err != nil {
		return "", nil, err
	}
	defer source.Close()

	digest, err := verifiedOpenFileSHA256(source)
	if err != nil || !strings.EqualFold(digest, expectedDigest) {
		return "", nil, errors.New("running executable changed before uninstall helper creation")
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		return "", nil, err
	}

	dir := filepath.Dir(exe)
	helper, err := os.CreateTemp(dir, integratedUninstallHelperPrefix+"*.exe")
	if err != nil {
		return "", nil, err
	}
	helperPath := helper.Name()
	cleanup := func() {
		_ = helper.Close()
		_ = os.Remove(helperPath)
	}

	if _, err := io.Copy(helper, source); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := helper.Sync(); err != nil {
		cleanup()
		return "", nil, err
	}
	if err := helper.Close(); err != nil {
		_ = os.Remove(helperPath)
		return "", nil, err
	}

	pinned, err := openVerifiedRegularFile(helperPath, false)
	if err != nil {
		_ = os.Remove(helperPath)
		return "", nil, err
	}
	helperDigest, err := verifiedOpenFileSHA256(pinned)
	if err != nil || !strings.EqualFold(helperDigest, expectedDigest) {
		_ = pinned.Close()
		_ = os.Remove(helperPath)
		return "", nil, errors.New("uninstall helper integrity verification failed")
	}
	return helperPath, pinned, nil
}

func launchIntegratedUninstallHelper(exe, expectedDigest string) error {
	helperPath, pinned, err := prepareIntegratedUninstallHelper(exe, expectedDigest)
	if err != nil {
		return err
	}

	cmd := exec.Command(helperPath, integratedUninstallFinalizeArg, strconv.Itoa(os.Getpid()))
	cmd.Env = helperEnvironment(expectedDigest)
	startErr := cmd.Start()
	_ = pinned.Close()
	if startErr != nil {
		_ = os.Remove(helperPath)
		return startErr
	}
	return cmd.Process.Release()
}

func waitForUninstallParent(pid uint32) error {
	h, _, callErr := openProcessIntegratedUninstall.Call(processSynchronizeAccess, 0, uintptr(pid))
	if h == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return callErr
		}
		return errors.New("uninstall parent process is unavailable")
	}
	defer syscall.CloseHandle(syscall.Handle(h))

	result, _, callErr := waitForSingleObjectIntegratedUninstall.Call(h, waitInfinite)
	if result != waitObject0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return callErr
		}
		return errors.New("uninstall parent process wait failed")
	}
	return nil
}

func handleIntegratedUninstallFinalizer(args []string) (bool, int) {
	if len(args) != 3 || !strings.EqualFold(strings.TrimSpace(args[1]), integratedUninstallFinalizeArg) {
		return false, 0
	}

	exe, err := os.Executable()
	if err != nil {
		return true, 1
	}
	exe, err = canonicalWindowsPath(exe)
	if err != nil {
		return true, 1
	}
	expected, err := installedGhostFTPExecutablePath()
	if err != nil || sameCanonicalWindowsPath(exe, expected) {
		return true, 1
	}
	if !sameCanonicalWindowsPath(filepath.Dir(exe), filepath.Dir(expected)) || !strings.HasPrefix(strings.ToLower(filepath.Base(exe)), integratedUninstallHelperPrefix) {
		return true, 1
	}

	digest := strings.TrimSpace(os.Getenv(integratedUninstallDigestEnv))
	_ = os.Unsetenv(integratedUninstallDigestEnv)
	if !validSHA256Digest(digest) {
		return true, 1
	}
	helperDigest, err := VerifiedRegularFileSHA256(exe)
	if err != nil || !strings.EqualFold(helperDigest, digest) {
		return true, 1
	}

	parent, err := strconv.ParseUint(strings.TrimSpace(args[2]), 10, 32)
	if err != nil || parent == 0 || uint32(parent) == uint32(os.Getpid()) {
		return true, 1
	}
	if err := waitForUninstallParent(uint32(parent)); err != nil {
		return true, 1
	}

	removed, err := RemoveVerifiedRegularFileMatchingSHA256(expected, digest)
	if err != nil || !removed {
		return true, 1
	}

	// Best-effort cleanup of the short-lived helper. The installed application
	// has already been deleted object-bound at this point. If Windows still has
	// the helper image mapped, only the random helper pathname is deferred.
	if _, err := RemoveVerifiedRegularFileMatchingSHA256(exe, digest); err != nil {
		_ = ScheduleDeleteOnReboot(exe)
	}
	return true, 0
}

// HandleIntegratedUninstall implements the Windows Installed Apps uninstall
// entry without shipping a second permanent Uninstall.exe. The installed
// GhostFTP.exe is invoked with --uninstall, proves registry and digest ownership,
// and launches a verified short-lived copy of itself to delete the exact
// installed file object after the interactive parent process exits.
func HandleIntegratedUninstall(args []string) (handled bool, exitCode int) {
	if handled, exitCode := handleIntegratedUninstallFinalizer(args); handled {
		return true, exitCode
	}
	if len(args) != 2 || !strings.EqualFold(strings.TrimSpace(args[1]), "--uninstall") {
		return false, 0
	}

	exe, err := os.Executable()
	if err != nil {
		return true, 1
	}
	_, digest, err := verifyIntegratedUninstallExecutable(exe)
	if err != nil {
		ErrorDialog(
			brand.ProductName+" Setup",
			"Uninstall is unavailable",
			"The installed Ghost FTP application could not prove its installation ownership. Portable copies and replaced executables are never removed by this command.",
		)
		return true, 1
	}

	if !ConfirmDialog(
		brand.ProductName+" Setup",
		"Uninstall Ghost FTP?",
		"Ghost FTP application files and shortcuts will be removed. Saved profiles and settings are kept on this computer so an accidental uninstall does not destroy connection configuration.",
	) {
		return true, 0
	}

	if err := launchIntegratedUninstallHelper(exe, digest); err != nil {
		ErrorDialog(
			brand.ProductName+" Setup",
			"Uninstall could not start safely",
			"Ghost FTP could not prepare verified final cleanup. No installation registration was removed. Close other applications and try again.",
		)
		return true, 1
	}

	var warnings []string
	if err := RemoveShortcuts(); err != nil {
		warnings = append(warnings, "Some shortcuts could not be removed.")
	}
	if err := DeleteRegistryKey(ghostFTPAppPathsKey); err != nil {
		warnings = append(warnings, "The Windows App Paths registration could not be removed.")
	}
	if err := DeleteRegistryKey(ghostFTPUninstallKey); err != nil {
		warnings = append(warnings, "The Windows Installed Apps entry could not be removed.")
	}

	message := "Ghost FTP uninstall is completing. Saved profiles and settings were preserved. The verified application executable will be removed after this window closes."
	if len(warnings) > 0 {
		message += "\n\n" + strings.Join(warnings, " ")
	}
	InfoDialog(brand.ProductName+" Setup", "Uninstall completed", message)
	return true, 0
}
