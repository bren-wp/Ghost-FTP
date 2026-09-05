//go:build linux

package desktop

import (
	"fmt"

	"github.com/bren-wp/Ghost-FTP/internal/config"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/usererror"
)

type linuxSettingsRects struct {
	parallelMinus, parallelPlus linuxRect
	retriesMinus, retriesPlus   linuxRect
	delayMinus, delayPlus       linuxRect
	timeoutMinus, timeoutPlus   linuxRect
	conflict, confirmDelete     linuxRect
	save, close                 linuxRect
}

func (u *linuxDesktop) openSettings() {
	if u.busy || u.promptKind != linuxPromptNone || u.pendingFingerprint != "" {
		return
	}
	settings, err := u.engine.Settings()
	if err != nil {
		u.setStatus(usererror.MessageFor(u.language, err, "Settings could not be loaded."))
		return
	}
	u.settingsDraft = settings
	u.settingsOpen = true
}

func (u *linuxDesktop) closeSettings() {
	u.settingsOpen = false
	u.settingsRects = linuxSettingsRects{}
}

func clampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func nextConflictPolicy(current string) string {
	switch current {
	case model.ConflictPolicySkip:
		return model.ConflictPolicyReplace
	case model.ConflictPolicyReplace:
		return model.ConflictPolicyReplaceBackup
	default:
		return model.ConflictPolicySkip
	}
}

func conflictPolicyLabel(policy string) string {
	switch policy {
	case model.ConflictPolicySkip:
		return "Skip existing"
	case model.ConflictPolicyReplace:
		return "Replace"
	default:
		return "Replace + backup"
	}
}

func (u *linuxDesktop) saveSettings() {
	saved, err := u.engine.SetSettings(u.settingsDraft)
	if err != nil {
		u.setStatus(usererror.MessageFor(u.language, err, "Settings were not saved."))
		return
	}
	u.settingsDraft = saved
	u.language = saved.Language
	u.closeSettings()
	u.setStatus("Settings saved.")
}

func (u *linuxDesktop) settingsStep(rectMinus, rectPlus linuxRect, x, y int, value *int, low, high, step int) bool {
	if rectMinus.contains(x, y) {
		*value = clampInt(*value-step, low, high)
		return true
	}
	if rectPlus.contains(x, y) {
		*value = clampInt(*value+step, low, high)
		return true
	}
	return false
}

func (u *linuxDesktop) handleSettingsMouse(x, y int) bool {
	if !u.settingsOpen {
		return false
	}
	r := u.settingsRects
	if u.settingsStep(r.parallelMinus, r.parallelPlus, x, y, &u.settingsDraft.Parallelism, config.MinParallelism, config.MaxParallelism, 1) {
		return true
	}
	if u.settingsStep(r.retriesMinus, r.retriesPlus, x, y, &u.settingsDraft.AutoRetryCount, config.MinAutoRetryCount, config.MaxAutoRetryCount, 1) {
		return true
	}
	if u.settingsStep(r.delayMinus, r.delayPlus, x, y, &u.settingsDraft.RetryDelaySeconds, config.MinRetryDelaySeconds, config.MaxRetryDelaySeconds, 1) {
		return true
	}
	if u.settingsStep(r.timeoutMinus, r.timeoutPlus, x, y, &u.settingsDraft.ConnectionTimeoutSeconds, config.MinConnectionTimeoutSeconds, config.MaxConnectionTimeoutSeconds, 5) {
		return true
	}
	if r.conflict.contains(x, y) {
		u.settingsDraft.ConflictPolicy = nextConflictPolicy(u.settingsDraft.ConflictPolicy)
		return true
	}
	if r.confirmDelete.contains(x, y) {
		u.settingsDraft.ConfirmDelete = !u.settingsDraft.ConfirmDelete
		return true
	}
	if r.save.contains(x, y) {
		u.saveSettings()
		return true
	}
	if r.close.contains(x, y) {
		u.closeSettings()
		u.setStatus("Settings changes discarded.")
		return true
	}
	return true
}

func (u *linuxDesktop) handleSettingsKey(sym uint32) bool {
	if !u.settingsOpen {
		return false
	}
	switch sym {
	case x11KeyEscape:
		u.closeSettings()
		u.setStatus("Settings changes discarded.")
	case x11KeyReturn:
		u.saveSettings()
	}
	return true
}

