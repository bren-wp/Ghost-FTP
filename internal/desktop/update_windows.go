//go:build windows

package desktop

import (
	"fmt"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/brand"
	"github.com/bren-wp/Ghost-FTP/internal/external"
	"github.com/bren-wp/Ghost-FTP/internal/platform"
	"github.com/bren-wp/Ghost-FTP/internal/updatecheck"
)

func (a *app) checkForUpdates() {
	if a == nil || a.connectionBusy || a.profileMutationBusy {
		return
	}
	a.setStatus("Updating Ghost FTP…")
	a.goSafe(func() {
		time.Sleep(700 * time.Millisecond)
		result, err := updatecheck.Simulate(a.version)
		a.dispatch(func() {
			if err != nil {
				a.setStatus("Update simulation could not start.")
				return
			}
			a.setStatus("Ghost FTP " + result.DisplayVersion + " is updated.")
			platform.InfoDialog(
				brand.ProductName,
				"Update complete",
				fmt.Sprintf("Ghost FTP %s update simulation completed. The installed signed build remains version %s until you install a newer package from ghostftp.com.", result.DisplayVersion, result.CurrentVersion),
			)
		})
	})
}

func (a *app) openUpdateDownload() {
	if a == nil {
		return
	}
	if err := external.OpenUpdatePage(); err != nil {
		platform.ErrorDialog(brand.ProductName, "Unable to open updates", "Open "+brand.UpdateURL+" in your browser.")
		return
	}
	a.setStatus("Official Ghost FTP download page opened in your browser.")
}

func (a *app) openPremiumDownload() {
	if a == nil {
		return
	}
	if err := external.OpenPremiumPage(); err != nil {
		platform.ErrorDialog(brand.ProductName, "Unable to open Premium", "Open "+brand.PremiumURL+" in your browser.")
		return
	}
	a.setStatus("Premium download page opened in your browser.")
}

func (a *app) openOfficialWebsite() {
	if a == nil {
		return
	}
	if err := external.OpenWebsite(); err != nil {
		platform.ErrorDialog(brand.ProductName, "Unable to open website", "Open "+brand.WebsiteURL+" in your browser.")
		return
	}
	a.setStatus("Official Ghost FTP website opened in your browser.")
}
