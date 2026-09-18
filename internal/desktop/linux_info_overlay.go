//go:build linux

package desktop

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/bren-wp/Ghost-FTP/internal/brand"
	"github.com/bren-wp/Ghost-FTP/internal/i18n"
)

type linuxInfoOverlayKind int

const (
	linuxInfoOverlayNone linuxInfoOverlayKind = iota
	linuxInfoOverlayConnection
	linuxInfoOverlayAbout
	linuxInfoOverlayMore
)

type linuxInfoOverlayLayout struct {
	panel linuxRect
	close linuxRect
}

type linuxMoreMenuLayout struct {
	panel   linuxRect
	actions [15]linuxRect
	close   linuxRect
}

func buildLinuxMoreMenuLayout(width, height int) linuxMoreMenuLayout {
	panelWidth := min(760, width-80)
	if panelWidth < 620 {
		panelWidth = 620
	}
	panelHeight := min(500, height-80)
	if panelHeight < 430 {
		panelHeight = 430
	}
	left := (width - panelWidth) / 2
	top := (height - panelHeight) / 2
	panel := linuxRectWH(left, top, panelWidth, panelHeight)
	layout := linuxMoreMenuLayout{panel: panel}
	gap := 8
	columnGap := 10
	buttonH := 34
	buttonW := (panelWidth - 40 - columnGap) / 2
	startY := top + 66
	for index := range layout.actions {
		row := index / 2
		column := index % 2
		x := left + 20 + column*(buttonW+columnGap)
		y := startY + row*(buttonH+gap)
		layout.actions[index] = linuxRectWH(x, y, buttonW, buttonH)
	}
	layout.close = linuxRectWH(panel.right-124, panel.bottom-48, 104, 30)
	return layout
}

func buildLinuxInfoOverlayLayout(width, height int, kind linuxInfoOverlayKind) linuxInfoOverlayLayout {
	panelWidth := min(760, width-80)
	if panelWidth < 560 {
		panelWidth = 560
	}
	panelHeight := 330
	if kind == linuxInfoOverlayAbout {
		panelHeight = 360
	}
	if panelHeight > height-80 {
		panelHeight = height - 80
	}
	if panelHeight < 260 {
		panelHeight = 260
	}
	left := (width - panelWidth) / 2
	top := (height - panelHeight) / 2
	panel := linuxRectWH(left, top, panelWidth, panelHeight)
	return linuxInfoOverlayLayout{
		panel: panel,
		close: linuxRectWH(panel.right-124, panel.bottom-52, 104, 32),
	}
}

