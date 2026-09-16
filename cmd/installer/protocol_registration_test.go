package main

import "testing"

func TestBrowserProtocolCommandQuotesExecutableAndPayload(t *testing.T) {
	got, err := browserProtocolCommand(`C:\Users\Example\AppData\Local\Ghost FTP\GhostFTP.exe`)
	if err != nil {
		t.Fatalf("browserProtocolCommand returned error: %v", err)
	}
	want := `"C:\Users\Example\AppData\Local\Ghost FTP\GhostFTP.exe" "%1"`
	if got != want {
		t.Fatalf("browser protocol command = %q, want %q", got, want)
	}
}

func TestBrowserProtocolCommandRejectsUnsafeExecutablePaths(t *testing.T) {
	for _, value := range []string{"", "   ", `C:\Bad\"Path\GhostFTP.exe`, "C:\\Bad\nPath\\GhostFTP.exe", "C:\\Bad\rPath\\GhostFTP.exe", "C:\\Bad\x00Path\\GhostFTP.exe"} {
		if _, err := browserProtocolCommand(value); err == nil {
			t.Fatalf("expected unsafe path to be rejected: %q", value)
		}
	}
}

func TestBrowserProtocolRegistryLayoutUsesPerUserClasses(t *testing.T) {
	if browserProtocolKey != `Software\Classes\ghostftp` {
		t.Fatalf("unexpected protocol root: %q", browserProtocolKey)
	}
	if browserProtocolCommandKey != `Software\Classes\ghostftp\shell\open\command` {
		t.Fatalf("unexpected protocol command key: %q", browserProtocolCommandKey)
	}
}
