package security

import "testing"

func TestValidateHostAcceptsCanonicalIPv6Forms(t *testing.T) {
	for _, host := range []string{
		"2001:db8::1",
		"[2001:db8::1]",
		"::1",
	} {
		if err := ValidateHost(host); err != nil {
			t.Fatalf("ValidateHost(%q) = %v, want nil", host, err)
		}
	}
}

func TestValidateHostRejectsHostPortAndMalformedIPv6(t *testing.T) {
	for _, host := range []string{
		"example.test:21",
		"[2001:db8::1",
		"2001:db8::1]",
		"[[2001:db8::1]]",
	} {
		if err := ValidateHost(host); err == nil {
			t.Fatalf("ValidateHost(%q) unexpectedly succeeded", host)
		}
	}
}
