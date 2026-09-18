//go:build linux

package desktop

import (
	"fmt"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/brand"
	"github.com/bren-wp/Ghost-FTP/internal/model"
)

const (
	linuxMasterRailX           = 14
	linuxMasterRailWidth       = 166
	linuxMasterContentLeft     = 204
	linuxMasterRailCardH       = 46
	linuxMasterRailCardGap     = 8
	linuxMasterRailUtilityH    = 38
	linuxMasterRailUtilityGap  = 7
	linuxMasterRailPrimaryTop  = 64
	linuxMasterRailBottomInset = 42
)

type linuxMasterRailLayout struct {
	brand       linuxRect
	files       linuxRect
	connections linuxRect
	transfers   linuxRect
	settings    linuxRect
	bookmarks   linuxRect
	diagnostics linuxRect
	about       linuxRect
	content     linuxRect
}

func buildLinuxMasterRailLayout(width, height int) linuxMasterRailLayout {
	if width < premiumMinWidth {
		width = premiumMinWidth
	}
	if height < premiumMinHeight {
		height = premiumMinHeight
	}

	layout := linuxMasterRailLayout{
		brand:   linuxRectWH(linuxMasterRailX, 14, linuxMasterRailWidth, 34),
		content: linuxRect{left: linuxMasterContentLeft, top: 0, right: width - premiumOuterGap, bottom: height},
	}

	y := linuxMasterRailPrimaryTop
	layout.files = linuxRectWH(linuxMasterRailX, y, linuxMasterRailWidth, linuxMasterRailCardH)
	y += linuxMasterRailCardH + linuxMasterRailCardGap
	layout.connections = linuxRectWH(linuxMasterRailX, y, linuxMasterRailWidth, linuxMasterRailCardH)
	y += linuxMasterRailCardH + linuxMasterRailCardGap
	layout.transfers = linuxRectWH(linuxMasterRailX, y, linuxMasterRailWidth, linuxMasterRailCardH)
	y += linuxMasterRailCardH + linuxMasterRailCardGap
	layout.settings = linuxRectWH(linuxMasterRailX, y, linuxMasterRailWidth, linuxMasterRailCardH)
	y += linuxMasterRailCardH

	utilityHeight := 3*linuxMasterRailUtilityH + 2*linuxMasterRailUtilityGap
	utilityY := height - linuxMasterRailBottomInset - utilityHeight
	if minimum := y + 18; utilityY < minimum {
		utilityY = minimum
	}
	layout.bookmarks = linuxRectWH(linuxMasterRailX, utilityY, linuxMasterRailWidth, linuxMasterRailUtilityH)
	utilityY += linuxMasterRailUtilityH + linuxMasterRailUtilityGap
	layout.diagnostics = linuxRectWH(linuxMasterRailX, utilityY, linuxMasterRailWidth, linuxMasterRailUtilityH)
	utilityY += linuxMasterRailUtilityH + linuxMasterRailUtilityGap
	layout.about = linuxRectWH(linuxMasterRailX, utilityY, linuxMasterRailWidth, linuxMasterRailUtilityH)
	return layout
}

func linuxMasterTransferBadge(jobs []model.TransferJob) int {
	return relevantTransferCount(jobs)
}

func linuxAffineRect(r linuxRect, oldLeft, oldRight, newLeft, newRight int) linuxRect {
	if oldRight <= oldLeft || newRight <= newLeft {
		return r
	}
	oldSpan := oldRight - oldLeft
	newSpan := newRight - newLeft
	mapX := func(value int) int {
		return newLeft + (value-oldLeft)*newSpan/oldSpan
	}
	left := mapX(r.left)
	right := mapX(r.right)
	if right <= left {
		right = left + 1
	}
	return linuxRect{left: left, top: r.top, right: right, bottom: r.bottom}
}

