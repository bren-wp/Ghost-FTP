//go:build !linux && !darwin

package remote

import "errors"

func findTrustedTransportExecutable(string) (string, error) {
	return "", errors.New("trusted transport resolution is unavailable")
}
