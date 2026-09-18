package external

import "testing"

func TestTrustedURLRequiresOfficialHTTPSHost(t *testing.T) {
	for _, tc := range []struct {
		value string
		ok    bool
	}{
		{"https://ghostftp.com/", true},
		{"https://ghostftp.com/#download", true},
		{"https://ghostftp.com/premium/", true},
		{"https://www.ghostftp.com/premium/", true},
		{"http://ghostftp.com/premium/", false},
		{"https://example.com/premium/", false},
		{"https://user:pass@ghostftp.com/premium/", false},
	} {
		_, err := trustedURL(tc.value, officialHosts)
		if tc.ok && err != nil {
			t.Fatalf("%q rejected: %v", tc.value, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("%q should be rejected", tc.value)
		}
	}
}
