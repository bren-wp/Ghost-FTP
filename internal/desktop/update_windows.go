//go:build windows

package desktop

import (
	"context"
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
	a.setStatus("Checking for updates…")
	a.goSafe(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		result, err := updatecheck.New().Check(ctx, a.version)
		a.dispatch(func() {
			if err != nil {
				a.setStatus("Update check failed.")
				platform.ErrorDialog(brand.ProductName, "Update check failed", "Ghost FTP could not securely check the public release channel. Try again later.")
				return
			}
			if !result.Available {
				a.setStatus("Ghost FTP is up to date.")
				platform.InfoDialog(brand.ProductName, "You're up to date", fmt.Sprintf("Ghost FTP %s is the current stable release.", a.version))
				return
			}
			a.setStatus("Update available: " + result.LatestVersion)
			if !platform.ConfirmDialog(
				brand.ProductName+" — Update",
				"Ghost FTP "+result.LatestVersion+" is available",
				"Open the verified GitHub Release page to download the update? Ghost FTP never sends server credentials, paths or transfer data during this check.",
			) {
				return
			}
			if err := external.OpenReleasePage(result.ReleaseURL); err != nil {
				platform.ErrorDialog(brand.ProductName, "Unable to open update", "Open the Ghost FTP releases page in your browser and download the signed package for this device.")
			}
		})
	})
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
