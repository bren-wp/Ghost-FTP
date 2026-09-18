//go:build linux

package desktop

import (
	"fmt"
	"path/filepath"
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
	linuxWorkspaceHistoryLimit = 64
)

type linuxMasterRailLayout struct {
	brand       linuxRect
	files       linuxRect
	connections linuxRect
	transfers   linuxRect
	settings    linuxRect
	promo       linuxRect
	version     linuxRect
	content     linuxRect
}

type linuxWorkspaceHistoryEntry struct {
	Remote bool
	Path   string
}

type linuxMasterToolbarLayout struct {
	region     linuxRect
	connection linuxRect
	status     linuxRect
	quick      linuxRect
	back       linuxRect
	forward    linuxRect
	refresh    linuxRect
	newFolder  linuxRect
	upload     linuxRect
	download   linuxRect
	bookmarks  linuxRect
	more       linuxRect
}

func buildLinuxMasterToolbarLayout(width int) linuxMasterToolbarLayout {
	if width < premiumMinWidth {
		width = premiumMinWidth
	}
	left := linuxMasterContentLeft + 10
	right := width - premiumOuterGap
	contentWidth := right - left
	gap := 7
	rowH := 34
	badgeW := 138
	quickW := 132
	connectionW := contentWidth - badgeW - quickW - 2*gap
	if connectionW < 260 {
		connectionW = 260
	}
	row1Y := 78
	layout := linuxMasterToolbarLayout{
		region:     linuxRect{left: linuxMasterContentLeft, top: 72, right: width, bottom: 171},
		connection: linuxRectWH(left, row1Y, connectionW, rowH),
		status:     linuxRectWH(left+connectionW+gap, row1Y, badgeW, rowH),
		quick:      linuxRectWH(left+connectionW+gap+badgeW+gap, row1Y, quickW, rowH),
	}
	row2Y := 124
	toolbarWidth := contentWidth
	buttonW := (toolbarWidth - 7*gap) / 8
	if buttonW < 74 {
		buttonW = 74
	}
	x := left
	for _, target := range []*linuxRect{
		&layout.back, &layout.forward, &layout.refresh, &layout.newFolder,
		&layout.upload, &layout.download, &layout.bookmarks, &layout.more,
	} {
		*target = linuxRectWH(x, row2Y, buttonW, 38)
		x += buttonW + gap
	}
	return layout
}

func normalizeLinuxWorkspaceHistoryEntry(entry linuxWorkspaceHistoryEntry) linuxWorkspaceHistoryEntry {
	entry.Path = strings.TrimSpace(entry.Path)
	if entry.Remote {
		if entry.Path == "" {
			entry.Path = "/"
		} else {
			entry.Path = cleanRemote(entry.Path)
		}
	} else if entry.Path != "" {
		entry.Path = filepath.Clean(entry.Path)
	}
	return entry
}

func sameLinuxWorkspaceHistoryEntry(left, right linuxWorkspaceHistoryEntry) bool {
	left = normalizeLinuxWorkspaceHistoryEntry(left)
	right = normalizeLinuxWorkspaceHistoryEntry(right)
	return left.Remote == right.Remote && left.Path == right.Path
}

func appendLinuxWorkspaceHistory(history []linuxWorkspaceHistoryEntry, entry linuxWorkspaceHistoryEntry) []linuxWorkspaceHistoryEntry {
	entry = normalizeLinuxWorkspaceHistoryEntry(entry)
	if entry.Path == "" {
		return history
	}
	if len(history) > 0 && sameLinuxWorkspaceHistoryEntry(history[len(history)-1], entry) {
		return history
	}
	history = append(history, entry)
	if len(history) > linuxWorkspaceHistoryLimit {
		history = append([]linuxWorkspaceHistoryEntry(nil), history[len(history)-linuxWorkspaceHistoryLimit:]...)
	}
	return history
}

