//go:build windows

package desktop

const (
	idFilesNav           = 700
	idSiteManager        = 701
	idTransferQueueNav   = 702
	idDiagnostics        = 703
	idCheckUpdates       = 704
	idPremiumDownload    = 705
	idWorkspaceBack      = 706
	idWorkspaceForward   = 707
	idWorkspaceNewFolder = 708
	idWorkspaceMore      = 709
)

// nativeMenuWords remains as a narrow compatibility shim for one localization
// call site. The native menu itself and its unused translated nouns are gone;
// only the canonical connection-manager and diagnostics nouns are populated.
// Internal Site Manager IDs/classes remain compatibility details; visible UI uses
// the same Connections noun as the application rail.
func nativeMenuWords(language string) [9]string {
	labels := navigationLabelsForLanguage(language)
	var words [9]string
	words[5] = labels.Connections
	words[8] = labels.Diagnostics
	return words
}