// applyLinuxMasterLayoutTransform keeps the proven X11 workspace behavior and
// remaps only its horizontal geometry into the content column beside the new
// application rail. Several established render hooks can reach this helper in
// one frame, so it must detect the already-remapped layout and never compound
// the affine transform.
func (u *linuxDesktop) applyLinuxMasterLayoutTransform() {
	if u == nil {
		return
	}
	if u.layout.localPath.left >= linuxMasterContentLeft && u.layout.remotePath.left >= linuxMasterContentLeft {
		// Settings is application navigation rather than workspace content. Keep
		// its hit target canonical even when a second hook reaches this helper.
		u.layout.settings = buildLinuxMasterRailLayout(u.width, u.height).settings
		return
	}
	oldLeft := premiumOuterGap
	oldRight := u.width - premiumOuterGap
	newLeft := linuxMasterContentLeft
	newRight := oldRight
	if newRight-newLeft < 520 {
		return
	}
	transform := func(r linuxRect) linuxRect {
		return linuxAffineRect(r, oldLeft, oldRight, newLeft, newRight)
	}

	for _, target := range []*linuxRect{
		&u.layout.protocol, &u.layout.host, &u.layout.port, &u.layout.user, &u.layout.password,
		&u.layout.key, &u.layout.passphrase, &u.layout.profile, &u.layout.saveProfile, &u.layout.removeProfile,
		&u.layout.connect, &u.layout.disconnect,
		&u.layout.localPath, &u.layout.localUp, &u.layout.localRefresh,
		&u.layout.remotePath, &u.layout.remoteUp, &u.layout.remoteRefresh,
		&u.layout.localNew, &u.layout.localRename, &u.layout.localDelete,
		&u.layout.remoteNew, &u.layout.remoteRename, &u.layout.remoteDelete, &u.layout.remoteChmod,
		&u.layout.localList, &u.layout.remoteList, &u.layout.upload, &u.layout.download,
		&u.layout.pause, &u.layout.resume, &u.layout.cancelJob, &u.layout.retryJob, &u.layout.clearQueue,
		&u.layout.queue,
	} {
		*target = transform(*target)
	}

	// Settings is application navigation rather than workspace content in the
	// master design. Keeping its real hit target in the rail also prevents a
	// duplicate top-right Settings button from surviving the transition.
	u.layout.settings = buildLinuxMasterRailLayout(u.width, u.height).settings
}

func (u *linuxDesktop) renderLinuxBrandMark(r linuxRect) error {
	if u == nil || u.x == nil {
		return nil
	}
	x, y := r.left+2, r.top+1
	dark := RGB{R: 11, G: 15, B: 23}
	gold := RGB{R: 246, G: 196, B: 69}

	// Compact application mark from the supplied Ghost FTP references:
	// dark square, gold frame and a simple gold ghost with two dark eyes.
	if err := u.x.fillRect(x, y, 32, 32, gold); err != nil {
		return err
	}
	if err := u.x.fillRect(x+2, y+2, 28, 28, dark); err != nil {
		return err
	}
	if err := u.x.fillRect(x+9, y+9, 14, 14, gold); err != nil {
		return err
	}
	if err := u.x.fillRect(x+8, y+12, 16, 13, gold); err != nil {
		return err
	}
	if err := u.x.fillRect(x+10, y+23, 4, 3, gold); err != nil {
		return err
	}
	if err := u.x.fillRect(x+18, y+23, 4, 3, gold); err != nil {
		return err
	}
	if err := u.x.fillRect(x+12, y+14, 2, 2, dark); err != nil {
		return err
	}
	return u.x.fillRect(x+18, y+14, 2, 2, dark)
}

func linuxTransferBadgeLabel(count int) string {
	if count <= 0 {
		return ""
	}
	if count > 99 {
		return "99+"
	}
	return fmt.Sprintf("%d", count)
}

