package main

import (
    "net/http"
    "net/http/httptest"
    "testing"
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
