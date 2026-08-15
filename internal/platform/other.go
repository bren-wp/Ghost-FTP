//go:build !windows

package platform

import (
	"errors"
	"os"
)

func ChooseDirectory() (string, error)            { return "", errors.New("Windows only") }
func ChoosePrivateKey() (string, error)           { return "", errors.New("Windows only") }
func MessageBox(string, string, uintptr) int      { return 0 }
func ConfirmDialog(string, string, string) bool   { return false }
func InfoDialog(string, string, string)           {}
func ErrorDialog(string, string, string)          {}
func AcquireSingleInstance(string) (func(), bool) { return func() {}, true }
func HardenProcessPrivacy()                       {}
func TrustedAskPassParent() bool                  { return false }

func LocalAppData() (string, error)    { return os.UserConfigDir() }
func SystemDirectory() (string, error) { return "", errors.New("Windows only") }
