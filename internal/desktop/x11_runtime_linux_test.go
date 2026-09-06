//go:build linux

package desktop

import (
	"os"
	"testing"
)

// TestX11RuntimeSmoke is opt-in because normal CI runners may be headless. The
// authentic Linux UI workflow enables it under an isolated local Xvfb server.
// Keeping the test in-package exposes exact X11 protocol failures without
// weakening the production executable's user-facing error sanitization.
func TestX11RuntimeSmoke(t *testing.T) {
	if os.Getenv("GHOSTFTP_X11_SMOKE") != "1" {
		t.Skip("set GHOSTFTP_X11_SMOKE=1 under a local X11 server")
	}
	x, err := connectLocalX11()
	if err != nil {
		t.Fatalf("X11 handshake failed: %v", err)
	}
	defer x.close()
	if err := x.createWindow(premiumStartWidth, premiumStartHeight, "Ghost FTP X11 runtime smoke"); err != nil {
		t.Fatalf("X11 CreateWindow failed: %v", err)
	}
	if err := x.fillRect(0, 0, premiumStartWidth, premiumStartHeight, premiumTheme.Window); err != nil {
		t.Fatalf("X11 render request failed: %v", err)
	}
	if err := x.text(20, 30, "Ghost FTP runtime smoke", premiumTheme.Text, premiumTheme.Window); err != nil {
		t.Fatalf("X11 text request failed: %v", err)
	}
}
