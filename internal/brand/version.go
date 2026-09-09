package brand

import "strings"

// DisplayVersion returns the canonical user-facing product version.
//
// VERSION and package metadata stay strict X.Y.Z so Windows PE resources,
// Debian metadata and release automation remain machine-readable. Public
// 0.0.x releases are first-class current releases and therefore do not gain
// an automatic maturity suffix based only on their major version.
func DisplayVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return "dev"
	}
	return version
}
