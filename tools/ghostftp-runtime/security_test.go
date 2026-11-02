package main

import (
    "net/http"
    "net/http/httptest"
    "testing"
    "strings"
    "os"
    "path/filepath"
    "runtime"
)

func TestLocalHostGuardBlocksRebindingBeforeSessionToken(t *testing.T) {
    cases := []struct {
        host string
        origin string
        fetchSite string
        accept bool
    }{
        {"127.0.0.1:12345", "", "", true},
        {"127.0.0.1:12345", "http://127.0.0.1:12345", "same-origin", true},
        {"localhost:12345", "http://localhost:12345", "same-origin", true},
        {"[::1]:12345", "", "", true},
        {"evil.example:12345", "", "", false},
        {"127.0.0.1.evil.example:12345", "", "", false},
        {"127.0.0.1:12345", "http://evil.example", "", false},
        {"127.0.0.1:12345", "", "cross-site", false},
        {"127.0.0.1", "", "", false},
        {"0.0.0.0:12345", "", "", false},
    }
    for _, tc := range cases {
        req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:12345/api/session", nil)
        req.Host = tc.host
        req.Header.Set("Origin", tc.origin)
        req.Header.Set("Sec-Fetch-Site", tc.fetchSite)
        record := httptest.NewRecorder()
        ok := allowLocalHost(record, req)
        if ok != tc.accept {
            t.Errorf("Host %q Origin %q FetchSite %q: want accept %v got %v", tc.host, tc.origin, tc.fetchSite, tc.accept, ok)
        }
        if !tc.accept && record.Code != http.StatusForbidden {
            t.Errorf("Host %q: expected forbidden, got %d", tc.host, record.Code)
        }
    }
}

// A JSON mutation request must contain exactly one object; the previous
// decoder silently ignored trailing JSON values and arbitrary bytes.
func TestDecodeJSONRejectsTrailingPayloads(t *testing.T) {
    cases := []struct{ body string; accept bool }{
        {`{"action":"open"}`, true},
        {`{"action":"open"}  \n\t`, true},
        {`{"action":"open"}{"action":"delete"}`, false},
        {`{"action":"open"} true`, false},
        {`{"unknown":"x"}`, false},
        {`{"action":"open"} trailing-bytes`, false},
        {"", false},
    }
    for _, tc := range cases {
        req := httptest.NewRequest(http.MethodPost, "/api/window", strings.NewReader(tc.body))
        rec := httptest.NewRecorder()
        var payload struct{ Action string `json:"action"` }
        accepted := decodeJSON(rec, req, &payload)
        if accepted != tc.accept { t.Errorf("%q: accepted=%v, want %v", tc.body, accepted, tc.accept) }
        if !tc.accept && rec.Code != http.StatusBadRequest { t.Errorf("%q: response status %d", tc.body, rec.Code) }
    }
}

// A selected path or its parent must not redirect destructive operations
// through a filesystem symlink. Recursive preflight must not chmod ANY
// entry when an unsafe link is discovered.
func TestChmodRejectsSymlinksWithoutChangingExternalPermissions(t *testing.T) {
    if runtime.GOOS == "windows" { t.Skip("creating symlinks requires elevated Windows privileges") }
    root := t.TempDir()
    outside := filepath.Join(root, "outside")
    safe := filepath.Join(root, "safe")
    if err := os.MkdirAll(outside, 0700); err != nil { t.Fatal(err) }
    if err := os.MkdirAll(safe, 0700); err != nil { t.Fatal(err) }
    target := filepath.Join(outside, "secret.txt")
    if err := os.WriteFile(target, []byte("private"), 0600); err != nil { t.Fatal(err) }
    original, err := os.Stat(target); if err != nil { t.Fatal(err) }
    linkedFile := filepath.Join(safe, "secret-link")
    if err := os.Symlink(target, linkedFile); err != nil { t.Skip(err) }
    if err := chmodWithoutLinks(linkedFile, 0777, false); err == nil {
        t.Fatal("direct chmod followed file symlink")
    }
    if err := chmodWithoutLinks(safe, 0777, true); err == nil {
        t.Fatal("recursive chmod followed file symlink")
    }
    after, err := os.Stat(target); if err != nil { t.Fatal(err) }
    if after.Mode().Perm() != original.Mode().Perm() {
        t.Fatal("external target permission changed")
    }
    linkedDir := filepath.Join(root, "linked-dir")
    if err := os.Symlink(outside, linkedDir); err != nil { t.Fatal(err) }
    if !dangerousDestructivePath(filepath.Join(linkedDir, "secret.txt")) {
        t.Fatal("destructive operation accepted symlink parent")
    }
}