func (u *linuxDesktop) linuxHistoryCan(back bool) bool {
	if u == nil || u.busy {
		return false
	}
	history := u.workspaceForwardHistory
	if back {
		history = u.workspaceBackHistory
	}
	if len(history) == 0 {
		return false
	}
	target := history[len(history)-1]
	return !target.Remote || u.connected
}

func (u *linuxDesktop) completeLinuxWorkspaceNavigation(remote bool, next string) {
	if u == nil {
		return
	}
	currentPath := u.localCurrent
	if remote {
		currentPath = u.remoteCurrent
	}
	current := normalizeLinuxWorkspaceHistoryEntry(linuxWorkspaceHistoryEntry{Remote: remote, Path: currentPath})
	nextEntry := normalizeLinuxWorkspaceHistoryEntry(linuxWorkspaceHistoryEntry{Remote: remote, Path: next})
	if sameLinuxWorkspaceHistoryEntry(current, nextEntry) {
		if u.workspaceReplayActive && sameLinuxWorkspaceHistoryEntry(u.workspaceReplayTarget, nextEntry) {
			u.workspaceReplayActive = false
		}
		return
	}

	if u.workspaceReplayActive && sameLinuxWorkspaceHistoryEntry(u.workspaceReplayTarget, nextEntry) {
		source := &u.workspaceForwardHistory
		destination := &u.workspaceBackHistory
		if u.workspaceReplayBack {
			source = &u.workspaceBackHistory
			destination = &u.workspaceForwardHistory
		}
		if len(*source) > 0 && sameLinuxWorkspaceHistoryEntry((*source)[len(*source)-1], nextEntry) {
			*source = (*source)[:len(*source)-1]
		}
		*destination = appendLinuxWorkspaceHistory(*destination, current)
		u.workspaceReplayActive = false
		u.lastFilePaneRemote = remote
		return
	}

	u.workspaceBackHistory = appendLinuxWorkspaceHistory(u.workspaceBackHistory, current)
	u.workspaceForwardHistory = nil
	u.lastFilePaneRemote = remote
}

func (u *linuxDesktop) failLinuxWorkspaceReplay() {
	if u == nil {
		return
	}
	u.workspaceReplayActive = false
}

func (u *linuxDesktop) navigateLinuxWorkspaceHistory(back bool) {
	if u == nil || !u.linuxHistoryCan(back) {
		return
	}
	history := u.workspaceForwardHistory
	if back {
		history = u.workspaceBackHistory
	}
	target := history[len(history)-1]
	u.workspaceReplayActive = true
	u.workspaceReplayBack = back
	u.workspaceReplayTarget = target
	if target.Remote {
		u.refreshRemote(target.Path)
		return
	}
	u.refreshLocal(target.Path)
}

func (u *linuxDesktop) linuxConnectionSummary() string {
	if u.profileIndex >= 0 && u.profileIndex < len(u.profiles) {
		if name := strings.TrimSpace(u.profiles[u.profileIndex].Name); name != "" {
			return name
		}
	}
	host := strings.TrimSpace(u.host)
	if host == "" {
		return "Open Connections to configure a server"
	}
	user := strings.TrimSpace(u.username)
	prefix := strings.ToUpper(strings.TrimSpace(u.protocol)) + "://"
	if user != "" {
		return prefix + user + "@" + host
	}
	return prefix + host
}

