//go:build linux

package desktop

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/api"
	"github.com/bren-wp/Ghost-FTP/internal/brand"
	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/usererror"
)

const (
	linuxFieldProtocol = iota
	linuxFieldHost
	linuxFieldPort
	linuxFieldUser
	linuxFieldPassword
	linuxFieldKey
	linuxFieldPassphrase
	linuxFieldLocalPath
	linuxFieldRemotePath
	linuxFieldCount
)

const (
	linuxActionNone = iota
	linuxActionConnect
	linuxActionTrust
	linuxActionDisconnect
	linuxActionLocalRefresh
	linuxActionRemoteRefresh
	linuxActionTransfer
)

type linuxRect struct {
	left   int
	top    int
	right  int
	bottom int
}

func (r linuxRect) contains(x, y int) bool {
	return x >= r.left && x < r.right && y >= r.top && y < r.bottom
}

func linuxRectWH(left, top, width, height int) linuxRect {
	return linuxRect{left: left, top: top, right: left + width, bottom: top + height}
}

type linuxDesktopLayout struct {
	protocol, host, port, user, password, key, passphrase linuxRect
	profile, saveProfile, removeProfile, settings         linuxRect
	connect, disconnect, trust, cancelTrust               linuxRect
	localPath, localUp, localRefresh                      linuxRect
	remotePath, remoteUp, remoteRefresh                   linuxRect
	localNew, localRename, localDelete                    linuxRect
	remoteNew, remoteRename, remoteDelete, remoteChmod    linuxRect
	localList, remoteList                                 linuxRect
	upload, download                                      linuxRect
	promptInput, promptOK, promptCancel                   linuxRect
	pause, resume, cancelJob, retryJob, clearQueue        linuxRect
	queue                                                 linuxRect
}

func buildLinuxDesktopLayout(width, height int) linuxDesktopLayout {
	if width < premiumMinWidth {
		width = premiumMinWidth
	}
	if height < premiumMinHeight {
		height = premiumMinHeight
	}
	gap := premiumOuterGap
	contentWidth := width - 2*gap
	quickY := 92
	fieldH := 30
	rowGap := 8
	connectW := 108
	profileW := 150

	protocolW := 88
	portW := 66
	userW := 172
	passW := 160
	remaining := contentWidth - protocolW - portW - userW - passW - connectW - 5*rowGap
	if remaining < 260 {
		remaining = 260
	}
	hostW := remaining

	x := gap
	layout := linuxDesktopLayout{}
	layout.protocol = linuxRectWH(x, quickY, protocolW, fieldH)
	x += protocolW + rowGap
	layout.host = linuxRectWH(x, quickY, hostW, fieldH)
	x += hostW + rowGap
	layout.port = linuxRectWH(x, quickY, portW, fieldH)
	x += portW + rowGap
	layout.user = linuxRectWH(x, quickY, userW, fieldH)
	x += userW + rowGap
	layout.password = linuxRectWH(x, quickY, passW, fieldH)
	x += passW + rowGap
	layout.connect = linuxRectWH(x, quickY, connectW, fieldH)

	secondY := quickY + fieldH + rowGap
	x = gap
	layout.profile = linuxRectWH(x, secondY, profileW, fieldH)
	x += profileW + rowGap
	layout.saveProfile = linuxRectWH(x, secondY, 118, fieldH)
	x += 118 + rowGap
	layout.removeProfile = linuxRectWH(x, secondY, 118, fieldH)
	x += 118 + rowGap
	keyW := (contentWidth - (x - gap) - 120 - 118 - 4*rowGap) / 2
	if keyW < 150 {
		keyW = 150
	}
	layout.key = linuxRectWH(x, secondY, keyW, fieldH)
	x += keyW + rowGap
	layout.passphrase = linuxRectWH(x, secondY, keyW, fieldH)
	x += keyW + rowGap
	layout.disconnect = linuxRectWH(x, secondY, 118, fieldH)
	layout.settings = linuxRectWH(width-gap-92, 20, 76, 30)

	workspaceTop := secondY + fieldH + 24
	queueHeight := 164
	statusHeight := 34
	workspaceBottom := height - queueHeight - statusHeight - 2*gap
	if workspaceBottom < workspaceTop+220 {
		workspaceBottom = workspaceTop + 220
	}
	panelGap := premiumPanelGap
	panelWidth := (contentWidth - panelGap) / 2
	pathY := workspaceTop + 34
	pathH := 28
	buttonW := 66
	layout.localPath = linuxRectWH(gap, pathY, panelWidth-2*buttonW-2*rowGap, pathH)
	layout.localUp = linuxRectWH(layout.localPath.right+rowGap, pathY, buttonW, pathH)
	layout.localRefresh = linuxRectWH(layout.localUp.right+rowGap, pathY, buttonW, pathH)

	rightX := gap + panelWidth + panelGap
	layout.remotePath = linuxRectWH(rightX, pathY, panelWidth-2*buttonW-2*rowGap, pathH)
	layout.remoteUp = linuxRectWH(layout.remotePath.right+rowGap, pathY, buttonW, pathH)
	layout.remoteRefresh = linuxRectWH(layout.remoteUp.right+rowGap, pathY, buttonW, pathH)

	toolbarY := pathY + pathH + 8
	toolW := 80
	layout.localNew = linuxRectWH(gap, toolbarY, toolW, 28)
	layout.localRename = linuxRectWH(gap+toolW+rowGap, toolbarY, toolW, 28)
	layout.localDelete = linuxRectWH(gap+2*(toolW+rowGap), toolbarY, toolW, 28)
	layout.remoteNew = linuxRectWH(rightX, toolbarY, toolW, 28)
	layout.remoteRename = linuxRectWH(rightX+toolW+rowGap, toolbarY, toolW, 28)
	layout.remoteDelete = linuxRectWH(rightX+2*(toolW+rowGap), toolbarY, toolW, 28)
	layout.remoteChmod = linuxRectWH(rightX+3*(toolW+rowGap), toolbarY, toolW, 28)

	listY := toolbarY + 36
	listBottom := workspaceBottom - 44
	layout.localList = linuxRect{left: gap, top: listY, right: gap + panelWidth, bottom: listBottom}
	layout.remoteList = linuxRect{left: rightX, top: listY, right: rightX + panelWidth, bottom: listBottom}
	layout.upload = linuxRectWH(gap+panelWidth-112, workspaceBottom-34, 112, 30)
	layout.download = linuxRectWH(rightX, workspaceBottom-34, 112, 30)

	queueTop := workspaceBottom + 12
	layout.queue = linuxRect{left: gap, top: queueTop + 36, right: width - gap, bottom: height - statusHeight - gap - 8}
	buttonY := queueTop
	layout.pause = linuxRectWH(gap+90, buttonY, 74, 28)
	layout.resume = linuxRectWH(gap+172, buttonY, 74, 28)
	layout.cancelJob = linuxRectWH(gap+254, buttonY, 82, 28)
	layout.retryJob = linuxRectWH(gap+344, buttonY, 74, 28)
	layout.clearQueue = linuxRectWH(gap+426, buttonY, 110, 28)
	return layout
}

