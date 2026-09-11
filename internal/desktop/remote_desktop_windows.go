//go:build windows

package desktop

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

func (a *app) openRemoteDesktop() {
	initial := strings.TrimSpace(getText(a.host))
	if initial == "" {
		initial = "server.example.com:3389"
	}
	value, ok := platform.PromptDialogWithLabels(
		"Ghost FTP — Remote Desktop",
		"RDP server (host[:port]). Ghost FTP never stores or forwards the RDP password.",
		initial,
		"Open RDP",
		a.tr("common.cancel"),
	)
	if !ok {
		return
	}
	target, err := remoteDesktopTarget(value)
	if err != nil {
		a.setStatus(fmt.Sprintf("Remote Desktop: %v", err))
		return
	}
	cmd := exec.Command("mstsc.exe", "/v:"+target)
	if err := cmd.Start(); err != nil {
		a.setStatus("Remote Desktop could not be opened. Windows Remote Desktop (mstsc.exe) is required.")
		return
	}
	a.setStatus("Remote Desktop opened in Windows. Authentication remains inside the RDP client.")
}
