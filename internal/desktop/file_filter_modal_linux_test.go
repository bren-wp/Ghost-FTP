//go:build linux

package desktop

import "testing"

func TestLinuxModalBackgroundInputIsolation(t *testing.T) {
	u := &linuxDesktop{
		focus:  linuxFieldHost,
		layout: buildLinuxDesktopLayout(premiumStartWidth, premiumStartHeight),
	}

	if u.layout.host == (linuxRect{}) || u.layout.localPath == (linuxRect{}) || u.layout.remotePath == (linuxRect{}) {
		t.Fatal("test requires ordinary editable field hit targets before modal isolation")
	}

	u.isolateLinuxModalBackgroundInput()

	if u.focus != linuxFieldCount {
		t.Fatalf("modal isolation focus = %d, want sentinel %d", u.focus, linuxFieldCount)
	}
	for name, rect := range map[string]linuxRect{
		"protocol": u.layout.protocol,
		"host": u.layout.host,
		"port": u.layout.port,
		"user": u.layout.user,
		"password": u.layout.password,
		"key": u.layout.key,
		"passphrase": u.layout.passphrase,
		"local path": u.layout.localPath,
		"remote path": u.layout.remotePath,
	} {
		if rect != (linuxRect{}) {
			t.Fatalf("%s hit target remained active during modal workflow: %+v", name, rect)
		}
	}
}
