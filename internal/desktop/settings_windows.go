//go:build windows

package desktop

import (
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/brand"
	"github.com/bren-wp/Ghost-FTP/internal/config"
	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

func (a *app) loadSettings() {
	a.goSafe(func() {
		settings, err := a.engine.Settings()
		a.dispatch(func() {
			if err != nil {
				a.setStatus(a.userMessage(err, "settings.load_failed"))
				return
			}
			previousLanguage := a.languageCode()
			a.settings = settings
			// The controls were already created in the canonical startup locale.
			// Reapplying the same locale used to refill every list, relayout every
			// control and erase the whole client immediately after ShowWindow,
			// producing a visible startup flash. Only rebuild localized UI when the
			// persisted locale actually differs.
			if a.languageCode() != previousLanguage {
				a.applyLanguage(settings.Language)
			}
			a.updateActionControls()
		})
	})
}

func (a *app) setSettingsControlsEnabled(enabled bool) {
	value := uintptr(0)
	if enabled {
		value = 1
	}
	for _, hwnd := range []uintptr{a.languageCombo, a.settingsBtn} {
		if hwnd != 0 {
			enableWindow.Call(hwnd, value)
		}
	}
}

func conflictPolicyIndex(settings model.Settings) int {
	switch settings.ConflictPolicy {
	case model.ConflictPolicySkip:
		return 0
	case model.ConflictPolicyReplace:
		return 1
	case model.ConflictPolicyReplaceBackup:
		return 2
	}
	if settings.SkipExisting {
		return 0
	}
	if settings.BackupBeforeOverwrite {
		return 2
	}
	return 1
}

func applyConflictPolicySelection(settings *model.Settings, index int) {
	if settings == nil {
		return
	}
	switch index {
	case 0:
		settings.ConflictPolicy = model.ConflictPolicySkip
		settings.SkipExisting = true
		// Legacy consumers historically saw backup enabled together with skip.
		// The backup flag has no effect because skip performs no overwrite.
		settings.BackupBeforeOverwrite = true
	case 1:
		settings.ConflictPolicy = model.ConflictPolicyReplace
		settings.SkipExisting = false
		settings.BackupBeforeOverwrite = false
	default:
		settings.ConflictPolicy = model.ConflictPolicyReplaceBackup
		settings.SkipExisting = false
		settings.BackupBeforeOverwrite = true
	}
}

func normalizeSettingsForPrompt(settings model.Settings) model.Settings {
	defaults := config.DefaultSettings()
	if settings.Appearance == "" {
		settings.Appearance = defaults.Appearance
	}
	if settings.Parallelism < config.MinParallelism || settings.Parallelism > config.MaxParallelism {
		settings.Parallelism = defaults.Parallelism
	}
	if settings.AutoRetryCount < config.MinAutoRetryCount || settings.AutoRetryCount > config.MaxAutoRetryCount {
		settings.AutoRetryCount = defaults.AutoRetryCount
	}
	if settings.RetryDelaySeconds < config.MinRetryDelaySeconds || settings.RetryDelaySeconds > config.MaxRetryDelaySeconds {
		settings.RetryDelaySeconds = defaults.RetryDelaySeconds
	}
	if settings.ConnectionTimeoutSeconds < config.MinConnectionTimeoutSeconds || settings.ConnectionTimeoutSeconds > config.MaxConnectionTimeoutSeconds {
		settings.ConnectionTimeoutSeconds = defaults.ConnectionTimeoutSeconds
	}
	return settings
}

func settingsNumber(label string, value, min, max int, invalid string) platform.SettingsDialogNumber {
	return platform.SettingsDialogNumber{
		Label:       label,
		Value:       value,
		Min:         min,
		Max:         max,
		InvalidText: invalid,
	}
}

func (a *app) openSettings() {
	if a.connectionBusy {
		return
	}

	settings := normalizeSettingsForPrompt(a.settings)
	language := a.languageCode()
	appearance := appearanceText(language)
	conflictOptions := []string{
		a.tr("settings.skip_existing"),
		a.tr("settings.overwrite"),
		a.tr("settings.overwrite") + " + " + a.tr("settings.backup_title"),
	}

	parallelLabel := a.tr("settings.parallel")
	timeoutLabel := a.tr("settings.timeout")
	retriesLabel := a.tr("settings.retries")
	retryDelayLabel := a.tr("settings.retry_delay")
	result, ok := platform.SettingsDialog(platform.SettingsDialogConfig{
		Title:             a.tr("settings.title"),
		Heading:           brand.ProductName,
		Intro:             a.tr("settings.title") + " · FTP • FTPS • SFTP",
		AppearanceLabel:   appearance.Title,
		AppearanceOptions: []string{appearance.Dark, appearance.Light},
		AppearanceIndex:   appearanceIndex(settings.Appearance),
		Numbers: []platform.SettingsDialogNumber{
			settingsNumber(parallelLabel, settings.Parallelism, config.MinParallelism, config.MaxParallelism, parallelLabel+" "+a.tr("settings.enter_range", config.MinParallelism, config.MaxParallelism)),
			settingsNumber(timeoutLabel, settings.ConnectionTimeoutSeconds, config.MinConnectionTimeoutSeconds, config.MaxConnectionTimeoutSeconds, timeoutLabel+" "+a.tr("settings.enter_range", config.MinConnectionTimeoutSeconds, config.MaxConnectionTimeoutSeconds)),
			settingsNumber(retriesLabel, settings.AutoRetryCount, config.MinAutoRetryCount, config.MaxAutoRetryCount, retriesLabel+" "+a.tr("settings.enter_range", config.MinAutoRetryCount, config.MaxAutoRetryCount)),
			settingsNumber(retryDelayLabel, settings.RetryDelaySeconds, config.MinRetryDelaySeconds, config.MaxRetryDelaySeconds, retryDelayLabel+" "+a.tr("settings.enter_range", config.MinRetryDelaySeconds, config.MaxRetryDelaySeconds)),
		},
		ConflictLabel:   a.tr("settings.skip_title"),
		ConflictOptions: conflictOptions,
		ConflictIndex:   conflictPolicyIndex(settings),
		ConfirmDelete:   a.tr("settings.confirm_delete_title"),
		ConfirmDeleteOn: settings.ConfirmDelete,
		Footer:          appearance.Hint,
		ApplyLabel:      okLabel(language),
		CancelLabel:     a.tr("common.cancel"),
	})
	if !ok || len(result.Numbers) != 4 {
		return
	}

	applyAppearanceSelection(&settings, result.AppearanceIndex)
	settings.Parallelism = result.Numbers[0]
	settings.ConnectionTimeoutSeconds = result.Numbers[1]
	settings.AutoRetryCount = result.Numbers[2]
	settings.RetryDelaySeconds = result.Numbers[3]
	applyConflictPolicySelection(&settings, result.ConflictIndex)
	settings.ConfirmDelete = result.ConfirmDelete

	title := a.tr("settings.title")
	a.setSettingsControlsEnabled(false)
	a.goSafe(func() {
		saved, err := a.engine.SetSettings(settings)
		a.dispatch(func() {
			a.setSettingsControlsEnabled(true)
			if err != nil {
				platform.ErrorDialog(title, a.tr("settings.save_failed"), a.userMessage(err, "settings.save_failed_body"))
				return
			}
			displayedLanguage := a.languageCode()
			a.settings = saved
			if i18n.Normalize(saved.Language) != displayedLanguage {
				a.applyLanguage(saved.Language)
			}
			status := a.tr("settings.saved", saved.Parallelism, saved.ConnectionTimeoutSeconds, retrySummary(a, saved), overwriteSummary(a, saved))
			if isDarkAppearance(saved.Appearance) != activeThemeIsDark() {
				status += " · " + appearanceText(saved.Language).Hint
			}
			a.setStatus(status)
			a.updateActionControls()
		})
	})
}

func (a *app) openAbout() {
	// About is the only user-facing surface that carries author/publisher identity.
	// All generic runtime, package and support metadata remains Ghost FTP-only.
	aboutBody := strings.ReplaceAll(a.tr("about.body", brand.Website, aboutSupport), "GhostFTP", brand.ProductName)
	body := aboutBody + "\n\n" +
		aboutPublisher + " · " + aboutAuthorWebsite + "\n" +
		"FTP • FTPS • SFTP  ·  " + brand.ProductName + " " + a.version
	platform.InfoCardDialog(
		brand.ProductName+" — "+a.tr("about.title"),
		a.tr("about.heading"),
		body,
		okLabel(a.languageCode()),
	)
}
