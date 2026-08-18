package security

import "testing"

func TestValidateHostRejectsMalformedIPv6Brackets(t *testing.T) {
	invalid := []string{
		"[2001:db8::1",
		"2001:db8::1]",
		"[[2001:db8::1]]",
		"[127.0.0.1]",
		"[]",
		"[not-an-ip]",
		"ftp.example.com:21",
	}
	for _, host := range invalid {
		if err := ValidateHost(host); err == nil {
			t.Fatalf("ValidateHost(%q) unexpectedly succeeded", host)
		}
	}
}

func TestValidateHostAcceptsCanonicalIPForms(t *testing.T) {
	valid := []string{
		"127.0.0.1",
		"2001:db8::1",
		"[2001:db8::1]",
	}
	for _, host := range valid {
		if err := ValidateHost(host); err != nil {
			t.Fatalf("ValidateHost(%q): %v", host, err)
		}
	}
}
