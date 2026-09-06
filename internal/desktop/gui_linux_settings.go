//go:build linux

package desktop

import (
	"fmt"

	"github.com/bren-wp/Ghost-FTP/internal/config"
	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/usererror"
)

type linuxSettingsRects struct {
	language, appearance        linuxRect
	parallelMinus, parallelPlus linuxRect
	retriesMinus, retriesPlus   linuxRect
	delayMinus, delayPlus       linuxRect
	timeoutMinus, timeoutPlus   linuxRect
	conflict, confirmDelete     linuxRect
	save, close                 linuxRect
}

func (u *linuxDesktop) tr(key string, args ...any) string {
	return i18n.T(u.language, key, args...)
}

func (u *linuxDesktop) draftTr(key string, args ...any) string {
	language := u.language
	if u.settingsOpen {
		language = i18n.Normalize(u.settingsDraft.Language)
	}
	return i18n.T(language, key, args...)
}

func (u *linuxDesktop) openSettings() {
	if u.busy || u.promptKind != linuxPromptNone || u.pendingFingerprint != "" {
		return
	}
	settings, err := u.engine.Settings()
	if err != nil {
		u.setStatus(usererror.MessageFor(u.language, err, u.tr("settings.load_failed")))
		return
	}
	settings.Language = i18n.Normalize(settings.Language)
	if settings.Appearance == "" {
		settings.Appearance = model.AppearanceDark
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

func (u *linuxDesktop) conflictPolicyLabel(policy string) string {
	switch policy {
	case model.ConflictPolicySkip:
		return u.draftTr("settings.skip_existing")
	case model.ConflictPolicyReplace:
		return u.draftTr("settings.overwrite")
	default:
		return u.draftTr("settings.overwrite") + " + " + u.draftTr("settings.backup_title")
	}
}

func nextLanguage(code string) string {
	languages := i18n.Languages()
	if len(languages) == 0 {
		return i18n.DefaultLanguage
	}
	code = i18n.Normalize(code)
	for index, language := range languages {
		if language.Code == code {
			return languages[(index+1)%len(languages)].Code
		}
	}
	return i18n.DefaultLanguage
}

func nextAppearance(current string) string {
	if current == model.AppearanceLight {
		return model.AppearanceDark
	}
	return model.AppearanceLight
}

func (u *linuxDesktop) appearanceLabel() string {
	words := appearanceText(u.settingsDraft.Language)
	if u.settingsDraft.Appearance == model.AppearanceLight {
		return words.Light
	}
	return words.Dark
}

func (u *linuxDesktop) saveSettings() {
	u.settingsDraft.Language = i18n.Normalize(u.settingsDraft.Language)
	saved, err := u.engine.SetSettings(u.settingsDraft)
	if err != nil {
		u.setStatus(usererror.MessageFor(u.language, err, u.tr("settings.save_failed_body")))
		return
	}
	u.settingsDraft = saved
	u.language = i18n.Normalize(saved.Language)
	u.closeSettings()
	status := u.tr("settings.saved", saved.Parallelism, saved.ConnectionTimeoutSeconds, linuxRetrySummary(u, saved), linuxConflictSummary(u, saved))
	if isDarkAppearance(saved.Appearance) != activeThemeIsDark() {
		status += " · " + appearanceText(saved.Language).Hint
	}
	u.setStatus(status)
}

func linuxRetrySummary(u *linuxDesktop, settings model.Settings) string {
	if settings.AutoRetryCount <= 0 {
		return u.tr("settings.retry_none")
	}
	return u.tr("settings.retry_count", settings.AutoRetryCount)
}

func linuxConflictSummary(u *linuxDesktop, settings model.Settings) string {
	if settings.ConflictPolicy == model.ConflictPolicySkip {
		return u.tr("settings.skip_existing")
	}
	return u.tr("settings.overwrite")
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
	if r.language.contains(x, y) {
		u.settingsDraft.Language = nextLanguage(u.settingsDraft.Language)
		return true
	}
	if r.appearance.contains(x, y) {
		u.settingsDraft.Appearance = nextAppearance(u.settingsDraft.Appearance)
		return true
	}
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
		u.setStatus(u.tr("status.ready"))
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
		u.setStatus(u.tr("status.ready"))
	case x11KeyReturn:
		u.saveSettings()
	}
	return true
}

func (u *linuxDesktop) drawSettingStepper(label string, value int, top int, minus *linuxRect, plus *linuxRect) error {
	left := (u.width - min(700, u.width-100)) / 2
	width := min(700, u.width-100)
	if err := u.x.text(left+24, top+20, linuxTrimForUI(label, 48), premiumTheme.Text, premiumTheme.Panel); err != nil {
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
	if err := u.drawButton(*minus, "−", true, false); err != nil {
		return err
	}
	return u.drawButton(*plus, "+", true, false)
}

func (u *linuxDesktop) renderSettingsOverlay() error {
	if !u.settingsOpen {
		return nil
	}
	width := min(700, u.width-100)
	height := 538
	left := (u.width - width) / 2
	top := (u.height - height) / 2
	panel := linuxRectWH(left, top, width, height)
	if err := u.drawPanel(panel); err != nil {
		return err
	}
	if err := u.x.text(left+24, top+32, linuxTrimForUI(u.draftTr("settings.title"), 54), premiumTheme.Text, premiumTheme.Panel); err != nil {
		return err
	}
	if err := u.x.text(left+24, top+54, linuxTrimForUI(u.draftTr("settings.confirm_delete_body"), 82), premiumTheme.Muted, premiumTheme.Panel); err != nil {
		return err
	}

	row := top + 76
	language := i18n.LanguageByCode(u.settingsDraft.Language)
	if err := u.x.text(left+24, row+20, "Aa", premiumTheme.Text, premiumTheme.Panel); err != nil {
		return err
	}
	u.settingsRects.language = linuxRectWH(left+width-246, row, 230, 30)
	languageLabel := language.NativeName + " (" + language.Code + ")"
	if err := u.drawButton(u.settingsRects.language, linuxTrimForUI(languageLabel, 30), true, false); err != nil {
		return err
	}

	row += 43
	appearance := appearanceText(u.settingsDraft.Language)
	if err := u.x.text(left+24, row+20, linuxTrimForUI(appearance.Title, 48), premiumTheme.Text, premiumTheme.Panel); err != nil {
		return err
	}
	u.settingsRects.appearance = linuxRectWH(left+width-246, row, 230, 30)
	if err := u.drawButton(u.settingsRects.appearance, linuxTrimForUI(u.appearanceLabel(), 30), true, false); err != nil {
		return err
	}
	if err := u.x.text(left+24, row+40, linuxTrimForUI(appearance.Hint, 88), premiumTheme.Muted, premiumTheme.Panel); err != nil {
		return err
	}

	row += 56
	if err := u.drawSettingStepper(u.draftTr("settings.parallel"), u.settingsDraft.Parallelism, row, &u.settingsRects.parallelMinus, &u.settingsRects.parallelPlus); err != nil {
		return err
	}
	row += 45
	if err := u.drawSettingStepper(u.draftTr("settings.retries"), u.settingsDraft.AutoRetryCount, row, &u.settingsRects.retriesMinus, &u.settingsRects.retriesPlus); err != nil {
		return err
	}
	row += 45
	if err := u.drawSettingStepper(u.draftTr("settings.retry_delay"), u.settingsDraft.RetryDelaySeconds, row, &u.settingsRects.delayMinus, &u.settingsRects.delayPlus); err != nil {
		return err
	}
	row += 45
	if err := u.drawSettingStepper(u.draftTr("settings.timeout"), u.settingsDraft.ConnectionTimeoutSeconds, row, &u.settingsRects.timeoutMinus, &u.settingsRects.timeoutPlus); err != nil {
		return err
	}
	row += 48
	if err := u.x.text(left+24, row+20, linuxTrimForUI(u.draftTr("settings.skip_title"), 48), premiumTheme.Text, premiumTheme.Panel); err != nil {
		return err
	}
	u.settingsRects.conflict = linuxRectWH(left+width-246, row, 230, 30)
	if err := u.drawButton(u.settingsRects.conflict, linuxTrimForUI(u.conflictPolicyLabel(u.settingsDraft.ConflictPolicy), 31), true, false); err != nil {
		return err
	}
	row += 44
	if err := u.x.text(left+24, row+20, linuxTrimForUI(u.draftTr("settings.confirm_delete_title"), 48), premiumTheme.Text, premiumTheme.Panel); err != nil {
		return err
	}
	u.settingsRects.confirmDelete = linuxRectWH(left+width-156, row, 140, 30)
	confirm := "○"
	if u.settingsDraft.ConfirmDelete {
		confirm = "✓"
	}
	if err := u.drawButton(u.settingsRects.confirmDelete, confirm, true, u.settingsDraft.ConfirmDelete); err != nil {
		return err
	}

	u.settingsRects.save = linuxRectWH(left+width-226, top+height-48, 98, 30)
	u.settingsRects.close = linuxRectWH(left+width-118, top+height-48, 98, 30)
	if err := u.drawButton(u.settingsRects.save, "OK", true, true); err != nil {
		return err
	}
	return u.drawButton(u.settingsRects.close, u.draftTr("common.cancel"), true, false)
}
