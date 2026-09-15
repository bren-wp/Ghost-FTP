package main

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

const (
	browserNativeHostName       = "com.ghostftp.bridge"
	browserNativeHostExecutable = "GhostFTPNativeHost.exe"
	browserFirefoxManifestName  = "com.ghostftp.bridge.firefox.json"
	browserFirefoxExtensionID   = "ghostftp-connection-helper@ghostftp.com"
	firefoxNativeMessagingKey   = `Software\Mozilla\NativeMessagingHosts\com.ghostftp.bridge`
)

type firefoxNativeHostDocument struct {
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	Path              string   `json:"path"`
	Type              string   `json:"type"`
	AllowedExtensions []string `json:"allowed_extensions"`
}

func canonicalAbsoluteRegistrationPath(value string) (string, error) {
	if strings.TrimSpace(value) == "" || strings.ContainsAny(value, "\x00\r\n") {
		return "", errors.New("native messaging path is invalid")
	}
	clean := filepath.Clean(value)
	if !filepath.IsAbs(clean) {
		return "", errors.New("native messaging path must be absolute")
	}
	absolute, err := filepath.Abs(clean)
	if err != nil || !filepath.IsAbs(absolute) {
		return "", errors.New("native messaging path is invalid")
	}
	return absolute, nil
}

func firefoxNativeHostManifest(hostPath string) ([]byte, error) {
	absolute, err := canonicalAbsoluteRegistrationPath(hostPath)
	if err != nil {
		return nil, errors.New("native host path is invalid")
	}

	doc := firefoxNativeHostDocument{
		Name:              browserNativeHostName,
		Description:       "Ghost FTP local browser bridge",
		Path:              absolute,
		Type:              "stdio",
		AllowedExtensions: []string{browserFirefoxExtensionID},
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func registerBrowserBridge(manifestPath string) error {
	absolute, err := canonicalAbsoluteRegistrationPath(manifestPath)
	if err != nil {
		return errors.New("native host manifest path is invalid")
	}
	return platform.SetRegistryString(firefoxNativeMessagingKey, "", absolute)
}
