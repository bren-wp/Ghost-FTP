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

func makePayload(t *testing.T, files []testPayloadFile, includeManifest bool) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	manifest := payloadManifest{Schema: 1}
	for _, item := range files {
		w, err := zw.Create(item.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(item.body)); err != nil {
			t.Fatal(err)
		}
		if item.name == "ByFTP.exe" || item.name == "Uninstall.exe" {
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

func TestParsePayloadAcceptsExactlyRequiredFilesAndManifest(t *testing.T) {
	data := makePayload(t, []testPayloadFile{{"ByFTP.exe", "app"}, {"Uninstall.exe", "un"}}, true)
	app, un, err := parsePayload(data)
	if err != nil {
		t.Fatal(err)
	}
	if string(app) != "app" || string(un) != "un" {
		t.Fatalf("unexpected payload: app=%q un=%q", app, un)
	}
}

func TestParsePayloadRejectsDuplicateRequiredFile(t *testing.T) {
	data := makePayload(t, []testPayloadFile{{"ByFTP.exe", "a"}, {"ByFTP.exe", "b"}, {"Uninstall.exe", "u"}}, true)
	_, _, err := parsePayload(data)
	if err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate rejection, got %v", err)
	}
}

func TestParsePayloadRejectsUnexpectedFile(t *testing.T) {
	data := makePayload(t, []testPayloadFile{{"ByFTP.exe", "a"}, {"Uninstall.exe", "u"}, {"extra.dll", "x"}}, true)
	_, _, err := parsePayload(data)
	if err == nil || !strings.Contains(err.Error(), "unexpected") {
		t.Fatalf("expected unexpected-file rejection, got %v", err)
	}
}

func TestParsePayloadRequiresManifest(t *testing.T) {
	data := makePayload(t, []testPayloadFile{{"ByFTP.exe", "a"}, {"Uninstall.exe", "u"}}, false)
	if _, _, err := parsePayload(data); err == nil {
		t.Fatal("expected missing manifest to be rejected")
	}
}

func TestValidatePayloadManifestRejectsTamperedHash(t *testing.T) {
	files := map[string][]byte{"ByFTP.exe": []byte("app"), "Uninstall.exe": []byte("un")}
	manifest := []byte(`{"schema":1,"files":[{"name":"ByFTP.exe","size":3,"sha256":"00"},{"name":"Uninstall.exe","size":2,"sha256":"00"}]}`)
	if err := validatePayloadManifest(manifest, files); err == nil {
		t.Fatal("expected tampered manifest to be rejected")
	}
}