type linuxUIResult struct {
	action        int
	err           error
	connectResult bool
	requiresTrust bool
	fingerprint   string
	localBase     string
	localItems    []model.Item
	remoteItems   []model.Item
}

type linuxDesktop struct {
	x       *x11Client
	engine  *api.Engine
	version string

	language string
	layout   linuxDesktopLayout
	width    int
	height   int
	focus    int
	busy     bool
	action   int

	protocol   string
	host       string
	port       string
	username   string
	password   string
	keyPath    string
	passphrase string

	profiles          []model.PublicProfile
	profileIndex      int
	selectedProfileID string

	connected          bool
	pendingFingerprint string
	lastConnectConfig  model.ConnectionConfig
	localCurrent       string
	remoteCurrent      string
	localItems         []model.Item
	remoteItems        []model.Item
	selectedLocal      int
	selectedRemote     int
	transferJobs       []model.TransferJob
	selectedTransfer   int
	queuePaused        bool
	status             string
	confirmKind        string
	confirmUntil       time.Time
	promptKind         int
	promptTitle        string
	promptValue        string
	settingsOpen       bool
	settingsDraft      model.Settings
	settingsRects      linuxSettingsRects

	resultCh chan linuxUIResult
}

func newLinuxDesktop(x *x11Client, engine *api.Engine, version string) *linuxDesktop {
	u := &linuxDesktop{
		x:                x,
		engine:           engine,
		version:          version,
		language:         i18n.DefaultLanguage,
		width:            premiumStartWidth,
		height:           premiumStartHeight,
		focus:            linuxFieldHost,
		protocol:         "ftps",
		port:             "21",
		profileIndex:     -1,
		selectedLocal:    -1,
		selectedRemote:   -1,
		selectedTransfer: -1,
		remoteCurrent:    "/",
		resultCh:         make(chan linuxUIResult, 8),
	}
	if settings, err := engine.Settings(); err == nil {
		u.language = i18n.Normalize(settings.Language)
	}
	u.status = i18n.T(u.language, "status.ready")
	if profiles, err := engine.Profiles(); err == nil {
		u.profiles = profiles
	}
	u.layout = buildLinuxDesktopLayout(u.width, u.height)
	return u
}

func linuxTrimForUI(value string, max int) string {
	value = strings.ReplaceAll(strings.ReplaceAll(value, "\r", " "), "\n", " ")
	if max < 4 || len(value) <= max {
		return value
	}
	return value[:max-3] + "..."
}

func linuxHumanSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	if size < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	}
	if size < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(size)/(1024*1024*1024))
}

func (u *linuxDesktop) fieldValue(index int) string {
	switch index {
	case linuxFieldProtocol:
		return u.protocol
	case linuxFieldHost:
		return u.host
	case linuxFieldPort:
		return u.port
	case linuxFieldUser:
		return u.username
	case linuxFieldPassword:
		return u.password
	case linuxFieldKey:
		return u.keyPath
	case linuxFieldPassphrase:
		return u.passphrase
	case linuxFieldLocalPath:
		return u.localCurrent
	case linuxFieldRemotePath:
		return u.remoteCurrent
	default:
		return ""
	}
}

func (u *linuxDesktop) setFieldValue(index int, value string) {
	if len(value) > 4096 {
		value = value[:4096]
	}
	switch index {
	case linuxFieldHost:
		u.host = value
	case linuxFieldPort:
		u.port = value
	case linuxFieldUser:
		u.username = value
	case linuxFieldPassword:
		u.password = value
	case linuxFieldKey:
		u.keyPath = value
	case linuxFieldPassphrase:
		u.passphrase = value
	case linuxFieldLocalPath:
		u.localCurrent = value
	case linuxFieldRemotePath:
		u.remoteCurrent = value
	}
}

func (u *linuxDesktop) fieldRect(index int) linuxRect {
	switch index {
	case linuxFieldProtocol:
		return u.layout.protocol
	case linuxFieldHost:
		return u.layout.host
	case linuxFieldPort:
		return u.layout.port
	case linuxFieldUser:
		return u.layout.user
	case linuxFieldPassword:
		return u.layout.password
	case linuxFieldKey:
		return u.layout.key
	case linuxFieldPassphrase:
		return u.layout.passphrase
	case linuxFieldLocalPath:
		return u.layout.localPath
	case linuxFieldRemotePath:
		return u.layout.remotePath
	default:
		return linuxRect{}
	}
}

