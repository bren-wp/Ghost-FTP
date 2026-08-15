//go:build !windows

package platform

func ShortcutPaths() (string, string, error) { return "", "", nil }
func CreateShortcuts(string) error           { return nil }
func RemoveShortcuts() error                 { return nil }
