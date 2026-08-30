package remote

import (
	"strings"

	"github.com/bren-wp/by-ftp/internal/model"
)

// ConnectionDiagnostics contains non-secret facts derived from the connection
// that ByFTP already established. It intentionally carries no host, username,
// credentials, certificate material or server banners.
type ConnectionDiagnostics struct {
	Secure          bool   `json:"secure"`
	RootMode        string `json:"rootMode"`
	WebRoot         string `json:"webRoot,omitempty"`
	WebRootDetected bool   `json:"webRootDetected"`
	RootEntryCount  int    `json:"rootEntryCount"`
}

var sharedHostingWebRoots = []string{
	"public_html",
	"httpdocs",
	"htdocs",
	"www",
	"web",
	"html",
}

func diagnoseConnection(protocol string, items []model.Item) ConnectionDiagnostics {
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	diagnostics := ConnectionDiagnostics{
		Secure:         protocol != "ftp",
		RootMode:       "account",
		RootEntryCount: len(items),
	}
	if protocol == "sftp" {
		diagnostics.RootMode = "home"
	}

	directories := make(map[string]string, len(items))
	for _, item := range items {
		if !item.IsDirectory || item.IsSymlink {
			continue
		}
		name := item.Name
		if name == "" {
			continue
		}
		directories[strings.ToLower(name)] = name
	}
	for _, candidate := range sharedHostingWebRoots {
		if actual, ok := directories[candidate]; ok {
			diagnostics.WebRoot = actual
			diagnostics.WebRootDetected = true
			break
		}
	}
	return diagnostics
}
