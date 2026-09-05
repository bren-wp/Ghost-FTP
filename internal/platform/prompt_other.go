//go:build linux

package platform

func PromptDialog(string, string, string) (string, bool) { return "", false }
func PromptDialogWithLabels(string, string, string, string, string) (string, bool) {
	return "", false
}
