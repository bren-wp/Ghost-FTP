//go:build !windows

package platform

import "errors"

func NativeWindowsArchitecture() (string, error) {
	return "", errors.New("Windows architecture detection is unavailable on this platform")
}