func (u *linuxDesktop) drawPanel(r linuxRect) error {
	if err := u.x.fillRect(r.left, r.top, r.right-r.left, r.bottom-r.top, premiumTheme.Panel); err != nil {
		return err
	}
	return u.x.strokeRect(r.left, r.top, r.right-r.left, r.bottom-r.top, premiumTheme.Border)
}

func (u *linuxDesktop) drawButton(r linuxRect, label string, enabled bool, accent bool) error {
	fill := premiumTheme.List
	text := premiumTheme.Text
	border := premiumTheme.Border
	if accent && enabled {
		fill = premiumTheme.Accent
		border = premiumTheme.AccentStrong
	}
	if !enabled {
		text = premiumTheme.Muted
	}
	if err := u.x.fillRect(r.left, r.top, r.right-r.left, r.bottom-r.top, fill); err != nil {
		return err
	}
	if err := u.x.strokeRect(r.left, r.top, r.right-r.left, r.bottom-r.top, border); err != nil {
		return err
	}
	return u.x.text(r.left+9, r.top+19, linuxTrimForUI(label, 24), text, fill)
}

func (u *linuxDesktop) drawField(index int, hint string) error {
	r := u.fieldRect(index)
	fill := premiumTheme.List
	border := premiumTheme.Border
	if u.focus == index {
		border = premiumTheme.Accent
	}
	if err := u.x.fillRect(r.left, r.top, r.right-r.left, r.bottom-r.top, fill); err != nil {
		return err
	}
	if err := u.x.strokeRect(r.left, r.top, r.right-r.left, r.bottom-r.top, border); err != nil {
		return err
	}
	value := u.fieldValue(index)
	if index == linuxFieldPassword || index == linuxFieldPassphrase {
		if value != "" {
			value = strings.Repeat("*", min(len(value), 32))
		}
	}
	color := premiumTheme.Text
	if value == "" {
		value = hint
		color = premiumTheme.Muted
	}
	return u.x.text(r.left+8, r.top+20, linuxTrimForUI(value, max(8, (r.right-r.left-16)/7)), color, fill)
}

func (u *linuxDesktop) renderHeader() error {
	if err := u.x.fillRect(0, 0, u.width, 72, premiumTheme.Panel); err != nil {
		return err
	}
	if err := u.x.text(premiumOuterGap, 30, strings.ToUpper(brand.ProductName), premiumTheme.Text, premiumTheme.Panel); err != nil {
		return err
	}
	if err := u.x.text(premiumOuterGap, 51, "FTP / FTPS / SFTP  |  "+u.tr("app.subtitle"), premiumTheme.Muted, premiumTheme.Panel); err != nil {
		return err
	}
	badge := strings.ToUpper(u.tr("badge.disconnected"))
	color := premiumTheme.Muted
	if u.connected {
		badge = strings.ToUpper(u.tr("badge.connected"))
		color = premiumTheme.Success
	} else if u.busy {
		badge = strings.ToUpper(linuxTrimForUI(u.tr("connection.connecting", u.host), 24))
		color = premiumTheme.Warn
	}
	if err := u.x.text(u.width-300, 34, badge+"  "+u.version, color, premiumTheme.Panel); err != nil {
		return err
	}
	return u.drawButton(u.layout.settings, u.tr("common.settings"), !u.busy, false)
}

func (u *linuxDesktop) renderQuickConnect() error {
	if err := u.x.text(premiumOuterGap, 84, strings.ToUpper(linuxTrimForUI(u.tr("profile.quick"), 28)), premiumTheme.Muted, premiumTheme.Window); err != nil {
		return err
	}
	if err := u.drawField(linuxFieldProtocol, u.tr("terminal.protocol")); err != nil {
		return err
	}
	if err := u.drawField(linuxFieldHost, u.tr("terminal.server")); err != nil {
		return err
	}
	if err := u.drawField(linuxFieldPort, u.tr("terminal.port")); err != nil {
		return err
	}
	if err := u.drawField(linuxFieldUser, u.tr("terminal.username")); err != nil {
		return err
	}
	if err := u.drawField(linuxFieldPassword, u.tr("terminal.password")); err != nil {
		return err
	}
	if err := u.drawButton(u.layout.connect, u.tr("common.connect"), !u.connected && !u.busy, true); err != nil {
		return err
	}

	profileLabel := u.tr("profile.quick")
	if u.profileIndex >= 0 && u.profileIndex < len(u.profiles) {
		profileLabel = u.profiles[u.profileIndex].Name
	}
	if err := u.drawButton(u.layout.profile, profileLabel, len(u.profiles) > 0, false); err != nil {
		return err
	}
	if err := u.drawButton(u.layout.saveProfile, u.tr("profile.save"), u.host != "" && !u.busy, false); err != nil {
		return err
	}
	if err := u.drawButton(u.layout.removeProfile, u.tr("profile.delete"), u.selectedProfileID != "" && !u.busy, false); err != nil {
		return err
	}
	if err := u.drawField(linuxFieldKey, u.tr("cue.private_key")); err != nil {
		return err
	}
	if err := u.drawField(linuxFieldPassphrase, u.tr("cue.passphrase")); err != nil {
		return err
	}
	return u.drawButton(u.layout.disconnect, u.tr("common.disconnect"), u.connected && !u.busy, false)
}

