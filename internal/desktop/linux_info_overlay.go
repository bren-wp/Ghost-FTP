//go:build linux

package desktop

import (
	"fmt"
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
)

type linuxInfoOverlayLayout struct {
	panel linuxRect
	close linuxRect
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

func (u *linuxDesktop) renderLinuxInfoOverlay() error {
	if !u.linuxInfoOverlayOpen() {
		return nil
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

func (u *linuxDesktop) handleLinuxInfoOverlayMouse(x, y int) bool {
	if !u.linuxInfoOverlayOpen() {
		return false
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
	case x11KeyEscape, x11KeyReturn:
		u.closeLinuxInfoOverlay()
	}
	return true
}
