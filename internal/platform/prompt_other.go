//go:build !windows

package platform

func PromptDialog(string, string, string) (string, bool) { return "", false }
