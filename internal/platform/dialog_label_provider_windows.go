//go:build windows

package platform

import "sync"

type dialogLabelProvider func() (okLabel, cancelLabel, yesLabel, noLabel string)

var (
	dialogLabelProviderMu sync.RWMutex
	dialogLabelsProvider  dialogLabelProvider
)

// SetDialogLabelProvider installs a desktop-owned resolver for application
// dialog action labels. The callback is evaluated when a dialog opens, so
// runtime language changes are reflected without duplicating the i18n catalog
// inside the platform package.
func SetDialogLabelProvider(provider func() (string, string, string, string)) {
	dialogLabelProviderMu.Lock()
	dialogLabelsProvider = provider
	dialogLabelProviderMu.Unlock()
}

func resolvedDialogLabels() (okLabel, cancelLabel, yesLabel, noLabel string) {
	dialogLabelProviderMu.RLock()
	provider := dialogLabelsProvider
	dialogLabelProviderMu.RUnlock()
	if provider != nil {
		okLabel, cancelLabel, yesLabel, noLabel = provider()
	}
	if okLabel == "" || cancelLabel == "" {
		fallbackOK, fallbackCancel := dialogActionLabels()
		if okLabel == "" {
			okLabel = fallbackOK
		}
		if cancelLabel == "" {
			cancelLabel = fallbackCancel
		}
	}
	if yesLabel == "" || noLabel == "" {
		fallbackYes, fallbackNo := dialogDecisionLabels()
		if yesLabel == "" {
			yesLabel = fallbackYes
		}
		if noLabel == "" {
			noLabel = fallbackNo
		}
	}
	return okLabel, cancelLabel, yesLabel, noLabel
}
