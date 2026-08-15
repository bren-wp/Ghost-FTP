package main

import (
	"errors"

	"brendigo.com/byftp/internal/platform"
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

type registrySnapshot struct {
	strings []registryStringSnapshot
	dwords  []registryDWORDSnapshot
}

var installerStringRegistryValues = []struct{ key, name string }{
	{uninstallKey, "DisplayName"},
	{uninstallKey, "DisplayVersion"},
	{uninstallKey, "Publisher"},
	{uninstallKey, "InstallLocation"},
	{uninstallKey, "DisplayIcon"},
	{uninstallKey, "UninstallString"},
	{uninstallKey, "InstallDate"},
	{`Software\Microsoft\Windows\CurrentVersion\App Paths\ByFTP.exe`, ""},
}

var installerDWORDRegistryValues = []struct{ key, name string }{
	{uninstallKey, "NoModify"},
	{uninstallKey, "NoRepair"},
}

func captureRegistrySnapshot() (registrySnapshot, error) {
	var out registrySnapshot
	for _, item := range installerStringRegistryValues {
		value, existed, err := platform.GetRegistryString(item.key, item.name)
		if err != nil {
			return registrySnapshot{}, err
		}
		out.strings = append(out.strings, registryStringSnapshot{key: item.key, name: item.name, value: value, existed: existed})
	}
	for _, item := range installerDWORDRegistryValues {
		value, existed, err := platform.GetRegistryDWORD(item.key, item.name)
		if err != nil {
			return registrySnapshot{}, err
		}
		out.dwords = append(out.dwords, registryDWORDSnapshot{key: item.key, name: item.name, value: value, existed: existed})
	}
	return out, nil
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
	return errors.Join(errs...)
}