func (u *linuxDesktop) drawSettingStepper(label string, value int, top int, minus *linuxRect, plus *linuxRect) error {
	left := (u.width - min(700, u.width-100)) / 2
	width := min(700, u.width-100)
	if err := u.x.text(left+24, top+20, label, premiumTheme.Text, premiumTheme.Panel); err != nil {
		return err
	}
	valueRect := linuxRectWH(left+width-210, top, 84, 30)
	*minus = linuxRectWH(left+width-120, top, 48, 30)
	*plus = linuxRectWH(left+width-64, top, 48, 30)
	if err := u.x.fillRect(valueRect.left, valueRect.top, 84, 30, premiumTheme.List); err != nil {
		return err
	}
	if err := u.x.strokeRect(valueRect.left, valueRect.top, 84, 30, premiumTheme.Border); err != nil {
		return err
	}
	if err := u.x.text(valueRect.left+12, valueRect.top+20, fmt.Sprintf("%d", value), premiumTheme.Text, premiumTheme.List); err != nil {
		return err
	}
	if err := u.drawButton(*minus, "-", true, false); err != nil {
		return err
	}
	return u.drawButton(*plus, "+", true, false)
}

func (u *linuxDesktop) renderSettingsOverlay() error {
	if !u.settingsOpen {
		return nil
	}
	width := min(700, u.width-100)
	height := 430
	left := (u.width - width) / 2
	top := (u.height - height) / 2
	panel := linuxRectWH(left, top, width, height)
	if err := u.drawPanel(panel); err != nil {
		return err
	}
	if err := u.x.text(left+24, top+32, "SETTINGS", premiumTheme.Text, premiumTheme.Panel); err != nil {
		return err
	}
	if err := u.x.text(left+24, top+54, "Transfer, overwrite and safety behavior", premiumTheme.Muted, premiumTheme.Panel); err != nil {
		return err
	}

	row := top + 78
	if err := u.drawSettingStepper("Parallel transfers", u.settingsDraft.Parallelism, row, &u.settingsRects.parallelMinus, &u.settingsRects.parallelPlus); err != nil {
		return err
	}
	row += 48
	if err := u.drawSettingStepper("Automatic retries", u.settingsDraft.AutoRetryCount, row, &u.settingsRects.retriesMinus, &u.settingsRects.retriesPlus); err != nil {
		return err
	}
	row += 48
	if err := u.drawSettingStepper("Retry delay (seconds)", u.settingsDraft.RetryDelaySeconds, row, &u.settingsRects.delayMinus, &u.settingsRects.delayPlus); err != nil {
		return err
	}
	row += 48
	if err := u.drawSettingStepper("Connection timeout (seconds)", u.settingsDraft.ConnectionTimeoutSeconds, row, &u.settingsRects.timeoutMinus, &u.settingsRects.timeoutPlus); err != nil {
		return err
	}
	row += 52
	if err := u.x.text(left+24, row+20, "Existing destination files", premiumTheme.Text, premiumTheme.Panel); err != nil {
		return err
	}
	u.settingsRects.conflict = linuxRectWH(left+width-246, row, 230, 30)
	if err := u.drawButton(u.settingsRects.conflict, conflictPolicyLabel(u.settingsDraft.ConflictPolicy), true, false); err != nil {
		return err
	}
	row += 46
	if err := u.x.text(left+24, row+20, "Confirm destructive deletes", premiumTheme.Text, premiumTheme.Panel); err != nil {
		return err
	}
	u.settingsRects.confirmDelete = linuxRectWH(left+width-156, row, 140, 30)
	confirm := "Enabled"
	if !u.settingsDraft.ConfirmDelete {
		confirm = "Disabled"
	}
	if err := u.drawButton(u.settingsRects.confirmDelete, confirm, true, u.settingsDraft.ConfirmDelete); err != nil {
		return err
	}

	u.settingsRects.save = linuxRectWH(left+width-226, top+height-48, 98, 30)
	u.settingsRects.close = linuxRectWH(left+width-118, top+height-48, 98, 30)
	if err := u.drawButton(u.settingsRects.save, "Save", true, true); err != nil {
		return err
	}
	return u.drawButton(u.settingsRects.close, "Cancel", true, false)
}