func (u *linuxDesktop) renderItemRows(r linuxRect, items []model.Item, selected int) error {
	rowH := 24
	maxRows := (r.bottom - r.top - 26) / rowH
	if maxRows < 0 {
		maxRows = 0
	}
	if err := u.x.fillRect(r.left, r.top, r.right-r.left, r.bottom-r.top, premiumTheme.List); err != nil {
		return err
	}
	if err := u.x.strokeRect(r.left, r.top, r.right-r.left, r.bottom-r.top, premiumTheme.Border); err != nil {
		return err
	}
	if err := u.x.text(r.left+8, r.top+17, strings.ToUpper(u.tr("column.name")), premiumTheme.Muted, premiumTheme.List); err != nil {
		return err
	}
	if err := u.x.text(r.right-112, r.top+17, strings.ToUpper(u.tr("column.size")), premiumTheme.Muted, premiumTheme.List); err != nil {
		return err
	}
	for i := 0; i < len(items) && i < maxRows; i++ {
		item := items[i]
		y := r.top + 28 + i*rowH
		bg := premiumTheme.List
		if i == selected {
			bg = premiumTheme.Selection
			if err := u.x.fillRect(r.left+1, y-17, r.right-r.left-2, rowH, bg); err != nil {
				return err
			}
		}
		prefix := "FILE  "
		if item.IsDirectory {
			prefix = "DIR   "
		} else if item.IsSymlink {
			prefix = "LINK  "
		}
		if err := u.x.text(r.left+8, y, linuxTrimForUI(prefix+item.Name, max(12, (r.right-r.left-150)/7)), premiumTheme.Text, bg); err != nil {
			return err
		}
		size := linuxHumanSize(item.Size)
		if item.IsDirectory {
			size = "--"
		}
		if err := u.x.text(r.right-112, y, size, premiumTheme.Muted, bg); err != nil {
			return err
		}
	}
	return nil
}

func (u *linuxDesktop) renderWorkspace() error {
	leftPanel := linuxRect{left: premiumOuterGap - 6, top: u.layout.localPath.top - 30, right: u.layout.localList.right + 6, bottom: u.layout.upload.bottom + 8}
	rightPanel := linuxRect{left: u.layout.remotePath.left - 6, top: leftPanel.top, right: u.layout.remoteList.right + 6, bottom: leftPanel.bottom}
	if err := u.drawPanel(leftPanel); err != nil {
		return err
	}
	if err := u.drawPanel(rightPanel); err != nil {
		return err
	}
	if err := u.x.text(premiumOuterGap, leftPanel.top+20, strings.ToUpper(u.tr("section.local")), premiumTheme.Muted, premiumTheme.Panel); err != nil {
		return err
	}
	if err := u.x.text(u.layout.remotePath.left, leftPanel.top+20, strings.ToUpper(u.tr("section.remote")), premiumTheme.Muted, premiumTheme.Panel); err != nil {
		return err
	}
	if err := u.drawField(linuxFieldLocalPath, u.tr("column.local")); err != nil {
		return err
	}
	if err := u.drawButton(u.layout.localUp, u.tr("common.up"), !u.busy, false); err != nil {
		return err
	}
	if err := u.drawButton(u.layout.localRefresh, u.tr("common.refresh"), !u.busy, false); err != nil {
		return err
	}
	if err := u.drawField(linuxFieldRemotePath, u.tr("column.remote")); err != nil {
		return err
	}
	if err := u.drawButton(u.layout.remoteUp, u.tr("common.up"), u.connected && !u.busy, false); err != nil {
		return err
	}
	if err := u.drawButton(u.layout.remoteRefresh, u.tr("common.refresh"), u.connected && !u.busy, false); err != nil {
		return err
	}
	if err := u.drawButton(u.layout.localNew, u.tr("common.new_folder"), !u.busy, false); err != nil {
		return err
	}
	if err := u.drawButton(u.layout.localRename, u.tr("common.rename"), u.selectedLocal >= 0 && !u.busy, false); err != nil {
		return err
	}
	if err := u.drawButton(u.layout.localDelete, u.tr("common.delete"), u.selectedLocal >= 0 && !u.busy, false); err != nil {
		return err
	}
	if err := u.drawButton(u.layout.remoteNew, u.tr("common.new_folder"), u.connected && !u.busy, false); err != nil {
		return err
	}
	if err := u.drawButton(u.layout.remoteRename, u.tr("common.rename"), u.connected && u.selectedRemote >= 0 && !u.busy, false); err != nil {
		return err
	}
	if err := u.drawButton(u.layout.remoteDelete, u.tr("common.delete"), u.connected && u.selectedRemote >= 0 && !u.busy, false); err != nil {
		return err
	}
	if err := u.drawButton(u.layout.remoteChmod, u.tr("common.permissions"), u.connected && u.selectedRemote >= 0 && !u.busy, false); err != nil {
		return err
	}
	if err := u.renderItemRows(u.layout.localList, u.localItems, u.selectedLocal); err != nil {
		return err
	}
	if err := u.renderItemRows(u.layout.remoteList, u.remoteItems, u.selectedRemote); err != nil {
		return err
	}
	if err := u.drawButton(u.layout.upload, u.tr("transfer.upload")+" →", u.connected && u.selectedLocal >= 0 && !u.busy, true); err != nil {
		return err
	}
	return u.drawButton(u.layout.download, "← "+u.tr("transfer.download"), u.connected && u.selectedRemote >= 0 && !u.busy, true)
}

