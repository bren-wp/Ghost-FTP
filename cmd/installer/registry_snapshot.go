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

type registrySnapshot struct {
	strings []registryStringSnapshot
}

var installerStringRegistryValues = []struct{ key, name string }{
	{appPathsKey, ""},
}

func captureRegistrySnapshot() (registrySnapshot, error) {
	var out registrySnapshot
	for _, item := range installerStringRegistryValues {
		value, existed, err := platform.GetRegistryString(item.key, item.name)
		if err != nil {
			return registrySnapshot{}, err
		}
		out.strings = append(out.strings, registryStringSnapshot{
			key: item.key, name: item.name, value: value, existed: existed,
		})
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
	return errors.Join(errs...)
}
