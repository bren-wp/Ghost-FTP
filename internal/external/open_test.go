package external

import "testing"

func TestTrustedURLRequiresHTTPSAndAllowedHost(t *testing.T) {
	for _, tc := range []struct {
		value string
		ok    bool
	}{
		{"https://ghostftp.com/premium/", true},
		{"https://www.ghostftp.com/premium/", true},
		{"http://ghostftp.com/premium/", false},
		{"https://evil.example/premium/", false},
		{"https://user:pass@ghostftp.com/premium/", false},
	} {
		_, err := trustedURL(tc.value, []string{"ghostftp.com", "www.ghostftp.com"})
		if tc.ok && err != nil {
			t.Fatalf("%q rejected: %v", tc.value, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("%q should be rejected", tc.value)
		}
	}
}

func TestReleaseURLTrustBoundary(t *testing.T) {
	if _, err := trustedURL("https://github.com/bren-wp/Ghost-FTP/releases/tag/ghostftp-v0.0.8", []string{"github.com"}); err != nil {
		t.Fatal(err)
	}
	if _, err := trustedURL("https://github.example/bren-wp/Ghost-FTP/releases/tag/ghostftp-v0.0.8", []string{"github.com"}); err == nil {
		t.Fatal("unexpected trusted lookalike host")
	}
}