func (u *linuxDesktop) renderQueue() error {
	if err := u.x.text(premiumOuterGap, u.layout.pause.top+19, strings.ToUpper(u.tr("section.transfers")), premiumTheme.Muted, premiumTheme.Window); err != nil {
		return err
	}
	if err := u.drawButton(u.layout.pause, u.tr("transfer.pause"), !u.queuePaused, false); err != nil {
		return err
	}
	if err := u.drawButton(u.layout.resume, u.tr("transfer.resume"), u.queuePaused, false); err != nil {
		return err
	}
	if err := u.drawButton(u.layout.cancelJob, u.tr("common.cancel"), u.selectedTransfer >= 0, false); err != nil {
		return err
	}
	if err := u.drawButton(u.layout.retryJob, u.tr("transfer.retry"), u.selectedTransfer >= 0, false); err != nil {
		return err
	}
	if err := u.drawButton(u.layout.clearQueue, u.tr("transfer.clear"), len(u.transferJobs) > 0, false); err != nil {
		return err
	}
	if err := u.x.fillRect(u.layout.queue.left, u.layout.queue.top, u.layout.queue.right-u.layout.queue.left, u.layout.queue.bottom-u.layout.queue.top, premiumTheme.List); err != nil {
		return err
	}
	if err := u.x.strokeRect(u.layout.queue.left, u.layout.queue.top, u.layout.queue.right-u.layout.queue.left, u.layout.queue.bottom-u.layout.queue.top, premiumTheme.Border); err != nil {
		return err
	}
	if err := u.x.text(u.layout.queue.left+8, u.layout.queue.top+17, strings.ToUpper(u.tr("column.direction")+"   "+u.tr("column.local")+" / "+u.tr("column.remote")), premiumTheme.Muted, premiumTheme.List); err != nil {
		return err
	}
	rowH := 22
	maxRows := (u.layout.queue.bottom - u.layout.queue.top - 28) / rowH
	for i := 0; i < len(u.transferJobs) && i < maxRows; i++ {
		job := u.transferJobs[i]
		y := u.layout.queue.top + 28 + i*rowH
		bg := premiumTheme.List
		if i == u.selectedTransfer {
			bg = premiumTheme.Selection
			if err := u.x.fillRect(u.layout.queue.left+1, y-16, u.layout.queue.right-u.layout.queue.left-2, rowH, bg); err != nil {
				return err
			}
		}
		line := fmt.Sprintf("%-10s %s  ->  %s", strings.ToUpper(job.Direction), filepath.Base(job.LocalPath), job.RemotePath)
		if err := u.x.text(u.layout.queue.left+8, y, linuxTrimForUI(line, max(24, (u.layout.queue.right-u.layout.queue.left-220)/7)), premiumTheme.Text, bg); err != nil {
			return err
		}
		status := fmt.Sprintf("%-10s %3.0f%%", job.Status, job.Progress*100)
		if err := u.x.text(u.layout.queue.right-170, y, status, premiumTheme.Muted, bg); err != nil {
			return err
		}
	}
	return nil
}

func (u *linuxDesktop) render() error {
	u.layout = buildLinuxDesktopLayout(u.width, u.height)
	if err := u.x.fillRect(0, 0, u.width, u.height, premiumTheme.Window); err != nil {
		return err
	}
	if err := u.renderHeader(); err != nil {
		return err
	}
	if err := u.renderQuickConnect(); err != nil {
		return err
	}
	if err := u.renderWorkspace(); err != nil {
		return err
	}
	if err := u.renderQueue(); err != nil {
		return err
	}
	statusY := u.height - 15
	status := linuxTrimForUI(u.status, max(20, (u.width-2*premiumOuterGap)/7))
	return u.x.text(premiumOuterGap, statusY, status, premiumTheme.Muted, premiumTheme.Window)
}

func (u *linuxDesktop) setStatus(message string) {
	u.status = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(message, "\r", " "), "\n", " "))
}

func (u *linuxDesktop) startAction(action int, fn func() linuxUIResult) {
	if u.busy {
		return
	}
	u.busy = true
	u.action = action
	go func() {
		result := fn()
		result.action = action
		u.resultCh <- result
	}()
}

func (u *linuxDesktop) refreshLocal(target string) {
	if u.busy {
		return
	}
	if target == "" {
		target = u.localCurrent
	}
	u.setStatus(u.tr("common.refresh") + " · " + u.tr("section.local"))
	u.startAction(linuxActionLocalRefresh, func() linuxUIResult {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		base, items, err := u.engine.LocalList(ctx, target)
		return linuxUIResult{localBase: base, localItems: items, err: err}
	})
}

func (u *linuxDesktop) refreshRemote(target string) {
	if u.busy || !u.connected {
		return
	}
	if target == "" {
		target = u.remoteCurrent
	}
	u.setStatus(u.tr("common.refresh") + " · " + u.tr("section.remote"))
	u.startAction(linuxActionRemoteRefresh, func() linuxUIResult {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		items, err := u.engine.RemoteList(ctx, target)
		return linuxUIResult{remoteItems: items, localBase: target, err: err}
	})
}

func (u *linuxDesktop) connectToServer(trust string) {
	if u.busy || u.connected {
		return
	}
	port, err := validateRawConnectionInput(u.protocol, u.host, u.port, u.username)
	if err != nil {
		u.setStatus(i18n.T(u.language, "connection.invalid_data_body"))
		return
	}
	cfg := model.ConnectionConfig{
		Protocol:       u.protocol,
		Host:           u.host,
		Port:           port,
		Username:       u.username,
		Password:       u.password,
		PrivateKeyPath: u.keyPath,
		Passphrase:     u.passphrase,
	}
	if trust != "" {
		cfg = u.lastConnectConfig
		cfg.Password = ""
		cfg.Passphrase = ""
	}
	profileID := u.selectedProfileID
	u.lastConnectConfig = cfg
	u.lastConnectConfig.Password = ""
	u.lastConnectConfig.Passphrase = ""
	u.password = ""
	u.passphrase = ""
	u.setStatus(u.tr("connection.connecting", u.host))
	u.startAction(linuxActionConnect, func() linuxUIResult {
		defer func() {
			cfg.Password = ""
			cfg.Passphrase = ""
		}()
		ctx, cancel := context.WithTimeout(context.Background(), connectionTimeoutDuration(func() model.Settings { settings, _ := u.engine.Settings(); return settings }()))
		defer cancel()
		result, err := u.engine.Connect(ctx, profileID, cfg, trust, trust != "")
		return linuxUIResult{
			err:           err,
			connectResult: result.Connected,
			requiresTrust: result.RequiresTrust,
			fingerprint:   result.Fingerprint,
		}
	})
}

func (u *linuxDesktop) disconnect() {
	if u.busy || !u.connected {
		return
	}
	u.setStatus(u.tr("disconnect.progress"))
	u.startAction(linuxActionDisconnect, func() linuxUIResult {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		return linuxUIResult{err: u.engine.Disconnect(ctx)}
	})
}

