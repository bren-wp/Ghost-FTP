package main

import "testing"

func TestNormalizedRole(t *testing.T) {
	for _, value := range []string{"setup", "portable", " Setup ", "PORTABLE"} {
		if _, err := normalizedRole(value); err != nil {
			t.Fatalf("role %q rejected: %v", value, err)
		}
	}
	if _, err := normalizedRole("other"); err == nil {
		t.Fatal("expected unsupported role to be rejected")
	}
}

func TestPayloadPathAcceptsOnlySupportedArchitectures(t *testing.T) {
	for arch, want := range map[string]string{
		"x86": "payload/x86/GhostFTP.exe",
		"x64": "payload/x64/GhostFTP.exe",
	} {
		got, err := payloadPath(arch)
		if err != nil {
			t.Fatalf("arch %q rejected: %v", arch, err)
		}
		if got != want {
			t.Fatalf("arch %q path=%q want=%q", arch, got, want)
		}
	}
	if _, err := payloadPath("arm64"); err == nil {
		t.Fatal("expected unsupported architecture to be rejected")
	}
}
