//go:build !windows

package security

func isReparsePoint(string) bool { return false }