func (u *linuxDesktop) renderLinuxMasterToolbar() error {
	if u == nil || u.x == nil || u.masterConnectionsVisible {
		return nil
	}
	layout := buildLinuxMasterToolbarLayout(u.width)
	if err := u.x.fillRect(layout.region.left, layout.region.top, layout.region.right-layout.region.left, layout.region.bottom-layout.region.top, premiumTheme.Window); err != nil {
		return err
	}
	if err := u.drawButton(layout.connection, linuxTrimForUI(u.linuxConnectionSummary(), 68), !u.busy, false); err != nil {
		return err
	}
	statusLabel := u.tr("badge.disconnected")
	statusColor := premiumTheme.Muted
	if u.connected {
		statusLabel = u.tr("badge.connected")
		statusColor = premiumTheme.Success
	}
	if err := u.x.fillRect(layout.status.left, layout.status.top, layout.status.right-layout.status.left, layout.status.bottom-layout.status.top, premiumTheme.List); err != nil {
		return err
	}
	if err := u.x.strokeRect(layout.status.left, layout.status.top, layout.status.right-layout.status.left, layout.status.bottom-layout.status.top, premiumTheme.Border); err != nil {
		return err
	}
	if err := u.x.text(layout.status.left+10, layout.status.top+22, strings.ToUpper(linuxTrimForUI(statusLabel, 18)), statusColor, premiumTheme.List); err != nil {
		return err
	}
	quickLabel := "Quick Connect"
	if u.connected {
		quickLabel = u.tr("common.disconnect")
	}
	if err := u.drawButton(layout.quick, quickLabel, !u.busy, true); err != nil {
		return err
	}

	toolbar := []struct {
		rect    linuxRect
		label   string
		enabled bool
		accent  bool
	}{
		{layout.back, "Back", u.linuxHistoryCan(true), false},
		{layout.forward, "Forward", u.linuxHistoryCan(false), false},
		{layout.refresh, u.tr("common.refresh"), !u.busy, false},
		{layout.newFolder, u.tr("common.new_folder"), !u.busy && (!u.lastFilePaneRemote || u.connected), false},
		{layout.upload, u.tr("transfer.upload"), u.connected && u.selectedLocal >= 0 && !u.busy, true},
		{layout.download, u.tr("transfer.download"), u.connected && u.selectedRemote >= 0 && !u.busy, false},
		{layout.bookmarks, bookmarkWordsForLanguage(u.language).Title, !u.busy, false},
		{layout.more, "More", !u.busy, false},
	}
	for _, item := range toolbar {
		if err := u.drawButtonWithLimit(item.rect, item.label, item.enabled, item.accent, 14); err != nil {
			return err
		}
	}
	return nil
}

