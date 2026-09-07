//go:build windows

package desktop

const (
	idSiteManager = 701
	idDiagnostics = 703
)

// nativeMenuWords remains as a narrow compatibility shim for one localization
// call site. The native menu itself and its unused translated nouns are gone;
// only the two canonical sidebar labels are populated.
func nativeMenuWords(language string) [9]string {
	labels := navigationLabelsForLanguage(language)
	var words [9]string
	words[5] = labels.SiteManager
	words[8] = labels.Diagnostics
	return words
}
