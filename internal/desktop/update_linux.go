//go:build linux

package desktop

import (
	"context"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/external"
	"github.com/bren-wp/Ghost-FTP/internal/updatecheck"
)

func (u *linuxDesktop) checkForUpdates() {
	if u == nil || u.busy {
		return
	}
	if u.pendingUpdateURL != "" {
		if err := external.OpenReleasePage(u.pendingUpdateURL); err != nil {
			u.setStatus("Unable to open the verified release page.")
			return
		}
		u.setStatus("Update download page opened in your browser.")
		return
	}

	u.setStatus("Checking for updates…")
	u.startAction(linuxActionUpdateCheck, func() linuxUIResult {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		result, err := updatecheck.New().Check(ctx, u.version)
		return linuxUIResult{
			err:             err,
			updateLatest:    result.LatestVersion,
			updateURL:       result.ReleaseURL,
			updateAvailable: result.Available,
		}
	})
}

func (u *linuxDesktop) handleLinuxUpdateResult(result linuxUIResult) bool {
	if result.action != linuxActionUpdateCheck {
		return false
	}
	u.pendingUpdateURL = ""
	u.pendingUpdateVersion = ""
	if result.err != nil {
		u.setStatus("Update check failed. Try again later.")
		return true
	}
	if !result.updateAvailable {
		u.setStatus("Ghost FTP " + u.version + " is the current stable release.")
		return true
	}
	u.pendingUpdateURL = result.updateURL
	u.pendingUpdateVersion = result.updateLatest
	u.setStatus("Ghost FTP " + result.updateLatest + " is available. Click Check for updates again to open the verified release page.")
	return true
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