func (u *linuxDesktop) handleLinuxMasterToolbarMouse(x, y int) bool {
	if u == nil || u.masterConnectionsVisible {
		return false
	}
	layout := buildLinuxMasterToolbarLayout(u.width)
	if !layout.region.contains(x, y) {
		return false
	}
	switch {
	case layout.connection.contains(x, y):
		if len(u.profiles) > 0 && !u.busy && !u.connected {
			u.cycleProfile()
		} else {
			u.masterConnectionsVisible = true
			u.focus = linuxFieldHost
			u.setStatus(navigationLabelsForLanguage(u.language).Connections)
		}
	case layout.quick.contains(x, y):
		if u.connected {
			u.disconnect()
		} else if strings.TrimSpace(u.host) == "" {
			u.masterConnectionsVisible = true
			u.focus = linuxFieldHost
			u.setStatus("Enter connection details.")
		} else {
			u.connectToServer("")
		}
	case layout.back.contains(x, y):
		u.navigateLinuxWorkspaceHistory(true)
	case layout.forward.contains(x, y):
		u.navigateLinuxWorkspaceHistory(false)
	case layout.refresh.contains(x, y):
		if u.lastFilePaneRemote && u.connected {
			u.refreshRemote(u.remoteCurrent)
		} else {
			u.refreshLocal(u.localCurrent)
		}
	case layout.newFolder.contains(x, y):
		if u.lastFilePaneRemote && u.connected {
			u.openPrompt(linuxPromptRemoteMkdir, u.tr("common.new_folder")+" · "+u.tr("section.remote"), u.tr("common.new_folder"))
		} else {
			u.openPrompt(linuxPromptLocalMkdir, u.tr("common.new_folder")+" · "+u.tr("section.local"), u.tr("common.new_folder"))
		}
	case layout.upload.contains(x, y):
		u.queueTransfer("upload")
	case layout.download.contains(x, y):
		u.queueTransfer("download")
	case layout.bookmarks.contains(x, y):
		u.openLinuxBookmarks("")
	case layout.more.contains(x, y):
		u.openLinuxInfoOverlay(linuxInfoOverlayConnection)
	}
	return true
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

	// Match the supplied desktop master: the rail has only the four primary
	// destinations. Bookmarks lives in the Files toolbar; Connection info and
	// About live in More. The lower rail is reserved for the product promise
	// and version instead of duplicate navigation.
	promoH := 112
	versionH := 20
	versionTop := height - linuxMasterRailBottomInset - versionH
	promoTop := versionTop - 12 - promoH
	if minimum := y + 18; promoTop < minimum {
		promoTop = minimum
	}
	layout.promo = linuxRectWH(linuxMasterRailX, promoTop, linuxMasterRailWidth, promoH)
	layout.version = linuxRectWH(linuxMasterRailX, versionTop, linuxMasterRailWidth, versionH)
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

	if err := u.drawPanel(rail.promo); err != nil {
		return err
	}
	promoX := rail.promo.left + 14
	promoY := rail.promo.top + 25
	if err := u.x.text(promoX, promoY, brand.ProductName, premiumTheme.Text, premiumTheme.Panel); err != nil {
		return err
	}
	for _, line := range []string{"One client.", "Five platforms.", "Zero friction."} {
		promoY += 22
		if err := u.x.text(promoX, promoY, line, premiumTheme.Muted, premiumTheme.Panel); err != nil {
			return err
		}
	}
	if err := u.x.text(rail.version.left+2, rail.version.top+15, "v"+u.version+" (Linux)", premiumTheme.Muted, premiumTheme.Panel); err != nil {
		return err
	}

	// Replace the obsolete wide branding subtitle with a compact content title
	// and truthful live connection state. The old header was already painted by
	// renderHeader, so clearing this strip prevents stale subtitle fragments.
	contentHeader := linuxRect{left: linuxMasterContentLeft, top: 0, right: u.width, bottom: 72}
	if err := u.x.fillRect(contentHeader.left, contentHeader.top, contentHeader.right-contentHeader.left, contentHeader.bottom-contentHeader.top, premiumTheme.Window); err != nil {
		return err
	}
	contentTitle := labels.Files
	if u.masterConnectionsVisible {
		contentTitle = labels.Connections
	}
	if err := u.x.text(linuxMasterContentLeft+12, 31, strings.ToUpper(contentTitle), premiumTheme.Text, premiumTheme.Window); err != nil {
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
	if err := u.x.text(max(linuxMasterContentLeft+180, u.width-300), 31, badge+"  "+u.version, badgeColor, premiumTheme.Window); err != nil {
		return err
	}
	return u.renderLinuxMasterToolbar()
}

func (u *linuxDesktop) handleLinuxMasterRailMouse(x, y int) bool {
	if u == nil {
		return false
	}
	rail := buildLinuxMasterRailLayout(u.width, u.height)
	switch {
	case rail.files.contains(x, y):
		u.masterConnectionsVisible = false
		u.focus = linuxFieldLocalPath
		u.setStatus(u.tr("section.local"))
	case rail.connections.contains(x, y):
		u.masterConnectionsVisible = true
		u.focus = linuxFieldHost
		u.setStatus(navigationLabelsForLanguage(u.language).Connections)
	case rail.transfers.contains(x, y):
		u.masterConnectionsVisible = false
		if len(u.transferJobs) > 0 && u.selectedTransfer < 0 {
			u.selectedTransfer = 0
		}
		u.setStatus(u.tr("section.transfers"))
	case rail.settings.contains(x, y):
		if !u.busy {
			u.openSettings()
		}
	default:
		return false
	}
	return true
}
