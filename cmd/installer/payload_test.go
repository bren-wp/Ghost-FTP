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
		if item.name == "GhostFTP.exe" {
			digest := fmt.Sprintf("%x", sha256.Sum256([]byte(item.body)))
			manifest.Files = append(manifest.Files, struct {
				Name   string `json:"name"`
				Size   int    `json:"size"`
				SHA256 string `json:"sha256"`
			}{Name: item.name, Size: len(item.body), SHA256: digest})
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

func TestParsePayloadAcceptsAppOnlySchemaTwo(t *testing.T) {
	data := makePayload(t, []testPayloadFile{{"GhostFTP.exe", "app"}}, true, 2)
	app, err := parsePayload(data)
	if err != nil {
		t.Fatal(err)
	}
	if string(app) != "app" {
		t.Fatalf("unexpected payload app=%q", app)
	}
}

func TestParsePayloadRejectsDuplicateRequiredFile(t *testing.T) {
	data := makePayload(t, []testPayloadFile{{"GhostFTP.exe", "a"}, {"GhostFTP.exe", "b"}}, true, 2)
	_, err := parsePayload(data)
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate rejection, got %v", err)
	}
}

func TestParsePayloadRejectsLegacyUninstallerEntry(t *testing.T) {
	data := makePayload(t, []testPayloadFile{{"GhostFTP.exe", "a"}, {"Uninstall.exe", "legacy"}}, true, 2)
	_, err := parsePayload(data)
	if err == nil || !strings.Contains(err.Error(), "unexpected") {
		t.Fatalf("expected legacy uninstaller entry rejection, got %v", err)
	}
}

func TestParsePayloadRejectsUnexpectedFile(t *testing.T) {
	data := makePayload(t, []testPayloadFile{{"GhostFTP.exe", "a"}, {"extra.dll", "x"}}, true, 2)
	_, err := parsePayload(data)
	if err == nil || !strings.Contains(err.Error(), "unexpected") {
		t.Fatalf("expected unexpected-file rejection, got %v", err)
	}
}

func TestParsePayloadRequiresManifest(t *testing.T) {
	data := makePayload(t, []testPayloadFile{{"GhostFTP.exe", "a"}}, false, 2)
	if _, err := parsePayload(data); err == nil {
		t.Fatal("expected missing manifest to be rejected")
	}
}

func TestParsePayloadRejectsLegacySchemaOne(t *testing.T) {
	data := makePayload(t, []testPayloadFile{{"GhostFTP.exe", "a"}}, true, 1)
	if _, err := parsePayload(data); err == nil {
		t.Fatal("expected legacy payload schema to be rejected")
	}
}

func TestValidatePayloadManifestRejectsTamperedHash(t *testing.T) {
	files := map[string][]byte{"GhostFTP.exe": []byte("app")}
	manifest := []byte(`{"schema":2,"files":[{"name":"GhostFTP.exe","size":3,"sha256":"00"}]}`)
	if err := validatePayloadManifest(manifest, files); err == nil {
		t.Fatal("expected tampered manifest to be rejected")
	}
}