func (u *linuxDesktop) renderLinuxMasterRail() error {
	if u == nil || u.x == nil {
		return nil
	}
	rail := buildLinuxMasterRailLayout(u.width, u.height)

	// Cover the baseline panel that was rendered before the rail hook. This is
	// deliberate: the X11 renderer has no child windows, so the rail is a real
	// late paint layer over the reserved column, while all actionable workspace
	// rectangles have already been transformed out of that column.
	if err := u.x.fillRect(0, 0, linuxMasterContentLeft, u.height, premiumTheme.Panel); err != nil {
		return err
	}
	if err := u.x.strokeRect(linuxMasterContentLeft-1, 0, 1, u.height, premiumTheme.Border); err != nil {
		return err
	}

	if err := u.renderLinuxBrandMark(rail.brand); err != nil {
		return err
	}
	if err := u.x.text(rail.brand.left+42, rail.brand.top+22, strings.ToUpper(brand.ProductName), premiumTheme.Text, premiumTheme.Panel); err != nil {
		return err
	}
	labels := navigationLabelsForLanguage(u.language)
	if err := u.drawButton(rail.files, labels.Files, !u.busy, true); err != nil {
		return err
	}
	if err := u.drawButton(rail.connections, labels.Connections, !u.busy, false); err != nil {
		return err
	}
	transferLabel := labels.TransferQueue
	if badge := linuxTransferBadgeLabel(linuxMasterTransferBadge(u.transferJobs)); badge != "" {
		transferLabel += " · " + badge
	}
	if err := u.drawButton(rail.transfers, transferLabel, !u.busy, false); err != nil {
		return err
	}
	if err := u.drawButton(rail.settings, u.tr("common.settings"), !u.busy, false); err != nil {
		return err
	}
	if err := u.drawButton(rail.bookmarks, bookmarkWordsForLanguage(u.language).Title, !u.busy, false); err != nil {
		return err
	}
	if err := u.drawButton(rail.diagnostics, labels.Diagnostics, !u.busy, false); err != nil {
		return err
	}
	if err := u.drawButton(rail.about, u.tr("common.about"), !u.busy, false); err != nil {
		return err
	}

	// Replace the obsolete wide branding subtitle with a compact content title
	// and truthful live connection state. The old header was already painted by
	// renderHeader, so clearing this strip prevents stale subtitle fragments.
	contentHeader := linuxRect{left: linuxMasterContentLeft, top: 0, right: u.width, bottom: 72}
	if err := u.x.fillRect(contentHeader.left, contentHeader.top, contentHeader.right-contentHeader.left, contentHeader.bottom-contentHeader.top, premiumTheme.Window); err != nil {
		return err
	}
	if err := u.x.text(linuxMasterContentLeft+12, 31, strings.ToUpper(labels.Files), premiumTheme.Text, premiumTheme.Window); err != nil {
		return err
	}
	badge := strings.ToUpper(u.tr("badge.disconnected"))
	badgeColor := premiumTheme.Muted
	if u.connected {
		badge = strings.ToUpper(u.tr("badge.connected"))
		badgeColor = premiumTheme.Success
	} else if u.busy {
		badge = strings.ToUpper(linuxTrimForUI(u.tr("connection.connecting", u.host), 24))
		badgeColor = premiumTheme.Warn
	}
	return u.x.text(max(linuxMasterContentLeft+180, u.width-300), 31, badge+"  "+u.version, badgeColor, premiumTheme.Window)
}

func (u *linuxDesktop) handleLinuxMasterRailMouse(x, y int) bool {
	if u == nil {
		return false
	}
	rail := buildLinuxMasterRailLayout(u.width, u.height)
	switch {
	case rail.files.contains(x, y):
		u.focus = linuxFieldLocalPath
		u.setStatus(u.tr("section.local"))
	case rail.connections.contains(x, y):
		u.focus = linuxFieldHost
		u.setStatus(navigationLabelsForLanguage(u.language).Connections)
	case rail.transfers.contains(x, y):
		if len(u.transferJobs) > 0 && u.selectedTransfer < 0 {
			u.selectedTransfer = 0
		}
		u.setStatus(u.tr("section.transfers"))
	case rail.settings.contains(x, y):
		if !u.busy {
			u.openSettings()
		}
	case rail.bookmarks.contains(x, y):
		if !u.busy {
			u.openLinuxBookmarks("")
		}
	case rail.diagnostics.contains(x, y):
		if !u.busy {
			u.openLinuxInfoOverlay(linuxInfoOverlayConnection)
		}
	case rail.about.contains(x, y):
		if !u.busy {
			u.openLinuxInfoOverlay(linuxInfoOverlayAbout)
		}
	default:
		return false
	}
	return true
}