func (u *linuxDesktop) selectedLocalItem() (model.Item, bool) {
	if u.selectedLocal < 0 || u.selectedLocal >= len(u.localItems) {
		return model.Item{}, false
	}
	return u.localItems[u.selectedLocal], true
}

func (u *linuxDesktop) selectedRemoteItem() (model.Item, bool) {
	if u.selectedRemote < 0 || u.selectedRemote >= len(u.remoteItems) {
		return model.Item{}, false
	}
	return u.remoteItems[u.selectedRemote], true
}

func (u *linuxDesktop) queueTransfer(direction string) {
	if u.busy || !u.connected {
		return
	}
	var localPath, remotePath string
	var item model.Item
	var ok bool
	if direction == "upload" {
		item, ok = u.selectedLocalItem()
		if !ok || item.IsSymlink {
			return
		}
		localPath = filepath.Join(u.localCurrent, item.Name)
		remotePath = terminalRemotePath(u.remoteCurrent, item.Name)
	} else {
		item, ok = u.selectedRemoteItem()
		if !ok || item.IsSymlink {
			return
		}
		localPath = filepath.Join(u.localCurrent, item.Name)
		remotePath = terminalRemotePath(u.remoteCurrent, item.Name)
	}
	if item.IsDirectory {
		u.setStatus("Queueing folder tree transfer...")
		u.startAction(linuxActionTransfer, func() linuxUIResult {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			_, err := u.engine.AddTreeTransfer(ctx, direction, localPath, remotePath)
			return linuxUIResult{err: err}
		})
		return
	}
	_, err := u.engine.AddTransfer(direction, localPath, remotePath, u.localCurrent)
	if err != nil {
		u.setStatus(usererror.MessageFor(u.language, err, i18n.T(u.language, "error.generic")))
		return
	}
	u.setStatus("Transfer queued.")
	u.transferJobs = u.engine.Transfers()
}

func (u *linuxDesktop) cycleProtocol() {
	protocols := []struct {
		name string
		port string
	}{{"ftp", "21"}, {"ftps", "21"}, {"ftpsi", "990"}, {"sftp", "22"}}
	index := 0
	for i, p := range protocols {
		if p.name == u.protocol {
			index = (i + 1) % len(protocols)
			break
		}
	}
	u.protocol = protocols[index].name
	u.port = protocols[index].port
	if u.protocol != "sftp" {
		u.keyPath = ""
		u.passphrase = ""
	}
}

func (u *linuxDesktop) cycleProfile() {
	if len(u.profiles) == 0 {
		return
	}
	u.profileIndex = (u.profileIndex + 1) % len(u.profiles)
	p := u.profiles[u.profileIndex]
	u.selectedProfileID = p.ID
	u.protocol = p.Protocol
	u.host = p.Host
	u.port = strconv.Itoa(p.Port)
	u.username = p.Username
	u.password = ""
	u.keyPath = p.PrivateKeyPath
	u.passphrase = ""
	if p.LocalPath != "" {
		u.localCurrent = p.LocalPath
	}
	if p.RemotePath != "" {
		u.remoteCurrent = p.RemotePath
	}
	u.pendingFingerprint = ""
	u.setStatus("Loaded profile: " + p.Name + ". Linux login secrets are not persisted.")
}

func (u *linuxDesktop) saveProfile() {
	if u.busy || strings.TrimSpace(u.host) == "" {
		return
	}
	port, err := validateRawConnectionInput(u.protocol, u.host, u.port, u.username)
	if err != nil {
		u.setStatus("Profile was not saved: invalid connection details.")
		return
	}
	name := strings.TrimSpace(u.host)
	if name == "" {
		name = "Ghost FTP profile"
	}
	saved, err := u.engine.SaveProfile(model.ProfileInput{
		ID:             u.selectedProfileID,
		Name:           name,
		Protocol:       u.protocol,
		Host:           u.host,
		Port:           port,
		Username:       u.username,
		PrivateKeyPath: u.keyPath,
		Fingerprint:    u.pendingFingerprint,
		RemotePath:     u.remoteCurrent,
		LocalPath:      u.localCurrent,
	})
	if err != nil {
		u.setStatus(usererror.MessageFor(u.language, err, "Profile could not be saved."))
		return
	}
	u.selectedProfileID = saved.ID
	if profiles, listErr := u.engine.Profiles(); listErr == nil {
		u.profiles = profiles
		for i := range profiles {
			if profiles[i].ID == saved.ID {
				u.profileIndex = i
				break
			}
		}
	}
	u.setStatus("Profile saved without a persistent Linux password/passphrase.")
}

func (u *linuxDesktop) removeProfile() {
	if u.busy || u.selectedProfileID == "" {
		return
	}
	if err := u.engine.RemoveProfile(u.selectedProfileID); err != nil {
		u.setStatus(usererror.MessageFor(u.language, err, "Profile could not be removed."))
		return
	}
	u.selectedProfileID = ""
	u.profileIndex = -1
	if profiles, err := u.engine.Profiles(); err == nil {
		u.profiles = profiles
	}
	u.setStatus(u.tr("profile.delete"))
}

