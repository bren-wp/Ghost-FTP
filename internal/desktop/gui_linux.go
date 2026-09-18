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
	"unicode"
	"unicode/utf8"

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
	linuxActionUpdateCheck
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
	action          int
	err             error
	connectResult   bool
	requiresTrust   bool
	fingerprint     string
	localBase       string
	localItems      []model.Item
	remoteItems     []model.Item
	updateLatest    string
	updateURL       string
	updateAvailable bool
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

	connected            bool
	pendingFingerprint   string
	lastConnectConfig    model.ConnectionConfig
	localCurrent         string
	remoteCurrent        string
	localItems           []model.Item
	remoteItems          []model.Item
	selectedLocal        int
	selectedRemote       int
	transferJobs         []model.TransferJob
	selectedTransfer     int
	queuePaused          bool
	status               string
	confirmKind          string
	confirmUntil         time.Time
	promptKind           int
	promptTitle          string
	promptValue          string
	settingsOpen         bool
	settingsDraft        model.Settings
	settingsRects        linuxSettingsRects
	infoOverlay          linuxInfoOverlayKind
	pendingUpdateURL     string
	pendingUpdateVersion string

	masterConnectionsVisible bool
	lastFilePaneRemote       bool
	workspaceBackHistory     []linuxWorkspaceHistoryEntry
	workspaceForwardHistory  []linuxWorkspaceHistoryEntry
	workspaceReplayActive    bool
	workspaceReplayBack      bool
	workspaceReplayTarget    linuxWorkspaceHistoryEntry

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
		setActiveTheme(settings.Appearance)
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
	if max < 4 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max-3]) + "..."
}

