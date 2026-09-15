package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestFirefoxNativeHostManifestIsExactAndLeastPrivilege(t *testing.T) {
	host := filepath.Join(t.TempDir(), browserNativeHostExecutable)
	data, err := firefoxNativeHostManifest(host)
	if err != nil {
		t.Fatal(err)
	}
	var doc firefoxNativeHostDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Name != browserNativeHostName || doc.Type != "stdio" {
		t.Fatalf("unexpected native host identity: %#v", doc)
	}
	if !samePathForTest(doc.Path, host) {
		t.Fatalf("manifest path mismatch: got %q want %q", doc.Path, host)
	}
	if len(doc.AllowedExtensions) != 1 || doc.AllowedExtensions[0] != browserFirefoxExtensionID {
		t.Fatalf("unexpected Firefox allow-list: %#v", doc.AllowedExtensions)
	}
	if strings.Contains(string(data), "*") {
		t.Fatal("native host manifest must never use wildcard extension access")
	}
}

func TestFirefoxNativeHostManifestRejectsUnsafePath(t *testing.T) {
	for _, path := range []string{"", "relative.exe", "C:\\GhostFTP\\host.exe\nother"} {
		if _, err := firefoxNativeHostManifest(path); err == nil {
			t.Fatalf("unsafe path was accepted: %q", path)
		}
	}
}

func samePathForTest(a, b string) bool {
	left, err := filepath.Abs(filepath.Clean(a))
	if err != nil {
		return false
	}
	right, err := filepath.Abs(filepath.Clean(b))
	if err != nil {
		return false
	}
	return strings.EqualFold(left, right)
}
