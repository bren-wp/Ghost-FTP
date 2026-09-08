//go:build windows

package platform

import "sync"

type pickerLabelProvider func() (privateKeyTitle, privateKeyFilter, allFilesLabel, directoryTitle string)

var (
	pickerLabelProviderMu sync.RWMutex
	pickerLabelsProvider  pickerLabelProvider
)

// SetPickerLabelProvider installs a desktop-owned resolver for labels shown by
// native Windows file and folder pickers. The callback is evaluated when a
// picker opens so runtime locale switches do not require a second cached i18n
// state inside the platform package.
func SetPickerLabelProvider(provider func() (string, string, string, string)) {
	pickerLabelProviderMu.Lock()
	pickerLabelsProvider = provider
	pickerLabelProviderMu.Unlock()
}

func resolvedPickerLabels() (privateKeyTitle, privateKeyFilter, allFilesLabel, directoryTitle string) {
	pickerLabelProviderMu.RLock()
	provider := pickerLabelsProvider
	pickerLabelProviderMu.RUnlock()
	if provider != nil {
		privateKeyTitle, privateKeyFilter, allFilesLabel, directoryTitle = provider()
	}
	if privateKeyTitle == "" {
		privateKeyTitle = "Select SSH private key"
	}
	if privateKeyFilter == "" {
		privateKeyFilter = "SSH private keys"
	}
	if allFilesLabel == "" {
		allFilesLabel = "All files"
	}
	if directoryTitle == "" {
		directoryTitle = "Select local folder for Ghost FTP"
	}
	return privateKeyTitle, privateKeyFilter, allFilesLabel, directoryTitle
}