func linuxButtonLabelLimit(r linuxRect) int {
	return max(4, (r.right-r.left-12)/6)
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

func linuxTruncateUTF8Bytes(value string, maxBytes int) string {
	if maxBytes < 0 {
		maxBytes = 0
	}
	if len(value) <= maxBytes {
		return value
	}
	end := maxBytes
	for end > 0 && !utf8.ValidString(value[:end]) {
		end--
	}
	return value[:end]
}

func linuxDeleteLastRune(value string) string {
	if value == "" {
		return ""
	}
	_, size := utf8.DecodeLastRuneInString(value)
	if size <= 0 {
		return ""
	}
	return value[:len(value)-size]
}

func (u *linuxDesktop) setFieldValue(index int, value string) {
	value = linuxTruncateUTF8Bytes(value, 4096)
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

func (u *linuxDesktop) drawButtonWithLimit(r linuxRect, label string, enabled bool, accent bool, labelLimit int) error {
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
	return u.x.text(r.left+9, r.top+19, linuxTrimForUI(label, max(4, labelLimit)), text, fill)
}

func (u *linuxDesktop) drawButton(r linuxRect, label string, enabled bool, accent bool) error {
	return u.drawButtonWithLimit(r, label, enabled, accent, 24)
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
	if err := u.renderBookmarksHeaderButton(); err != nil {
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
	if i := strings.Index(profileLabel, " ("); i > 0 {
		profileLabel = strings.TrimSpace(profileLabel[:i])
	}
	if u.profileIndex >= 0 && u.profileIndex < len(u.profiles) {
		profileLabel = u.profiles[u.profileIndex].Name
	}
	if err := u.drawButton(u.layout.profile, profileLabel, len(u.profiles) > 0 && !u.busy && !u.connected, false); err != nil {
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

func linuxModifiedLabel(item model.Item) string {
	if item.Modified.IsZero() {
		return "—"
	}
	return item.Modified.Local().Format("2006-01-02 15:04")
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
	width := r.right - r.left
	remote := r.left >= u.layout.remoteList.left
	nameX := r.left + 8
	sizeX := r.left + width*54/100
	modifiedX := r.left + width*70/100
	permissionsX := r.left + width*88/100
	if remote {
		sizeX = r.left + width*48/100
		modifiedX = r.left + width*64/100
		permissionsX = r.left + width*84/100
	}
	headerY := r.top + 17
	if err := u.x.text(nameX, headerY, u.tr("column.name"), premiumTheme.Muted, premiumTheme.List); err != nil {
		return err
	}
	if err := u.x.text(sizeX, headerY, u.tr("column.size"), premiumTheme.Muted, premiumTheme.List); err != nil {
		return err
	}
	if err := u.x.text(modifiedX, headerY, u.tr("column.modified"), premiumTheme.Muted, premiumTheme.List); err != nil {
		return err
	}
	if remote {
		if err := u.x.text(permissionsX, headerY, u.tr("common.permissions"), premiumTheme.Muted, premiumTheme.List); err != nil {
			return err
		}
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
		iconX := nameX
		if item.IsDirectory {
			if err := u.x.fillRect(iconX, y-11, 13, 9, premiumTheme.Accent); err != nil {
				return err
			}
			if err := u.x.fillRect(iconX+2, y-14, 6, 3, premiumTheme.AccentStrong); err != nil {
				return err
			}
		} else {
			if err := u.x.strokeRect(iconX, y-14, 11, 13, premiumTheme.Muted); err != nil {
				return err
			}
		}
		nameWidth := max(10, (sizeX-(nameX+22)-8)/7)
		if err := u.x.text(nameX+22, y, linuxTrimForUI(item.Name, nameWidth), premiumTheme.Text, bg); err != nil {
			return err
		}
		size := linuxHumanSize(item.Size)
		if item.IsDirectory {
			size = "—"
		}
		if err := u.x.text(sizeX, y, size, premiumTheme.Muted, bg); err != nil {
			return err
		}
		if err := u.x.text(modifiedX, y, linuxModifiedLabel(item), premiumTheme.Muted, bg); err != nil {
			return err
		}
		if remote {
			permissions := strings.TrimSpace(item.Permissions)
			if permissions == "" {
				permissions = "—"
			}
			if err := u.x.text(permissionsX, y, linuxTrimForUI(permissions, 12), premiumTheme.Muted, bg); err != nil {
				return err
			}
		}
	}
	return nil
}

func (u *linuxDesktop) renderWorkspace() error {
	leftPanel := linuxRect{left: u.layout.localPath.left - 6, top: u.layout.localPath.top - 30, right: u.layout.localList.right + 6, bottom: u.layout.localList.bottom + 8}
	rightPanel := linuxRect{left: u.layout.remotePath.left - 6, top: leftPanel.top, right: u.layout.remoteList.right + 6, bottom: leftPanel.bottom}
	if err := u.drawPanel(leftPanel); err != nil {
		return err
	}
	if err := u.drawPanel(rightPanel); err != nil {
		return err
	}
	localTitle := u.tr("section.local")
	remoteTitle := u.tr("section.remote")
	if err := u.x.text(u.layout.localPath.left+8, leftPanel.top+20, localTitle, premiumTheme.Text, premiumTheme.Panel); err != nil {
		return err
	}
	if err := u.x.text(u.layout.remotePath.left+8, leftPanel.top+20, remoteTitle, premiumTheme.Text, premiumTheme.Panel); err != nil {
		return err
	}
	if err := u.drawField(linuxFieldLocalPath, u.tr("column.local")); err != nil {
		return err
	}
	if err := u.drawField(linuxFieldRemotePath, u.tr("column.remote")); err != nil {
		return err
	}
	if err := u.renderFileFilterControls(); err != nil {
		return err
	}
	if err := u.renderItemRows(u.fileFilterListRect(false), u.localItems, u.selectedLocal); err != nil {
		return err
	}
	if err := u.renderItemRows(u.fileFilterListRect(true), u.remoteItems, u.selectedRemote); err != nil {
		return err
	}
	return nil
}

func linuxTransferFileName(job model.TransferJob) string {
	value := job.LocalPath
	if strings.EqualFold(job.Direction, "download") && strings.TrimSpace(job.RemotePath) != "" {
		value = job.RemotePath
	}
	value = strings.TrimRight(strings.ReplaceAll(strings.TrimSpace(value), "\\", "/"), "/")
	if value == "" {
		return "—"
	}
	if index := strings.LastIndex(value, "/"); index >= 0 && index+1 < len(value) {
		return value[index+1:]
	}
	return value
}

func linuxTransferSpeedLabel(job model.TransferJob) string {
	if job.BytesPerSecond <= 0 || (job.Status != "running" && job.Status != "done") {
		return "—"
	}
	return linuxHumanSize(int64(job.BytesPerSecond)) + "/s"
}

func linuxTransferETALabel(job model.TransferJob) string {
	if job.Status != "running" || job.ETASeconds <= 0 {
		return "—"
	}
	if job.ETASeconds < 60 {
		return fmt.Sprintf("%ds", job.ETASeconds)
	}
	minutes := job.ETASeconds / 60
	seconds := job.ETASeconds % 60
	return fmt.Sprintf("%d:%02d", minutes, seconds)
}

func (u *linuxDesktop) renderQueue() error {
	actions := u.linuxTransferActionState()
	title := u.tr("section.transfers")
	if count := len(u.transferJobs); count > 0 {
		title = fmt.Sprintf("%s (%d)", title, count)
	}
	if err := u.x.text(u.layout.queue.left+8, u.layout.pause.top+19, title, premiumTheme.Text, premiumTheme.Window); err != nil {
		return err
	}
	if err := u.drawButton(u.layout.clearQueue, u.tr("transfer.clear"), actions.Clear && !u.busy, false); err != nil {
		return err
	}
	if err := u.x.fillRect(u.layout.queue.left, u.layout.queue.top, u.layout.queue.right-u.layout.queue.left, u.layout.queue.bottom-u.layout.queue.top, premiumTheme.List); err != nil {
		return err
	}
	if err := u.x.strokeRect(u.layout.queue.left, u.layout.queue.top, u.layout.queue.right-u.layout.queue.left, u.layout.queue.bottom-u.layout.queue.top, premiumTheme.Border); err != nil {
		return err
	}
	width := u.layout.queue.right - u.layout.queue.left
	fileX := u.layout.queue.left + 8
	directionX := u.layout.queue.left + width*30/100
	progressX := u.layout.queue.left + width*43/100
	statusX := u.layout.queue.left + width*57/100
	speedX := u.layout.queue.left + width*78/100
	etaX := u.layout.queue.left + width*91/100
	headerY := u.layout.queue.top + 17
	headers := []struct {
		x     int
		label string
	}{
		{fileX, "File"},
		{directionX, u.tr("column.direction")},
		{progressX, "Progress"},
		{statusX, u.tr("column.status")},
		{speedX, "Speed"},
		{etaX, "ETA"},
	}
	for _, header := range headers {
		if err := u.x.text(header.x, headerY, header.label, premiumTheme.Muted, premiumTheme.List); err != nil {
			return err
		}
	}
	if len(u.transferJobs) == 0 {
		return u.x.text(fileX, u.layout.queue.top+42, u.tr("transfer.summary", 0, 0, 0), premiumTheme.Muted, premiumTheme.List)
	}
	rowH := 22
	maxRows := (u.layout.queue.bottom - u.layout.queue.top - 28) / rowH
	for i := 0; i < len(u.transferJobs) && i < maxRows; i++ {
		job := u.transferJobs[i]
		y := u.layout.queue.top + 28 + i*rowH
		bg := premiumTheme.List
		if i == u.selectedTransfer {
			bg = premiumTheme.Selection
			if err := u.x.fillRect(u.layout.queue.left+1, y-16, width-2, rowH, bg); err != nil {
				return err
			}
		}
		statusColor := premiumTheme.Muted
		switch job.Status {
		case "running":
			statusColor = premiumTheme.Warn
		case "done":
			statusColor = premiumTheme.Success
		case "failed", "cancelled":
			statusColor = premiumTheme.Danger
		}
		if err := u.x.text(fileX, y, linuxTrimForUI(linuxTransferFileName(job), max(12, (directionX-fileX-12)/7)), premiumTheme.Text, bg); err != nil {
			return err
		}
		if err := u.x.text(directionX, y, strings.ToUpper(job.Direction[:1])+strings.ToLower(job.Direction[1:]), premiumTheme.Text, bg); err != nil {
			return err
		}
		if err := u.x.text(progressX, y, fmt.Sprintf("%.0f%%", job.Progress*100), premiumTheme.Text, bg); err != nil {
			return err
		}
		if err := u.x.text(statusX, y, linuxTrimForUI(strings.ToUpper(job.Status[:1])+strings.ToLower(job.Status[1:]), max(8, (speedX-statusX-8)/7)), statusColor, bg); err != nil {
			return err
		}
		if err := u.x.text(speedX, y, linuxTransferSpeedLabel(job), premiumTheme.Muted, bg); err != nil {
			return err
		}
		if err := u.x.text(etaX, y, linuxTransferETALabel(job), premiumTheme.Muted, bg); err != nil {
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
	if u.busy || u.connected || len(u.profiles) == 0 {
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
	u.setStatus("Loaded profile: " + p.Name + ". Stored credentials remain protected and are never displayed.")
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

	password := u.password
	passphrase := u.passphrase
	if password != "" || passphrase != "" {
		now := time.Now()
		if u.confirmKind != "save-profile-credentials" || now.After(u.confirmUntil) {
			u.confirmKind = "save-profile-credentials"
			u.confirmUntil = now.Add(8 * time.Second)
			words := credentialConsentText(u.language)
			u.setStatus(words.Question + " " + words.Body + " Click Save again within 8 seconds to confirm.")
			return
		}
		u.confirmKind = ""
		u.confirmUntil = time.Time{}
	}

	name := strings.TrimSpace(u.host)
	if u.profileIndex >= 0 && u.profileIndex < len(u.profiles) && u.profiles[u.profileIndex].ID == u.selectedProfileID {
		if existingName := strings.TrimSpace(u.profiles[u.profileIndex].Name); existingName != "" {
			name = existingName
		}
	}
	if name == "" {
		name = "Ghost FTP profile"
	}

	u.password = ""
	u.passphrase = ""
	defer func() {
		password = ""
		passphrase = ""
	}()
	saved, err := u.engine.SaveProfile(model.ProfileInput{
		ID:             u.selectedProfileID,
		Name:           name,
		Protocol:       u.protocol,
		Host:           u.host,
		Port:           port,
		Username:       u.username,
		Password:       password,
		PrivateKeyPath: u.keyPath,
		Passphrase:     passphrase,
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
	if password != "" || passphrase != "" {
		u.setStatus("Profile and entered credentials saved in the protected local profile store.")
		return
	}
	u.setStatus("Profile saved. Existing protected credentials are preserved only when account identity is unchanged.")
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
	if u.handleRemoteEditResult(result) {
		return
	}
	if u.handleLinuxUpdateResult(result) {
		return
	}
	if result.err != nil {
		u.failLinuxWorkspaceReplay()
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
		u.clearLinuxRemoteFilterSource()
		u.workspaceBackHistory = nil
		u.workspaceForwardHistory = nil
		u.workspaceReplayActive = false
		u.lastFilePaneRemote = false
		u.setStatus(u.tr("disconnect.done"))
	case linuxActionLocalRefresh:
		u.completeLinuxWorkspaceNavigation(false, result.localBase)
		u.localCurrent = result.localBase
		u.acceptLinuxFileFilterSnapshot(false, result.localItems)
		u.selectedLocal = -1
		u.setStatus(fmt.Sprintf("Local files refreshed: %d items.", len(result.localItems)))
	case linuxActionRemoteRefresh:
		u.completeLinuxWorkspaceNavigation(true, result.localBase)
		u.remoteCurrent = result.localBase
		u.acceptLinuxFileFilterSnapshot(true, result.remoteItems)
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
	if u.handleLinuxMasterToolbarMouse(x, y) {
		return
	}
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
	if u.handleBookmarksHeaderMouse(x, y) {
		return
	}
	l := u.layout
	if u.handleQueuePriorityMouse(x, y) {
		return
	}
	if u.handleFileFilterMouse(x, y) {
		return
	}
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
	case u.fileFilterListRect(false).contains(x, y):
		u.lastFilePaneRemote = false
		u.selectedLocal = u.selectRow(u.fileFilterListRect(false), y, len(u.localItems))
	case u.fileFilterListRect(true).contains(x, y):
		u.lastFilePaneRemote = true
		u.selectedRemote = u.selectRow(u.fileFilterListRect(true), y, len(u.remoteItems))
	case l.pause.contains(x, y):
		u.pauseTransfersLinux()
	case l.resume.contains(x, y):
		u.resumeTransfersLinux()
	case l.cancelJob.contains(x, y):
		u.cancelSelectedTransferLinux()
	case l.retryJob.contains(x, y):
		u.retrySelectedTransferLinux()
	case l.clearQueue.contains(x, y):
		u.clearFinishedTransfersLinux()
	case l.queue.contains(x, y):
		index := (y - l.queue.top - 11) / 22
		if index >= 0 && index < len(u.transferJobs) {
			u.selectedTransfer = index
		}
	}
}

func linuxLegacyKeysymRune(sym uint32) (rune, bool) {
	// X11 Latin-2 keysyms still appear on Central European keyboard maps.
	switch sym {
	case 0x01a1:
		return 'Ą', true
	case 0x01a3:
		return 'Ł', true
	case 0x01a6:
		return 'Ś', true
	case 0x01a9:
		return 'Š', true
	case 0x01aa:
		return 'Ş', true
	case 0x01ab:
		return 'Ť', true
	case 0x01ac:
		return 'Ź', true
	case 0x01ae:
		return 'Ž', true
	case 0x01af:
		return 'Ż', true
	case 0x01b1:
		return 'ą', true
	case 0x01b3:
		return 'ł', true
	case 0x01b6:
		return 'ś', true
	case 0x01b9:
		return 'š', true
	case 0x01ba:
		return 'ş', true
	case 0x01bb:
		return 'ť', true
	case 0x01bc:
		return 'ź', true
	case 0x01be:
		return 'ž', true
	case 0x01bf:
		return 'ż', true
	case 0x01c3:
		return 'Ă', true
	case 0x01c6:
		return 'Ć', true
	case 0x01c8:
		return 'Č', true
	case 0x01ca:
		return 'Ę', true
	case 0x01cc:
		return 'Ě', true
	case 0x01cf:
		return 'Ď', true
	case 0x01d0:
		return 'Đ', true
	case 0x01d1:
		return 'Ń', true
	case 0x01d2:
		return 'Ň', true
	case 0x01d5:
		return 'Ő', true
	case 0x01d8:
		return 'Ř', true
	case 0x01d9:
		return 'Ů', true
	case 0x01db:
		return 'Ű', true
	case 0x01de:
		return 'Ţ', true
	case 0x01e3:
		return 'ă', true
	case 0x01e6:
		return 'ć', true
	case 0x01e8:
		return 'č', true
	case 0x01ea:
		return 'ę', true
	case 0x01ec:
		return 'ě', true
	case 0x01ef:
		return 'ď', true
	case 0x01f0:
		return 'đ', true
	case 0x01f1:
		return 'ń', true
	case 0x01f2:
		return 'ň', true
	case 0x01f5:
		return 'ő', true
	case 0x01f8:
		return 'ř', true
	case 0x01f9:
		return 'ů', true
	case 0x01fb:
		return 'ű', true
	case 0x01fe:
		return 'ţ', true
	case 0x02a9:
		return 'İ', true
	case 0x02ab:
		return 'Ğ', true
	case 0x02b9:
		return 'ı', true
	case 0x02bb:
		return 'ğ', true
	}

	// Russian/Ukrainian legacy keysyms. The standard Cyrillic letter blocks
	// are contiguous and ordered identically for lowercase and uppercase.
	if sym >= 0x06c0 && sym <= 0x06df {
		return []rune("юабцдефгхийклмнопярстужвьызшэщчъ")[sym-0x06c0], true
	}
	if sym >= 0x06e0 && sym <= 0x06ff {
		return []rune("ЮАБЦДЕФГХИЙКЛМНОПЯРСТУЖВЬЫЗШЭЩЧЪ")[sym-0x06e0], true
	}
	switch sym {
	case 0x06a3:
		return 'ё', true
	case 0x06a4:
		return 'є', true
	case 0x06a6:
		return 'і', true
	case 0x06a7:
		return 'ї', true
	case 0x06ad:
		return 'ґ', true
	case 0x06b3:
		return 'Ё', true
	case 0x06b4:
		return 'Є', true
	case 0x06b6:
		return 'І', true
	case 0x06b7:
		return 'Ї', true
	case 0x06bd:
		return 'Ґ', true
	}

	// Greek legacy keysyms use compact alphabet blocks plus explicit accent
	// symbols. Preserve final sigma and the accented/diaeresis variants.
	if sym >= 0x07c1 && sym <= 0x07d2 {
		return []rune("ΑΒΓΔΕΖΗΘΙΚΛΜΝΞΟΠΡΣ")[sym-0x07c1], true
	}
	if sym >= 0x07d4 && sym <= 0x07d9 {
		return []rune("ΤΥΦΧΨΩ")[sym-0x07d4], true
	}
	if sym >= 0x07e1 && sym <= 0x07f9 {
		return []rune("αβγδεζηθικλμνξοπρσςτυφχψω")[sym-0x07e1], true
	}
	switch sym {
	case 0x07a1:
		return 'Ά', true
	case 0x07a2:
		return 'Έ', true
	case 0x07a3:
		return 'Ή', true
	case 0x07a4:
		return 'Ί', true
	case 0x07a5:
		return 'Ϊ', true
	case 0x07a7:
		return 'Ό', true
	case 0x07a8:
		return 'Ύ', true
	case 0x07a9:
		return 'Ϋ', true
	case 0x07ab:
		return 'Ώ', true
	case 0x07b1:
		return 'ά', true
	case 0x07b2:
		return 'έ', true
	case 0x07b3:
		return 'ή', true
	case 0x07b4:
		return 'ί', true
	case 0x07b5:
		return 'ϊ', true
	case 0x07b6:
		return 'ΐ', true
	case 0x07b7:
		return 'ό', true
	case 0x07b8:
		return 'ύ', true
	case 0x07b9:
		return 'ϋ', true
	case 0x07ba:
		return 'ΰ', true
	case 0x07bb:
		return 'ώ', true
	}
	return 0, false
}

func linuxKeysymText(sym uint32) (string, bool) {
	var r rune
	switch {
	case sym >= 0x20 && sym <= 0x7e:
		r = rune(sym)
	case sym >= 0x00a0 && sym <= 0x00ff:
		// Legacy X11 Latin-1 keysyms are their Unicode code points.
		r = rune(sym)
	case sym >= 0x01000100 && sym <= 0x0110ffff:
		// Modern X11 Unicode keysyms use 0x01000000 | codepoint.
		r = rune(sym & 0x00ffffff)
	default:
		legacy, ok := linuxLegacyKeysymRune(sym)
		if !ok {
			return "", false
		}
		r = legacy
	}
	if !utf8.ValidRune(r) || !unicode.IsGraphic(r) {
		return "", false
	}
	return string(r), true
}

func (u *linuxDesktop) handleKey(keycode byte, state uint16) bool {
	sym := u.x.keysym(keycode, state)
	if u.handleLinuxInfoOverlayKey(sym) {
		return true
	}
	if u.handleRemoteEditKey(sym, state) {
		return true
	}
	if u.handleSettingsKey(sym) {
		return true
	}
	if u.handlePromptKey(sym) {
		return true
	}
	if state&x11ControlMask != 0 {
		if sym == 'f' || sym == 'F' {
			u.openFileFilterPrompt(false)
			return true
		}
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
		if value != "" {
			u.setFieldValue(u.focus, linuxDeleteLastRune(value))
		}
		return true
	}
	if u.focus == linuxFieldProtocol {
		return true
	}
	if text, ok := linuxKeysymText(sym); ok {
		if u.focus == linuxFieldPort && (len(text) != 1 || text[0] < '0' || text[0] > '9') {
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
	if u.linuxInfoOverlayOpen() {
		return u.renderLinuxInfoOverlay()
	}
	if u.remoteEditorOpen() {
		return u.renderRemoteEditorOverlay()
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
	if u.handleLinuxInfoOverlayMouse(x, y) {
		return true
	}
	if u.handleRemoteEditorMouse(x, y) {
		return true
	}
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
	defer linuxRemoteEditStates.Delete(u)
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
