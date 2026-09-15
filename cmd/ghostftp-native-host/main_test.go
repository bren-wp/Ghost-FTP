package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeFrameRoundTrip(t *testing.T) {
	payload := []byte(`{"id":"1","type":"hello","params":{}}`)
	var framed bytes.Buffer
	if err := binary.Write(&framed, binary.LittleEndian, uint32(len(payload))); err != nil {
		t.Fatal(err)
	}
	if _, err := framed.Write(payload); err != nil {
		t.Fatal(err)
	}
	got, err := readFrame(&framed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("frame mismatch: %q", got)
	}
}

func TestNativeFrameRejectsOversizeInput(t *testing.T) {
	var framed bytes.Buffer
	if err := binary.Write(&framed, binary.LittleEndian, uint32(maxInboundMessage+1)); err != nil {
		t.Fatal(err)
	}
	if _, err := readFrame(&framed); err == nil {
		t.Fatal("expected oversized native message rejection")
	}
}

func TestDecodeRequestRejectsUnknownFields(t *testing.T) {
	_, err := decodeRequest([]byte(`{"id":"1","type":"hello","unexpected":true}`))
	if err == nil {
		t.Fatal("expected unknown request field rejection")
	}
}

func TestDecodeParamsRejectsUnknownFields(t *testing.T) {
	type params struct {
		Path string `json:"path"`
	}
	_, err := decodeStrict[params](json.RawMessage(`{"path":"/","unexpected":true}`))
	if err == nil {
		t.Fatal("expected unknown parameter field rejection")
	}
}

func TestLocalRootConfinement(t *testing.T) {
	root := t.TempDir()
	insideDir := filepath.Join(root, "inside")
	if err := os.Mkdir(insideDir, 0700); err != nil {
		t.Fatal(err)
	}
	insideFile := filepath.Join(insideDir, "file.txt")
	if err := os.WriteFile(insideFile, []byte("ok"), 0600); err != nil {
		t.Fatal(err)
	}

	h := &host{}
	canonicalRoot, err := h.authorizeRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	if canonicalRoot == "" {
		t.Fatal("canonical root is empty")
	}
	resolved, err := h.existingLocalPath(insideFile)
	if err != nil {
		t.Fatal(err)
	}
	if !pathWithin(canonicalRoot, resolved) {
		t.Fatalf("expected %q within %q", resolved, canonicalRoot)
	}

	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("no"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := h.existingLocalPath(outsideFile); err == nil {
		t.Fatal("expected outside-root path rejection")
	}
}

func TestLocalRootRejectsSymlinkEscapeWhenSupported(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "secret.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	h := &host{}
	if _, err := h.authorizeRoot(root); err != nil {
		t.Fatal(err)
	}
	if _, err := h.existingLocalPath(filepath.Join(link, "secret.txt")); err == nil {
		t.Fatal("expected symlink escape rejection")
	}
}

func TestSafeLeafRejectsTraversalAndSeparators(t *testing.T) {
	for _, value := range []string{"", ".", "..", "../x", `a/b`, `a\\b`, "a\n"} {
		if _, err := safeLeaf(value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
	if got, err := safeLeaf("report.txt"); err != nil || got != "report.txt" {
		t.Fatalf("safe leaf rejected: %q %v", got, err)
	}
}

func TestOutboundResponseHasBoundedFallback(t *testing.T) {
	var out bytes.Buffer
	large := strings.Repeat("x", maxOutboundMessage)
	if err := writeResponse(&out, response{ID: "r1", OK: true, Result: large}); err != nil {
		t.Fatal(err)
	}
	payload, err := readFrame(&out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(payload, []byte("response_too_large")) {
		t.Fatalf("expected bounded response fallback: %s", payload)
	}
}
