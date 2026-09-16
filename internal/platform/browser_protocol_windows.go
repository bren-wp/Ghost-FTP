//go:build windows

package platform

import (
	"errors"
	"fmt"
	"strings"
)

const (
	ghostFTPProtocolKey        = `Software\Classes\ghostftp`
	ghostFTPProtocolIconKey    = ghostFTPProtocolKey + `\DefaultIcon`
	ghostFTPProtocolShellKey   = ghostFTPProtocolKey + `\shell`
	ghostFTPProtocolOpenKey    = ghostFTPProtocolShellKey + `\open`
	ghostFTPProtocolCommandKey = ghostFTPProtocolOpenKey + `\command`
)

func ghostFTPProtocolCommand(exe string) (string, error) {
	canonical, err := canonicalWindowsPath(exe)
	if err != nil || strings.ContainsAny(canonical, "\"\r\n\x00") {
		return "", errors.New("Ghost FTP executable path is unsafe for protocol registration")
	}
	return fmt.Sprintf("\"%s\" \"%%1\"", canonical), nil
}

// RemoveOwnedGhostFTPProtocolRegistration removes only a protocol handler whose
// open command still points at the exact Ghost FTP executable path supplied by
// the caller. A modified or third-party handler is left untouched.
func RemoveOwnedGhostFTPProtocolRegistration(exe string) error {
	expectedCommand, err := ghostFTPProtocolCommand(exe)
	if err != nil {
		return err
	}
	registeredCommand, exists, err := GetRegistryString(ghostFTPProtocolCommandKey, "")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if !strings.EqualFold(strings.TrimSpace(registeredCommand), expectedCommand) {
		return errors.New("Ghost FTP browser protocol registration ownership could not be proven")
	}

	var errs []error
	for _, item := range []struct{ key, name string }{
		{ghostFTPProtocolCommandKey, ""},
		{ghostFTPProtocolIconKey, ""},
		{ghostFTPProtocolKey, "URL Protocol"},
		{ghostFTPProtocolKey, ""},
	} {
		if err := DeleteRegistryValue(item.key, item.name); err != nil {
			errs = append(errs, err)
		}
	}
	for _, key := range []string{
		ghostFTPProtocolCommandKey,
		ghostFTPProtocolOpenKey,
		ghostFTPProtocolShellKey,
		ghostFTPProtocolIconKey,
		ghostFTPProtocolKey,
	} {
		if err := DeleteRegistryKey(key); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
