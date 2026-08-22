//go:build !windows

package platform

func PromptDialog(string, string, string) (string, bool) { return "", false }
func PromptDialogWithLabels(string, string, string, string, string) (string, bool) {
	return "", false
}
