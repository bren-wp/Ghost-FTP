package desktop

import "testing"

func TestRemoteDesktopTarget(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"server.example.com", "server.example.com:3389"},
		{"server.example.com:3390", "server.example.com:3390"},
		{"192.0.2.10", "192.0.2.10:3389"},
		{"2001:db8::10", "[2001:db8::10]:3389"},
		{"[2001:db8::10]:3390", "[2001:db8::10]:3390"},
	}
	for _, tt := range tests {
		got, err := remoteDesktopTarget(tt.input)
		if err != nil {
			t.Fatalf("remoteDesktopTarget(%q): %v", tt.input, err)
		}
		if got != tt.want {
			t.Fatalf("remoteDesktopTarget(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRemoteDesktopTargetRejectsUnsafeInput(t *testing.T) {
	for _, input := range []string{"", "host 3389", "host\n/v:evil", "host:0", "host:65536", "user@host", "host/path", "[::1]:bad"} {
		if _, err := remoteDesktopTarget(input); err == nil {
			t.Fatalf("remoteDesktopTarget(%q) unexpectedly succeeded", input)
		}
	}
}
