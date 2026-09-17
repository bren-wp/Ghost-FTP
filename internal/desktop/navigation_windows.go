//go:build windows

package desktop

const (
	idFilesNav         = 700
	idSiteManager      = 701
	idTransferQueueNav = 702
	idDiagnostics      = 703
)

// nativeMenuWords remains as a narrow compatibility shim for one localization
// call site. The native menu itself and its unused translated nouns are gone;
// only the canonical site-manager and diagnostics nouns are populated. The
// visible application rail relabels the Site Manager entry as Connections.
func nativeMenuWords(language string) [9]string {
	labels := navigationLabelsForLanguage(language)
	var words [9]string
	words[5] = labels.SiteManager
	words[8] = labels.Diagnostics
	return words
}
