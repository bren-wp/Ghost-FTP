package desktop

import (
	"path"
	"strings"
)

func cleanRemote(p string) string {
	p = strings.ReplaceAll(strings.TrimSpace(p), "\\", "/")
	if p == "" {
		return "."
	}
	p = path.Clean(p)
	if p == "" {
		return "."
	}
	return p
}

func optionalRemotePath(p string) string {
	if strings.TrimSpace(p) == "" {
		return ""
	}
	return cleanRemote(p)
}

func remoteParent(p string) string {
	p = cleanRemote(p)
	if p == "." || p == "/" {
		return p
	}
	return cleanRemote(path.Dir(p))
}
