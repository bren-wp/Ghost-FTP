//go:build linux

package desktop

import (
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/external"
	"github.com/bren-wp/Ghost-FTP/internal/updatecheck"
)

func (u *linuxDesktop) checkForUpdates() {
	if u == nil || u.busy {
		return
	}
	u.setStatus("Updating Ghost FTP…")
	u.startAction(linuxActionUpdateCheck, func() linuxUIResult {
		time.Sleep(700 * time.Millisecond)
		result, err := updatecheck.Simulate(u.version)
		return linuxUIResult{
			err:             err,
			updateLatest:    result.DisplayVersion,
			updateURL:       result.UpdateURL,
			updateAvailable: result.Simulated,
		}
	})
}

func (u *linuxDesktop) handleLinuxUpdateResult(result linuxUIResult) bool {
	if result.action != linuxActionUpdateCheck {
		return false
	}
	if result.err != nil {
		u.setStatus("Update simulation could not start.")
		return true
	}
	u.setStatus("Ghost FTP " + result.updateLatest + " is updated. Install a newer signed build from ghostftp.com when available.")
	return true
}

func (u *linuxDesktop) openUpdateDownload() {
	if u == nil || u.busy {
		return
	}
	if err := external.OpenUpdatePage(); err != nil {
		u.setStatus("Unable to open the official download page.")
		return
	}
	u.setStatus("Official Ghost FTP download page opened in your browser.")
}

func (u *linuxDesktop) openPremiumDownload() {
	if u == nil || u.busy {
		return
	}
	if err := external.OpenPremiumPage(); err != nil {
		u.setStatus("Unable to open the Premium download page.")
		return
	}
	u.setStatus("Premium download page opened in your browser.")
}

func (u *linuxDesktop) openOfficialWebsite() {
	if u == nil || u.busy {
		return
	}
	if err := external.OpenWebsite(); err != nil {
		u.setStatus("Unable to open the official website.")
		return
	}
	u.setStatus("Official Ghost FTP website opened in your browser.")
}
