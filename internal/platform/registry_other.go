//go:build !windows

package platform

import "os"

func SetRegistryString(string, string, string) error         { return nil }
func SetRegistryDWORD(string, string, uint32) error          { return nil }
func GetRegistryString(string, string) (string, bool, error) { return "", false, nil }
func GetRegistryDWORD(string, string) (uint32, bool, error)  { return 0, false, nil }
func DeleteRegistryValue(string, string) error               { return nil }
func DeleteRegistryKey(string) error                         { return nil }
func ReplaceFile(src, dst string) error                      { return os.Rename(src, dst) }
