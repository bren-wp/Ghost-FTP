//go:build linux

package desktop

import "testing"

func TestLinuxModalBackgroundInputIsolation(t *testing.T) {
	u := &linuxDesktop{}
	u.focus = linuxFieldHost
	u.layout = buildLinuxDesktopLayout(premiumStartWidth, premiumStartHeight)

	if u.layout.host == (linuxRect{}) || u.layout.localPath == (linuxRect{}) || u.layout.remotePath == (linuxRect{}) {
		t.Fatal("test requires ordinary editable field hit targets before modal isolation")
	}

	u.isolateLinuxModalBackgroundInput()

	if u.focus != linuxFieldCount {
		t.Fatalf("modal isolation focus = %d, want sentinel %d", u.focus, linuxFieldCount)
	}
	rects := []linuxRect{
		u.layout.protocol,
		u.layout.host,
		u.layout.port,
		u.layout.user,
		u.layout.password,
		u.layout.key,
		u.layout.passphrase,
		u.layout.localPath,
		u.layout.remotePath,
	}
	for _, rect := range rects {
		if rect != (linuxRect{}) {
			t.Fatalf("editable hit target remained active during modal workflow: %+v", rect)
		}
	}

	u.restoreLinuxModalBackgroundInput()
	if u.focus != linuxFieldHost {
		t.Fatalf("restored focus = %d, want previous field %d", u.focus, linuxFieldHost)
	}
}

func TestLinuxModalBackgroundInputRestoreDoesNotClobberNewFocus(t *testing.T) {
	u := &linuxDesktop{}
	u.focus = linuxFieldHost

	u.isolateLinuxModalBackgroundInput()
	u.focus = linuxFieldPort
	u.restoreLinuxModalBackgroundInput()

	if u.focus != linuxFieldPort {
		t.Fatalf("restore clobbered newer focus: got %d, want %d", u.focus, linuxFieldPort)
	}
}