func linuxWrapForUI(value string, maxRunes int) []string {
	value = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(value, "\r", " "), "\n", " "))
	if value == "" {
		return nil
	}
	if maxRunes < 8 {
		maxRunes = 8
	}
	words := strings.Fields(value)
	lines := make([]string, 0, 4)
	current := ""
	for _, word := range words {
		wordRunes := []rune(word)
		if len(wordRunes) > maxRunes {
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
			for len(wordRunes) > maxRunes {
				lines = append(lines, string(wordRunes[:maxRunes]))
				wordRunes = wordRunes[maxRunes:]
			}
			if len(wordRunes) == 0 {
				continue
			}
			word = string(wordRunes)
		}
		candidate := word
		if current != "" {
			candidate = current + " " + word
		}
		if utf8.RuneCountInString(candidate) <= maxRunes {
			current = candidate
			continue
		}
		if current != "" {
			lines = append(lines, current)
		}
		current = word
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func diagnosticsWordsForLanguage(language string) diagnosticsWordsSet {
	language = i18n.Normalize(language)
	words, ok := diagnosticsWords[language]
	if !ok {
		words = diagnosticsWords["en"]
	}
	return words
}

func linuxConnectionInfoLines(u *linuxDesktop) []string {
	if u == nil {
		return nil
	}
	words := diagnosticsWordsForLanguage(u.language)
	state := words.RemoteOffline
	if u.connected {
		state = u.tr("badge.connected")
	}
	protocol := strings.ToUpper(strings.TrimSpace(u.protocol))
	if protocol == "" {
		protocol = "FTP / FTPS / SFTP"
	}
	lines := []string{
		protocol + " | " + state,
	}
	lines = append(
		lines,
		navigationLabelsForLanguage(u.language).TransferQueue+": "+fmt.Sprintf("%d", linuxMasterTransferBadge(u.transferJobs)),
		words.PrivacyBody,
	)
	return lines
}

func linuxAboutInfoLines(u *linuxDesktop) []string {
	if u == nil {
		return nil
	}
	words := diagnosticsWordsForLanguage(u.language)
	return []string{
		brand.ProductName + " " + u.version,
		"FTP / FTPS / SFTP",
		brand.Website,
		words.PrivacyBody,
	}
}

func (u *linuxDesktop) linuxInfoOverlayOpen() bool {
	return u != nil && u.infoOverlay != linuxInfoOverlayNone
}

func (u *linuxDesktop) openLinuxInfoOverlay(kind linuxInfoOverlayKind) {
	if u == nil || kind == linuxInfoOverlayNone || u.busy || u.settingsOpen ||
		u.promptKind != linuxPromptNone || u.pendingFingerprint != "" || u.remoteEditorOpen() {
		return
	}
	u.infoOverlay = kind
}

func (u *linuxDesktop) closeLinuxInfoOverlay() {
	if u == nil {
		return
	}
	u.infoOverlay = linuxInfoOverlayNone
}

func (u *linuxDesktop) linuxInfoOverlayTitleAndHeading() (string, string) {
	switch u.infoOverlay {
	case linuxInfoOverlayConnection:
		title := navigationLabelsForLanguage(u.language).Diagnostics
		return title, brand.ProductName + " " + u.version
	case linuxInfoOverlayAbout:
		return u.tr("about.title"), u.tr("about.heading")
	case linuxInfoOverlayMore:
		return "More", "File and connection tools"
	default:
		return "", ""
	}
}

func (u *linuxDesktop) linuxInfoOverlayLines() []string {
	switch u.infoOverlay {
	case linuxInfoOverlayConnection:
		return linuxConnectionInfoLines(u)
	case linuxInfoOverlayAbout:
		return linuxAboutInfoLines(u)
	default:
		return nil
	}
}

func (u *linuxDesktop) linuxMoreMenuLabels() []string {
	search := recursiveSearchWordsForLanguage(u.language).Search
	local := u.tr("section.local")
	remote := u.tr("section.remote")
	return []string{
		local + ": " + u.tr("common.up"),
		remote + ": " + u.tr("common.up"),
		local + ": " + fileFilterWordsForLanguage(u.language).Cue,
		remote + ": " + fileFilterWordsForLanguage(u.language).Cue,
		local + ": " + search,
		remote + ": " + search,
		local + ": " + u.tr("common.rename"),
		remote + ": " + u.tr("common.rename"),
		local + ": " + u.tr("common.delete"),
		remote + ": " + u.tr("common.delete"),
		remote + ": " + u.tr("common.permissions"),
		"Remote Edit",
		directoryCompareWordsForLanguage(u.language).Compare,
		navigationLabelsForLanguage(u.language).Diagnostics,
		u.tr("common.about"),
	}
}

func (u *linuxDesktop) renderLinuxMoreMenu() error {
	layout := buildLinuxMoreMenuLayout(u.width, u.height)
	if err := u.drawPanel(layout.panel); err != nil {
		return err
	}
	if err := u.x.text(layout.panel.left+20, layout.panel.top+30, "MORE", premiumTheme.Text, premiumTheme.Panel); err != nil {
		return err
	}
	if err := u.x.text(layout.panel.left+20, layout.panel.top+52, "File, folder and connection tools", premiumTheme.Muted, premiumTheme.Panel); err != nil {
		return err
	}
	labels := u.linuxMoreMenuLabels()
	for index, rect := range layout.actions {
		enabled := !u.busy
		switch index {
		case 1, 3, 5:
			enabled = enabled && u.connected
		case 6, 8:
			enabled = enabled && u.selectedLocal >= 0
		case 7, 9, 10, 11:
			enabled = enabled && u.connected && u.selectedRemote >= 0
		case 12:
			enabled = enabled && u.connected
		}
		if err := u.drawButtonWithLimit(rect, labels[index], enabled, index == 12, linuxButtonLabelLimit(rect)); err != nil {
			return err
		}
	}
	return u.drawButton(layout.close, bookmarkWordsForLanguage(u.language).Close, true, false)
}

func (u *linuxDesktop) renderLinuxInfoOverlay() error {
	if !u.linuxInfoOverlayOpen() {
		return nil
	}
	if u.infoOverlay == linuxInfoOverlayMore {
		return u.renderLinuxMoreMenu()
	}
	layout := buildLinuxInfoOverlayLayout(u.width, u.height, u.infoOverlay)
	if err := u.drawPanel(layout.panel); err != nil {
		return err
	}
	title, heading := u.linuxInfoOverlayTitleAndHeading()
	if err := u.x.text(layout.panel.left+20, layout.panel.top+30, strings.ToUpper(linuxTrimForUI(title, 72)), premiumTheme.Text, premiumTheme.Panel); err != nil {
		return err
	}

	textWidth := max(24, (layout.panel.right-layout.panel.left-40)/7)
	y := layout.panel.top + 62
	for _, line := range linuxWrapForUI(heading, textWidth) {
		if err := u.x.text(layout.panel.left+20, y, line, premiumTheme.Text, premiumTheme.Panel); err != nil {
			return err
		}
		y += 21
	}
	y += 12

	lines := u.linuxInfoOverlayLines()
	for index, raw := range lines {
		color := premiumTheme.Text
		if index == len(lines)-1 {
			color = premiumTheme.Muted
		}
		for _, line := range linuxWrapForUI(raw, textWidth) {
			if y > layout.close.top-26 {
				break
			}
			if err := u.x.text(layout.panel.left+20, y, line, color, premiumTheme.Panel); err != nil {
				return err
			}
			y += 21
		}
		y += 5
	}

	if err := u.x.text(layout.panel.left+20, layout.panel.bottom-31, "Esc", premiumTheme.Muted, premiumTheme.Panel); err != nil {
		return err
	}
	return u.drawButton(layout.close, bookmarkWordsForLanguage(u.language).Close, true, true)
}

func (u *linuxDesktop) handleLinuxMoreMenuMouse(x, y int) bool {
	layout := buildLinuxMoreMenuLayout(u.width, u.height)
	if layout.close.contains(x, y) {
		u.closeLinuxInfoOverlay()
		return true
	}
	selected := -1
	for index, rect := range layout.actions {
		if rect.contains(x, y) {
			selected = index
			break
		}
	}
	if selected < 0 {
		return true
	}
	u.closeLinuxInfoOverlay()
	switch selected {
	case 0:
		u.lastFilePaneRemote = false
		u.refreshLocal(filepath.Dir(u.localCurrent))
	case 1:
		if u.connected {
			u.lastFilePaneRemote = true
			u.refreshRemote(terminalRemotePath(u.remoteCurrent, ".."))
		}
	case 2:
		u.openFileFilterPrompt(false)
	case 3:
		if u.connected {
			u.openFileFilterPrompt(true)
		}
	case 4:
		u.openRecursiveSearchPrompt(false)
	case 5:
		if u.connected {
			u.openRecursiveSearchPrompt(true)
		}
	case 6:
		u.openSelectedLocalRename()
	case 7:
		u.openSelectedRemoteRename()
	case 8:
		u.deleteSelectedLocal()
	case 9:
		u.deleteSelectedRemote()
	case 10:
		u.openSelectedRemoteChmod()
	case 11:
		u.openSelectedRemoteEditor()
	case 12:
		u.startDirectoryComparisonLinux()
	case 13:
		u.openLinuxInfoOverlay(linuxInfoOverlayConnection)
	case 14:
		u.openLinuxInfoOverlay(linuxInfoOverlayAbout)
	}
	return true
}

func (u *linuxDesktop) handleLinuxInfoOverlayMouse(x, y int) bool {
	if !u.linuxInfoOverlayOpen() {
		return false
	}
	if u.infoOverlay == linuxInfoOverlayMore {
		return u.handleLinuxMoreMenuMouse(x, y)
	}
	layout := buildLinuxInfoOverlayLayout(u.width, u.height, u.infoOverlay)
	if layout.close.contains(x, y) {
		u.closeLinuxInfoOverlay()
	}
	return true
}

func (u *linuxDesktop) handleLinuxInfoOverlayKey(sym uint32) bool {
	if !u.linuxInfoOverlayOpen() {
		return false
	}
	switch sym {
	case x11KeyEscape:
		u.closeLinuxInfoOverlay()
	case x11KeyReturn:
		if u.infoOverlay != linuxInfoOverlayMore {
			u.closeLinuxInfoOverlay()
		}
	}
	return true
}
