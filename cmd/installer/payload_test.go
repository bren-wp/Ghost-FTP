package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

type testPayloadFile struct{ name, body string }

func makePayload(t *testing.T, files []testPayloadFile, includeManifest bool, schema int) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	manifest := payloadManifest{Schema: schema}
	for _, item := range files {
		w, err := zw.Create(item.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(item.body)); err != nil {
			t.Fatal(err)
		}
		if item.name == "GhostFTP.exe" || item.name == browserNativeHostExecutable {
			digest := fmt.Sprintf("%x", sha256.Sum256([]byte(item.body)))
			manifest.Files = append(manifest.Files, payloadManifestFile{
				Name: item.name, Size: len(item.body), SHA256: digest,
			})
		}
	}
	if includeManifest {
		w, err := zw.Create("manifest.json")
		if err != nil {
			t.Fatal(err)
		}
		if err := json.NewEncoder(w).Encode(manifest); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestParsePayloadAcceptsAppAndNativeHostSchemaThree(t *testing.T) {
	data := makePayload(t, []testPayloadFile{
		{"GhostFTP.exe", "app"},
		{browserNativeHostExecutable, "bridge"},
	}, true, 3)
	bundle, err := parsePayload(data)
	if err != nil {
		t.Fatal(err)
	}
	if string(bundle.App) != "app" || string(bundle.NativeHost) != "bridge" {
		t.Fatalf("unexpected payload contents: app=%q native=%q", bundle.App, bundle.NativeHost)
	}
}

func TestParsePayloadRejectsDuplicateRequiredFile(t *testing.T) {
	data := makePayload(t, []testPayloadFile{
		{"GhostFTP.exe", "a"},
		{"GhostFTP.exe", "b"},
		{browserNativeHostExecutable, "bridge"},
	}, true, 3)
	_, err := parsePayload(data)
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate rejection, got %v", err)
	}
}

func TestParsePayloadRejectsLegacyUninstallerEntry(t *testing.T) {
	data := makePayload(t, []testPayloadFile{
		{"GhostFTP.exe", "a"},
		{browserNativeHostExecutable, "bridge"},
		{"Uninstall.exe", "legacy"},
	}, true, 3)
	_, err := parsePayload(data)
	if err == nil || !strings.Contains(err.Error(), "unexpected") {
		t.Fatalf("expected legacy uninstaller entry rejection, got %v", err)
	}
}

func TestParsePayloadRejectsUnexpectedFile(t *testing.T) {
	data := makePayload(t, []testPayloadFile{
		{"GhostFTP.exe", "a"},
		{browserNativeHostExecutable, "bridge"},
		{"extra.dll", "x"},
	}, true, 3)
	_, err := parsePayload(data)
	if err == nil || !strings.Contains(err.Error(), "unexpected") {
		t.Fatalf("expected unexpected-file rejection, got %v", err)
	}
}

func TestParsePayloadRequiresManifest(t *testing.T) {
	data := makePayload(t, []testPayloadFile{
		{"GhostFTP.exe", "a"},
		{browserNativeHostExecutable, "bridge"},
	}, false, 3)
	if _, err := parsePayload(data); err == nil {
		t.Fatal("expected missing manifest to be rejected")
	}
}

func TestParsePayloadRequiresNativeHost(t *testing.T) {
	data := makePayload(t, []testPayloadFile{{"GhostFTP.exe", "a"}}, true, 3)
	if _, err := parsePayload(data); err == nil {
		t.Fatal("expected missing native host to be rejected")
	}
}

func TestParsePayloadRejectsLegacySchemaTwo(t *testing.T) {
	data := makePayload(t, []testPayloadFile{
		{"GhostFTP.exe", "a"},
		{browserNativeHostExecutable, "bridge"},
	}, true, 2)
	if _, err := parsePayload(data); err == nil {
		t.Fatal("expected legacy payload schema to be rejected")
	}
}

func TestValidatePayloadManifestRejectsTamperedHash(t *testing.T) {
	files := map[string][]byte{
		"GhostFTP.exe":              []byte("app"),
		browserNativeHostExecutable: []byte("bridge"),
	}
	bridgeDigest := fmt.Sprintf("%x", sha256.Sum256(files[browserNativeHostExecutable]))
	manifest := []byte(fmt.Sprintf(
		`{"schema":3,"files":[{"name":"GhostFTP.exe","size":3,"sha256":"00"},{"name":"%s","size":6,"sha256":"%s"}]}`,
		browserNativeHostExecutable, bridgeDigest,
	))
	if err := validatePayloadManifest(manifest, files); err == nil {
		t.Fatal("expected tampered application digest to be rejected")
	}
}

func TestValidatePayloadManifestRejectsTamperedNativeHost(t *testing.T) {
	files := map[string][]byte{
		"GhostFTP.exe":              []byte("app"),
		browserNativeHostExecutable: []byte("bridge"),
	}
	appDigest := fmt.Sprintf("%x", sha256.Sum256(files["GhostFTP.exe"]))
	manifest := []byte(fmt.Sprintf(
		`{"schema":3,"files":[{"name":"GhostFTP.exe","size":3,"sha256":"%s"},{"name":"%s","size":6,"sha256":"00"}]}`,
		appDigest, browserNativeHostExecutable,
	))
	if err := validatePayloadManifest(manifest, files); err == nil {
		t.Fatal("expected tampered native-host digest to be rejected")
	}
}