func (u *linuxDesktop) handleResult(result linuxUIResult) {
	u.busy = false
	u.action = linuxActionNone
	if result.err != nil {
		u.setStatus(usererror.MessageFor(u.language, result.err, i18n.T(u.language, "error.generic")))
		return
	}
	switch result.action {
	case linuxActionConnect:
		if result.requiresTrust {
			u.pendingFingerprint = result.fingerprint
			u.setStatus("SFTP host key: " + result.fingerprint + " | Verify it, then click Trust.")
			return
		}
		if result.connectResult {
			u.connected = true
			if u.protocol == "sftp" && u.remoteCurrent == "/" {
				u.remoteCurrent = "."
			}
			u.setStatus(u.tr("connection.connected", u.host))
			return
		}
	case linuxActionDisconnect:
		u.connected = false
		u.pendingFingerprint = ""
		u.remoteItems = nil
		u.selectedRemote = -1
		u.setStatus(u.tr("disconnect.done"))
	case linuxActionLocalRefresh:
		u.localCurrent = result.localBase
		u.localItems = result.localItems
		u.selectedLocal = -1
		u.setStatus(fmt.Sprintf("Local files refreshed: %d items.", len(result.localItems)))
	case linuxActionRemoteRefresh:
		u.remoteCurrent = result.localBase
		u.remoteItems = result.remoteItems
		u.selectedRemote = -1
		u.setStatus(fmt.Sprintf("Server files refreshed: %d items.", len(result.remoteItems)))
	case linuxActionTransfer:
		u.transferJobs = u.engine.Transfers()
		u.setStatus("Folder transfer queued.")
	}
}

func (u *linuxDesktop) selectRow(r linuxRect, y int, count int) int {
	index := (y - r.top - 11) / 24
	if index < 0 || index >= count {
		return -1
	}
	return index
}

func (u *linuxDesktop) handleMouse(x, y int) {
	for i := 0; i < linuxFieldCount; i++ {
		if u.fieldRect(i).contains(x, y) {
			if i == linuxFieldProtocol {
				u.cycleProtocol()
				u.focus = linuxFieldHost
			} else {
				u.focus = i
			}
			return
		}
	}
	l := u.layout
	switch {
	case l.connect.contains(x, y):
		u.connectToServer("")
	case l.disconnect.contains(x, y):
		u.disconnect()
	case l.settings.contains(x, y):
		u.openSettings()
	case l.profile.contains(x, y):
		u.cycleProfile()
	case l.saveProfile.contains(x, y):
		u.saveProfile()
	case l.removeProfile.contains(x, y):
		u.removeProfile()
	case l.localUp.contains(x, y):
		u.refreshLocal(filepath.Dir(u.localCurrent))
	case l.localRefresh.contains(x, y):
		u.refreshLocal(u.localCurrent)
	case l.remoteUp.contains(x, y):
		u.refreshRemote(terminalRemotePath(u.remoteCurrent, ".."))
	case l.remoteRefresh.contains(x, y):
		u.refreshRemote(u.remoteCurrent)
	case l.localNew.contains(x, y):
		u.openPrompt(linuxPromptLocalMkdir, u.tr("common.new_folder")+" · "+u.tr("section.local"), u.tr("common.new_folder"))
	case l.localRename.contains(x, y):
		u.openSelectedLocalRename()
	case l.localDelete.contains(x, y):
		u.deleteSelectedLocal()
	case l.remoteNew.contains(x, y):
		u.openPrompt(linuxPromptRemoteMkdir, u.tr("common.new_folder")+" · "+u.tr("section.remote"), u.tr("common.new_folder"))
	case l.remoteRename.contains(x, y):
		u.openSelectedRemoteRename()
	case l.remoteDelete.contains(x, y):
		u.deleteSelectedRemote()
	case l.remoteChmod.contains(x, y):
		u.openSelectedRemoteChmod()
	case l.localList.contains(x, y):
		u.selectedLocal = u.selectRow(l.localList, y, len(u.localItems))
	case l.remoteList.contains(x, y):
		u.selectedRemote = u.selectRow(l.remoteList, y, len(u.remoteItems))
	case l.upload.contains(x, y):
		u.queueTransfer("upload")
	case l.download.contains(x, y):
		u.queueTransfer("download")
	case l.pause.contains(x, y):
		u.engine.PauseTransfers()
		u.queuePaused = true
		u.setStatus("Transfer queue paused.")
	case l.resume.contains(x, y):
		u.engine.ResumeTransfers()
		u.queuePaused = false
		u.setStatus("Transfer queue resumed.")
	case l.cancelJob.contains(x, y):
		if u.selectedTransfer >= 0 && u.selectedTransfer < len(u.transferJobs) {
			_ = u.engine.CancelTransfer(u.transferJobs[u.selectedTransfer].ID)
		}
	case l.retryJob.contains(x, y):
		if u.selectedTransfer >= 0 && u.selectedTransfer < len(u.transferJobs) {
			_ = u.engine.RetryTransfer(u.transferJobs[u.selectedTransfer].ID)
		}
	case l.clearQueue.contains(x, y):
		u.engine.ClearFinishedTransfers()
		u.selectedTransfer = -1
	case l.queue.contains(x, y):
		index := (y - l.queue.top - 11) / 22
		if index >= 0 && index < len(u.transferJobs) {
			u.selectedTransfer = index
		}
	}
}

func linuxKeysymText(sym uint32) (string, bool) {
	if sym >= 0x20 && sym <= 0x7e {
		return string(rune(sym)), true
	}
	return "", false
}

