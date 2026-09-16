package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

const (
	browserProtocolKey        = `Software\Classes\ghostftp`
	browserProtocolIconKey    = browserProtocolKey + `\DefaultIcon`
	browserProtocolShellKey   = browserProtocolKey + `\shell`
	browserProtocolOpenKey    = browserProtocolShellKey + `\open`
	browserProtocolCommandKey = browserProtocolOpenKey + `\command`
)

func browserProtocolCommand(appPath string) (string, error) {
	if strings.TrimSpace(appPath) == "" || strings.ContainsAny(appPath, "\"\r\n\x00") {
		return "", errors.New("application path is not safe for protocol registration")
	}
	return `"` + appPath + `" "%1"`, nil
}

func registerBrowserProtocol(appPath string) error {
	command, err := browserProtocolCommand(appPath)
	if err != nil {
		return err
	}
	icon := `"` + appPath + `",0`
	values := []struct {
		key   string
		name  string
		value string
	}{
		{browserProtocolKey, "", "URL:Ghost FTP Protocol"},
		{browserProtocolKey, "URL Protocol", ""},
		{browserProtocolIconKey, "", icon},
		{browserProtocolCommandKey, "", command},
	}
	for _, item := range values {
		if err := platform.SetRegistryString(item.key, item.name, item.value); err != nil {
			return fmt.Errorf("%s: %w", item.key, err)
		}
	}
	return nil
}

func removeBrowserProtocolRegistrationKeys() error {
	var errs []error
	for _, key := range []string{
		browserProtocolCommandKey,
		browserProtocolOpenKey,
		browserProtocolShellKey,
		browserProtocolIconKey,
		browserProtocolKey,
	} {
		if err := platform.DeleteRegistryKey(key); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
