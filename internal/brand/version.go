package brand

import "strings"

// DisplayVersion returns the user-facing product version.
//
// VERSION and package metadata stay strict X.Y.Z so Windows PE resources,
// Debian metadata and release automation remain machine-readable. During the
// pre-1.0 development line, the UI adds an explicit Beta label. The label
// disappears automatically for 1.0.0 and later stable versions.
func DisplayVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return "dev"
	}
	if strings.HasPrefix(version, "0.") {
		return version + " Beta"
	}
	return version
}
