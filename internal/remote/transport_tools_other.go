//go:build !linux

package remote

import "errors"

func findTrustedTransportExecutable(string) (string, error) {
	return "", errors.New("trusted Linux transport resolution is unavailable")
}
