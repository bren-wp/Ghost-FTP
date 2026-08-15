//go:build !windows

package security

import "errors"

func ProtectBytes([]byte) (string, error) {
	return "", errors.New("DPAPI je dostupan samo na Windowsu")
}
func ProtectString(string) (string, error) {
	return "", errors.New("DPAPI je dostupan samo na Windowsu")
}
func UnprotectString(string) (string, error) {
	return "", errors.New("DPAPI je dostupan samo na Windowsu")
}
func UnprotectBytes(string) ([]byte, error) {
	return nil, errors.New("DPAPI je dostupan samo na Windowsu")
}
func WipeBytes(data []byte) {
	for i := range data {
		data[i] = 0
	}
}
