//go:build linux

package desktop

import (
	"fmt"
	"os/exec"
	"strings"
)

const linuxPromptRemoteDesktop = 1001

func (u *linuxDesktop) remoteDesktopButtonRect() linuxRect {
	settings := u.layout.settings
	return linuxRectWH(settings.left-90, settings.top, 82, settings.bottom-settings.top)
}

func (u *linuxDesktop) openRemoteDesktopPrompt() {
	if u == nil || u.busy {
		return
	}
	initial := strings.TrimSpace(u.host)
	if initial == "" {
		initial = "server.example.com:3389"
	}
	u.openPrompt(linuxPromptRemoteDesktop, "Remote Desktop · host[:port]", initial)
}

func (u *linuxDesktop) submitRemoteDesktop(value string) {
	target, err := remoteDesktopTarget(value)
	if err != nil {
		u.setStatus(fmt.Sprintf("Remote Desktop: %v", err))
		return
	}
	client, err := exec.LookPath("xfreerdp3")
	if err != nil {
		client, err = exec.LookPath("xfreerdp")
	}
	if err != nil {
		u.setStatus("Remote Desktop requires FreeRDP (xfreerdp3 or xfreerdp).")
		return
	}
	cmd := exec.Command(client, "/v:"+target)
	if err := cmd.Start(); err != nil {
		u.setStatus("Remote Desktop could not be opened: " + err.Error())
		return
	}
	u.setStatus("Remote Desktop opened in FreeRDP. Authentication remains inside the RDP client.")
}
