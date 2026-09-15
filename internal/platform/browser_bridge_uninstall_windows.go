//go:build windows

package platform

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

const (
	ghostFTPInstalledNativeHostPathValue        = "InstalledNativeHostPath"
	ghostFTPInstalledNativeHostDigestValue      = "InstalledNativeHostSHA256"
	ghostFTPInstalledFirefoxManifestPathValue   = "InstalledFirefoxManifestPath"
	ghostFTPInstalledFirefoxManifestDigestValue = "InstalledFirefoxManifestSHA256"
	ghostFTPFirefoxNativeMessagingKey           = `Software\Mozilla\NativeMessagingHosts\com.ghostftp.bridge`
	ghostFTPBrowserNativeHostExecutable         = "GhostFTPNativeHost.exe"
	ghostFTPBrowserFirefoxManifestName          = "com.ghostftp.bridge.firefox.json"
)

type browserBridgeOwnedFile struct {
	path   string
	digest string
}

func readBrowserBridgeOwnedFile(pathValue, digestValue, expectedPath string) (browserBridgeOwnedFile, error) {
	path, ok, err := GetRegistryString(ghostFTPUninstallKey, pathValue)
	if err != nil || !ok || !sameCanonicalWindowsPath(path, expectedPath) {
		return browserBridgeOwnedFile{}, errors.New("browser bridge ownership path is unavailable")
	}
	digest, ok, err := GetRegistryString(ghostFTPUninstallKey, digestValue)
	if err != nil || !ok || !validSHA256Digest(digest) {
		return browserBridgeOwnedFile{}, errors.New("browser bridge ownership digest is unavailable")
	}
	digest = strings.ToLower(strings.TrimSpace(digest))
	actual, err := VerifiedRegularFileSHA256(expectedPath)
	if err != nil || !strings.EqualFold(actual, digest) {
		return browserBridgeOwnedFile{}, errors.New("browser bridge file no longer matches its ownership digest")
	}
	return browserBridgeOwnedFile{path: expectedPath, digest: digest}, nil
}

func verifyBrowserBridgeOwnership(installDir string) (browserBridgeOwnedFile, browserBridgeOwnedFile, error) {
	installDir, err := canonicalWindowsPath(installDir)
	if err != nil {
		return browserBridgeOwnedFile{}, browserBridgeOwnedFile{}, err
	}
	nativeExpected := filepath.Join(installDir, ghostFTPBrowserNativeHostExecutable)
	manifestExpected := filepath.Join(installDir, ghostFTPBrowserFirefoxManifestName)

	nativeHost, err := readBrowserBridgeOwnedFile(
		ghostFTPInstalledNativeHostPathValue,
		ghostFTPInstalledNativeHostDigestValue,
		nativeExpected,
	)
	if err != nil {
		return browserBridgeOwnedFile{}, browserBridgeOwnedFile{}, fmt.Errorf("native host ownership verification failed: %w", err)
	}
	manifest, err := readBrowserBridgeOwnedFile(
		ghostFTPInstalledFirefoxManifestPathValue,
		ghostFTPInstalledFirefoxManifestDigestValue,
		manifestExpected,
	)
	if err != nil {
		return browserBridgeOwnedFile{}, browserBridgeOwnedFile{}, fmt.Errorf("Firefox manifest ownership verification failed: %w", err)
	}

	registered, ok, err := GetRegistryString(ghostFTPFirefoxNativeMessagingKey, "")
	if err != nil {
		return browserBridgeOwnedFile{}, browserBridgeOwnedFile{}, err
	}
	if ok && !sameCanonicalWindowsPath(registered, manifest.path) {
		return browserBridgeOwnedFile{}, browserBridgeOwnedFile{}, errors.New("Firefox native messaging registration no longer belongs to this installation")
	}
	return nativeHost, manifest, nil
}

func removeVerifiedBrowserBridgeFile(file browserBridgeOwnedFile, allowDeferred bool) error {
	removed, err := RemoveVerifiedRegularFileMatchingSHA256(file.path, file.digest)
	if err == nil {
		if removed {
			return nil
		}
		return errors.New("verified browser bridge file disappeared before removal")
	}
	if !allowDeferred {
		return err
	}

	// A running native host can keep its image mapped. Re-prove both path and
	// digest immediately before scheduling the deterministic install path for
	// reboot cleanup; never schedule a foreign or modified file.
	actual, verifyErr := VerifiedRegularFileSHA256(file.path)
	if verifyErr != nil || !strings.EqualFold(actual, file.digest) {
		return errors.Join(err, errors.New("locked native host could not be re-verified for deferred cleanup"))
	}
	if scheduleErr := ScheduleDeleteOnReboot(file.path); scheduleErr != nil {
		return errors.Join(err, scheduleErr)
	}
	return nil
}

// RemoveInstalledBrowserBridge removes only the Firefox native-messaging
// registration and companion files whose deterministic paths and SHA-256
// ownership metadata still match this Ghost FTP installation.
func RemoveInstalledBrowserBridge(installDir string) error {
	nativeHost, manifest, err := verifyBrowserBridgeOwnership(installDir)
	if err != nil {
		return err
	}

	var errs []error
	if err := DeleteRegistryKey(ghostFTPFirefoxNativeMessagingKey); err != nil {
		errs = append(errs, err)
	}
	if err := removeVerifiedBrowserBridgeFile(manifest, false); err != nil {
		errs = append(errs, err)
	}
	if err := removeVerifiedBrowserBridgeFile(nativeHost, true); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