func (u *linuxDesktop) handleKey(keycode byte, state uint16) bool {
	sym := u.x.keysym(keycode, state)
	if u.handleSettingsKey(sym) {
		return true
	}
	if u.handlePromptKey(sym) {
		return true
	}
	if state&x11ControlMask != 0 {
		if sym == 'q' || sym == 'Q' {
			return false
		}
		if sym == 'r' || sym == 'R' {
			if u.connected {
				u.refreshRemote(u.remoteCurrent)
			} else {
				u.refreshLocal(u.localCurrent)
			}
			return true
		}
	}
	switch sym {
	case x11KeyEscape:
		if u.pendingFingerprint != "" && !u.connected {
			u.engine.CancelPendingTrust()
			u.pendingFingerprint = ""
			u.setStatus(u.tr("sftp.cancelled"))
		}
		return true
	case x11KeyTab:
		u.focus = (u.focus + 1) % linuxFieldCount
		return true
	case x11KeyReturn:
		switch u.focus {
		case linuxFieldLocalPath:
			u.refreshLocal(u.localCurrent)
		case linuxFieldRemotePath:
			u.refreshRemote(u.remoteCurrent)
		default:
			if u.pendingFingerprint != "" && !u.connected {
				u.connectToServer(u.pendingFingerprint)
			} else if !u.connected {
				u.connectToServer("")
			}
		}
		return true
	case x11KeyBackSpace, x11KeyDelete:
		if u.focus == linuxFieldProtocol {
			return true
		}
		value := u.fieldValue(u.focus)
		if len(value) > 0 {
			u.setFieldValue(u.focus, value[:len(value)-1])
		}
		return true
	}
	if u.focus == linuxFieldProtocol {
		return true
	}
	if text, ok := linuxKeysymText(sym); ok {
		if u.focus == linuxFieldPort && (text[0] < '0' || text[0] > '9') {
			return true
		}
		value := u.fieldValue(u.focus)
		if len(value)+len(text) <= 4096 {
			u.setFieldValue(u.focus, value+text)
		}
	}
	return true
}

func (u *linuxDesktop) renderTrustOverlay() error {
	if u.pendingFingerprint == "" || u.connected {
		return nil
	}
	w := min(720, u.width-80)
	h := 150
	left := (u.width - w) / 2
	top := (u.height - h) / 2
	panel := linuxRectWH(left, top, w, h)
	if err := u.drawPanel(panel); err != nil {
		return err
	}
	if err := u.x.text(left+20, top+28, "NEW SFTP HOST KEY", premiumTheme.Warn, premiumTheme.Panel); err != nil {
		return err
	}
	if err := u.x.text(left+20, top+53, "Verify this SHA-256 fingerprint before trusting the server:", premiumTheme.Text, premiumTheme.Panel); err != nil {
		return err
	}
	if err := u.x.text(left+20, top+78, linuxTrimForUI(u.pendingFingerprint, 92), premiumTheme.Text, premiumTheme.Panel); err != nil {
		return err
	}
	u.layout.trust = linuxRectWH(left+w-230, top+h-42, 96, 28)
	u.layout.cancelTrust = linuxRectWH(left+w-124, top+h-42, 96, 28)
	if err := u.drawButton(u.layout.trust, u.tr("sftp.trust"), !u.busy, true); err != nil {
		return err
	}
	return u.drawButton(u.layout.cancelTrust, u.tr("common.cancel"), !u.busy, false)
}

func (u *linuxDesktop) renderAll() error {
	if err := u.render(); err != nil {
		return err
	}
	if u.settingsOpen {
		return u.renderSettingsOverlay()
	}
	if u.promptKind != linuxPromptNone {
		return u.renderPromptOverlay()
	}
	return u.renderTrustOverlay()
}

func (u *linuxDesktop) handleOverlayMouse(x, y int) bool {
	if u.handleSettingsMouse(x, y) {
		return true
	}
	if u.handlePromptMouse(x, y) {
		return true
	}
	if u.pendingFingerprint == "" || u.connected {
		return false
	}
	if u.layout.trust.contains(x, y) {
		u.connectToServer(u.pendingFingerprint)
		return true
	}
	if u.layout.cancelTrust.contains(x, y) {
		u.engine.CancelPendingTrust()
		u.pendingFingerprint = ""
		u.setStatus(u.tr("sftp.cancelled"))
		return true
	}
	return true
}

func runLinuxGUI(engine *api.Engine, version string) error {
	x, err := connectLocalX11()
	if err != nil {
		return fmt.Errorf("Linux desktop UI is unavailable: %w", err)
	}
	defer x.close()
	if err := x.createWindow(premiumStartWidth, premiumStartHeight, brand.ProductName+" "+version+" - "+brand.Company); err != nil {
		return fmt.Errorf("Linux desktop window could not be created: %w", err)
	}
	u := newLinuxDesktop(x, engine, version)
	if err := u.renderAll(); err != nil {
		return err
	}

	events := make(chan x11Event, 32)
	eventErr := make(chan error, 1)
	go func() {
		for {
			event, err := x.nextEvent()
			if err != nil {
				eventErr <- err
				return
			}
			events <- event
		}
	}()

	u.refreshLocal("")
	ticker := time.NewTicker(750 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case err := <-eventErr:
			if errors.Is(err, os.ErrClosed) {
				return nil
			}
			return err
		case result := <-u.resultCh:
			u.handleResult(result)
			if u.connected && result.action == linuxActionConnect && result.connectResult {
				u.refreshRemote(u.remoteCurrent)
			}
			if err := u.renderAll(); err != nil {
				return err
			}
		case event := <-events:
			switch event.Type {
			case x11Expose:
				if err := u.renderAll(); err != nil {
					return err
				}
			case x11ConfigureNotify:
				if event.Width > 0 {
					u.width = event.Width
				}
				if event.Height > 0 {
					u.height = event.Height
				}
				if err := u.renderAll(); err != nil {
					return err
				}
			case x11ButtonPress:
				if !u.handleOverlayMouse(event.X, event.Y) {
					u.handleMouse(event.X, event.Y)
				}
				if err := u.renderAll(); err != nil {
					return err
				}
			case x11KeyPress:
				if !u.handleKey(event.Detail, event.State) {
					return nil
				}
				if err := u.renderAll(); err != nil {
					return err
				}
			case x11ClientMessage:
				if event.Atom == x.wmDelete {
					return nil
				}
			case x11DestroyNotify:
				return nil
			}
		case <-ticker.C:
			jobs := u.engine.Transfers()
			if reflect.DeepEqual(jobs, u.transferJobs) {
				continue
			}
			u.transferJobs = jobs
			if u.selectedTransfer >= len(u.transferJobs) {
				u.selectedTransfer = -1
			}
			if err := u.renderAll(); err != nil {
				return err
			}
		}
	}
}
