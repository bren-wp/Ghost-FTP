package main

import (
	"errors"

	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

type registryStringSnapshot struct {
	key     string
	name    string
	value   string
	existed bool
}

type registryDWORDSnapshot struct {
	key     string
	name    string
	value   uint32
	existed bool
}

type registryKeySnapshot struct {
	key     string
	existed bool
}

type registrySnapshot struct {
	strings []registryStringSnapshot
	dwords  []registryDWORDSnapshot
	keys    []registryKeySnapshot
}

var installerStringRegistryValues = []struct{ key, name string }{
	{appPathsKey, ""},
	{uninstallKey, "DisplayName"},
	{uninstallKey, "DisplayVersion"},
	{uninstallKey, "Publisher"},
	{uninstallKey, "InstallLocation"},
	{uninstallKey, "DisplayIcon"},
	{uninstallKey, "UninstallString"},
	{uninstallKey, installedExecutableDigestValue},
	{uninstallKey, "QuietUninstallString"},
	{uninstallKey, "URLInfoAbout"},
	{browserProtocolKey, ""},
	{browserProtocolKey, "URL Protocol"},
	{browserProtocolIconKey, ""},
	{browserProtocolCommandKey, ""},
}

var installerDWORDRegistryValues = []struct{ key, name string }{
	{uninstallKey, "NoModify"},
	{uninstallKey, "NoRepair"},
}

// Track protocol key existence separately from the values Ghost FTP owns.
// A key may predate this installer and contain unrelated named values even when
// all tracked defaults are absent. Rollback must preserve such keys verbatim.
var installerProtocolRegistryKeys = []string{
	browserProtocolKey,
	browserProtocolIconKey,
	browserProtocolCommandKey,
}

func captureRegistrySnapshot() (registrySnapshot, error) {
	var out registrySnapshot
	for _, key := range installerProtocolRegistryKeys {
		existed, err := platform.RegistryKeyExists(key)
		if err != nil {
			return registrySnapshot{}, err
		}
		out.keys = append(out.keys, registryKeySnapshot{key: key, existed: existed})
	}
	for _, item := range installerStringRegistryValues {
		value, existed, err := platform.GetRegistryString(item.key, item.name)
		if err != nil {
			return registrySnapshot{}, err
		}
		out.strings = append(out.strings, registryStringSnapshot{
			key: item.key, name: item.name, value: value, existed: existed,
		})
	}
	for _, item := range installerDWORDRegistryValues {
		value, existed, err := platform.GetRegistryDWORD(item.key, item.name)
		if err != nil {
			return registrySnapshot{}, err
		}
		out.dwords = append(out.dwords, registryDWORDSnapshot{
			key: item.key, name: item.name, value: value, existed: existed,
		})
	}
	return out, nil
}

func (s registrySnapshot) stringValue(key, name string) (string, bool) {
	for _, item := range s.strings {
		if item.key == key && item.name == name && item.existed {
			return item.value, true
		}
	}
	return "", false
}

func (s registrySnapshot) keyExisted(key string) bool {
	for _, item := range s.keys {
		if item.key == key {
			return item.existed
		}
	}
	// Fail closed: an untracked key is treated as pre-existing so rollback never
	// deletes registry structure without an explicit pre-transaction snapshot.
	return true
}

func (s registrySnapshot) restore() error {
	var errs []error
	for _, item := range s.strings {
		var err error
		if item.existed {
			err = platform.SetRegistryString(item.key, item.name, item.value)
		} else {
			err = platform.DeleteRegistryValue(item.key, item.name)
		}
		if err != nil {
			errs = append(errs, err)
		}
	}
	for _, item := range s.dwords {
		var err error
		if item.existed {
			err = platform.SetRegistryDWORD(item.key, item.name, item.value)
		} else {
			err = platform.DeleteRegistryValue(item.key, item.name)
		}
		if err != nil {
			errs = append(errs, err)
		}
	}

	// Remove only protocol keys that did not exist before this transaction, and
	// do so leaf-to-root. Pre-existing keys — including keys with unrelated named
	// values that Ghost FTP never snapshots — are never deleted.
	for _, key := range []string{browserProtocolCommandKey, browserProtocolIconKey, browserProtocolKey} {
		if s.keyExisted(key) {
			continue
		}
		if err := platform.DeleteRegistryKey(key); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
