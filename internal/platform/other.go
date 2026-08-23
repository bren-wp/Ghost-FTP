//go:build !windows

package platform

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

func LocalAppData() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errors.New("korisnička mapa nije dostupna")
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support"), nil
	}
	if xdg := os.Getenv("XDG_DATA_HOME"); filepath.IsAbs(xdg) {
		return xdg, nil
	}
	return filepath.Join(home, ".local", "share"), nil
}

func SystemDirectory() (string, error) { return "", errors.New("Windows sistemska mapa nije dostupna") }
func HardenProcessPrivacy()            {}
func TrustedAskPassParent() bool       { return false }
func ChoosePrivateKey() (string, error) {
	return "", errors.New("odabir privatnog ključa dostupan je iz terminalnog unosa")
}
func ChooseDirectory() (string, error) {
	return "", errors.New("odabir direktorija dostupan je iz terminalnog unosa")
}
func MessageBox(title, text string, flags uintptr) int {
	_, _ = os.Stderr.WriteString(title + ": " + text + "\n")
	return 0
}
func ConfirmDialog(string, string, string) bool { return false }
func InfoDialog(title, instruction, content string) {
	_, _ = os.Stdout.WriteString(title + ": " + instruction + "\n" + content + "\n")
}
func ErrorDialog(title, instruction, content string) {
	_, _ = os.Stderr.WriteString(title + ": " + instruction + "\n" + content + "\n")
}
func AcquireSingleInstance(string) (func(), bool) { return func() {}, true }
